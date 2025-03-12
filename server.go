package guerrilla

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/phires/go-guerrilla/mail"
	"github.com/phires/go-guerrilla/mail/rfc5321"
	"github.com/phires/go-guerrilla/response"
)

const (
	CommandVerbMaxLength = 16
	CommandLineMaxLength = 1024

	// Number of allowed unrecognized commands before we terminate the connection
	MaxUnrecognizedCommands = 5
)

const (
	// Server has just been created
	ServerStateNew = iota
	// Server has just been stopped
	ServerStateStopped
	// Server has been started and is running
	ServerStateRunning
	// Server could not start due to an error
	ServerStateStartError
)

const defaultMaxClients = 100
const defaultTimeout = 30
const defaultInterface = ":2525"
const defaultMaxSize = 10_485_760 // int64(10 << 20) // 10 Megabytes

// Server listens for SMTP clients on the port specified in its config
type Server struct {
	TLSConfig *tls.Config
	// AlwaysOn run this Server as a pure TLS Server, i.e. SMTPS
	TLSAlwaysOn bool

	// Hostname will be used in the Server's reply to HELO/EHLO. If TLS enabled
	// make sure that the Hostname matches the cert. Defaults to os.Hostname()
	// Hostname will also be used to fill the 'Host' property when the "RCPT TO" address is
	// addressed to just <postmaster>
	Hostname string

	// Addr is the interface specified in <ip>:<port> - defaults to ":25"
	Addr string

	// MaxSize is the maximum size of an email that will be accepted for delivery.
	// Defaults to 10 Mebibytes
	MaxSize int64
	// Timeout specifies the connection timeout in seconds. Defaults to 30
	Timeout int
	// MaxClients controls how many maximum clients we can handle at once.
	// Defaults to defaultMaxClients
	MaxClients int

	// XClientOn when using a proxy such as Nginx, XCLIENT command is used to pass the
	// original connection's IP address & connection's HELO
	XClientOn bool
	ProxyOn   bool

	listener         net.Listener
	closedListener   chan struct{}
	wgConnections    sync.WaitGroup
	countConnections atomic.Int64

	state int

	backend Backend
	logger  *slog.Logger
}

func (c *Server) setDefaults() error {
	if c.logger == nil {
		c.logger = noopLogger()
	}

	if c.Addr == "" {
		c.Addr = defaultInterface
	}
	if c.Hostname == "" {
		h, err := os.Hostname()
		if err != nil {
			return err
		}
		c.Hostname = h
	}
	if c.MaxClients == 0 {
		c.MaxClients = defaultMaxClients
	}
	if c.Timeout == 0 {
		c.Timeout = defaultTimeout
	}
	if c.MaxSize == 0 {
		c.MaxSize = defaultMaxSize // 10 Mebibytes
	}

	return nil
}

type allowedHosts struct {
	table      map[string]bool // host lookup table
	wildcards  []string        // host wildcard list (* is used as a wildcard)
	sync.Mutex                 // guard access to the map
}

type command []byte

var (
	cmdHELO     command = []byte("HELO")
	cmdEHLO     command = []byte("EHLO")
	cmdHELP     command = []byte("HELP")
	cmdXCLIENT  command = []byte("XCLIENT ")
	cmdMAIL     command = []byte("MAIL FROM:")
	cmdRCPT     command = []byte("RCPT TO:")
	cmdRSET     command = []byte("RSET")
	cmdVRFY     command = []byte("VRFY")
	cmdNOOP     command = []byte("NOOP")
	cmdQUIT     command = []byte("QUIT")
	cmdDATA     command = []byte("DATA")
	cmdSTARTTLS command = []byte("STARTTLS")
	cmdPROXY    command = []byte("PROXY ")
)

func (c command) match(in []byte) bool {
	return bytes.HasPrefix(in, c)
}

func (c command) content(in []byte) []byte {
	return bytes.TrimPrefix(in, c)
}

// ListenAndServe begin accepting SMTP clients. Will block unless there is an error or Server.Shutdown() is called
func (s *Server) ListenAndServe() error {

	var clientID uint64
	var err error

	s.listener, err = net.Listen("tcp", s.Addr)
	if err != nil {
		s.state = ServerStateStartError
		return fmt.Errorf("cannot listen on %s, err %w ", s.Addr, err)
	}

	s.log().Info("Listening on TCP", "interface", s.Addr)
	s.state = ServerStateRunning

	for {
		s.log().Debug("Waiting for a new connection", "next_client_id", clientID+1, "interface", s.Addr)
		conn, err := s.listener.Accept()
		clientID++
		if err != nil {
			// TODO error my be temporary?
			s.log().With("interface", s.Addr).Info("Server has stopped accepting new clients")
			s.state = ServerStateStopped
			close(s.closedListener)
			return nil
		}

		s.wgConnections.Add(1)
		s.countConnections.Add(1)
		go func(conn net.Conn, clientID uint64) {
			defer s.wgConnections.Done()
			defer s.countConnections.Add(-1)
			defer conn.Close()

			s.handleConn(newConnection(conn, clientID, s.logger))
		}(conn, clientID)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.listener != nil {
		// This will cause Start function to return, by causing an error on listener.Accept
		_ = s.listener.Close()
		// wait for the listener to listener.Accept
		select {
		case <-ctx.Done():
			return fmt.Errorf("context done, %w", ctx.Err())
		case <-s.closedListener:
			return nil
		}
	}
	return nil
}

func (s *Server) GetActiveClientsCount() int {
	return int(s.countConnections.Load())
}

const commandSuffix = "\r\n"

// Reads from the connection until a \n terminator is encountered,
// or until a timeout occurs.
func (s *Server) readCommand(conn *connection) ([]byte, error) {
	//var input string
	var err error
	var bs []byte
	// In command state, stop reading at line breaks
	bs, err = conn.bufin.ReadSlice('\n')
	if err != nil {
		return bs, err
	} else if bytes.HasSuffix(bs, []byte(commandSuffix)) {
		return bs[:len(bs)-2], err
	}
	return bs[:len(bs)-1], err
}

// flushResponse a response to the connection. Flushes the connection.bufout buffer to the connection
func (s *Server) flushResponse(conn *connection) error {
	if err := conn.setTimeout(time.Duration(s.Timeout) * time.Second); err != nil {
		return err
	}
	return conn.bufout.Flush()
}

func (s *Server) isShuttingDown() bool {
	select {
	case <-s.closedListener:
		return true
	default:
		return false
	}
}

// Handles an entire connection SMTP exchange
func (s *Server) handleConn(conn *connection) {
	defer conn.closeConn()
	s.log().Info("Handle connection", "ip", conn.RemoteAddr, "id", conn.ClientId)

	// Initial greeting
	greeting := fmt.Sprintf("220 %s SMTP %s(%s) #%d  %s",
		s.Hostname, Name, Version, conn.ID, time.Now().Format(time.RFC3339))

	helo := fmt.Sprintf("250 %s Hello", s.Hostname)
	// ehlo is a multi-line reply and need additional \r\n at the end
	ehlo := fmt.Sprintf("250-%s Hello\r\n", s.Hostname)

	// Extended feature advertisements
	messageSize := fmt.Sprintf("250-SIZE %d\r\n", s.MaxSize)
	pipelining := "250-PIPELINING\r\n"
	advertiseTLS := "250-STARTTLS\r\n"
	advertiseEnhancedStatusCodes := "250-ENHANCEDSTATUSCODES\r\n"
	// The last line doesn't need \r\n since string will be printed as a new line.
	// Also, Last line has no dash -
	help := "250 HELP"

	if s.TLSAlwaysOn && s.TLSConfig != nil {
		if err := conn.upgradeTLS(s.TLSConfig); err == nil {
			advertiseTLS = ""
		} else {
			s.log().Warn("Failed TLS handshake", "ip", conn.RemoteAddr, "err", err)
			// Server requires TLS, but can't handshake
			conn.kill()
			// TODO just return ?
		}
	}
	if s.TLSConfig == nil {
		// STARTTLS turned off, don't advertise it
		advertiseTLS = ""
	}
	for conn.isAlive() {
		switch conn.state {
		case ConnGreeting:
			conn.sendResponse(greeting)
			conn.state = ConnCmd
		case ConnCmd:
			conn.bufin.setLimit(CommandLineMaxLength)
			input, err := s.readCommand(conn)
			s.log().Debug("Client sent:", "command", input)
			if err == io.EOF {
				s.log().Warn("Client closed the connection", "ip", conn.RemoteAddr, "err", err)
				return
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				s.log().Warn("Timeout", "ip", conn.RemoteAddr, "err", err)
				return
			} else if err == LineLimitExceeded {
				conn.sendResponse(response.FailLineTooLong)
				conn.kill()
				break
			} else if err != nil {
				s.log().Warn("Read error", "ip", conn.RemoteAddr, "err", err)
				conn.kill()
				break
			}
			if s.isShuttingDown() {
				conn.state = ConnShutdown
				continue
			}

			cmdLen := len(input)
			if cmdLen > CommandVerbMaxLength {
				cmdLen = CommandVerbMaxLength
			}
			cmd := bytes.ToUpper(input[:cmdLen])
			switch {
			case cmdHELO.match(cmd):
				// Client: HELO example.com
				// The client sends the HELO command, followed by its own fully qualified domain name (FQDN) or IP address.
				// HELO is the older "Hello" command, used in basic SMTP sessions
				//  (as opposed to the extended ESMTP sessions initiated by EHLO).
				content := cmdHELO.content(cmd)
				if h, err := conn.parser.Helo(content); err == nil {
					conn.Helo = h
				} else {
					s.log().Warn("invalid helo", "helo", h, "connection", conn.ID)
					conn.sendResponse(response.FailSyntaxError)
					break
				}
				conn.resetTransaction()
				conn.sendResponse(helo)
				continue

			case cmdEHLO.match(cmd):
				// Client: EHLO example.com
				// The client sends the EHLO command, followed by its own fully qualified domain name (FQDN) or IP address.
				// Client is saying "Hello, I am example.com, and I would like to establish an ESMTP connection."
				content := cmdHELO.content(cmd)
				if h, _, err := conn.parser.Ehlo(content); err == nil {
					conn.Helo = h
				} else {
					s.log().Warn("invalid ehlo", "ehlo", h, "connection", conn.ID)
					conn.sendResponse(response.FailSyntaxError)
					break
				}
				conn.ESMTP = true
				conn.resetTransaction()
				conn.sendResponse(ehlo,
					messageSize,
					pipelining,
					advertiseTLS,
					advertiseEnhancedStatusCodes,
					help)
				continue

			case cmdHELP.match(cmd):
				quote := response.GetQuote()
				conn.sendResponse("214-OK\r\n", quote)
				continue

			case s.XClientOn && cmdXCLIENT.match(cmd):
				// Client: XCLIENT ADDR=192.168.1.10 NAME=client.example.com PROTO=ESMTP AUTH=user@example.com
				// The XCLIENT command is another Extended SMTP (ESMTP) command, but it's not standardized in the
				// official RFCs. It's used by some mail servers, primarily Postfix, to provide client information to
				// the server before the MAIL FROM command. This is particularly useful in situations where a proxy or
				// load balancer is involved.
				content := cmdXCLIENT.content(cmd)
				toks := bytes.Split(content, []byte{' '})
				for _, tok := range toks {
					key, val, found := bytes.Cut(tok, []byte{'='})
					if found {
						if bytes.Equal(val, []byte("[UNAVAILABLE]")) {
							continue
						}
						if bytes.Equal(key, []byte("ADDR")) {
							ip := net.ParseIP(string(val))
							conn.RemoteAddr = &net.TCPAddr{IP: ip}
						}
						if bytes.Equal(key, []byte("HELO")) {
							conn.Helo = string(val)
						}
					}
				}
				conn.sendResponse(response.SuccessMailCmd)
				continue

			case s.ProxyOn && cmdPROXY.match(cmd):
				// Client: PROXY TCP4 remote.host.example.com 192.168.1.10 192.168.1.20 5000 6000
				// PROXY
				// - TCP4: Protocol version.
				// - remote.host.example.com: The hostname of the connecting client.
				// - 192.168.1.10: The client's IP address.
				// - 192.168.1.20: The proxy's IP address.
				// - 5000: The client's source port.
				// - 6000: The proxy's destination port.
				content := bytes.TrimSpace(cmdPROXY.content(cmd))
				toks := bytes.Split(content, []byte{' '})
				s.log().Debug("PROXY", "command", content)

				switch len(toks) {
				case 5:
					ip := net.ParseIP(string(toks[1]))
					conn.RemoteAddr = &net.TCPAddr{IP: ip}
					conn.sendResponse(greeting)
					continue
				case 6:
					ip := net.ParseIP(string(toks[2]))
					conn.RemoteAddr = &net.TCPAddr{IP: ip}
					conn.sendResponse(greeting)
					continue
				default:
					s.log().Error("PROXY parse error, expected 5 or 6 parts", "data", "["+string(content)+"]")
					conn.sendResponse(response.FailSyntaxError)
					continue
				}

			case cmdMAIL.match(cmd):
				// Client: MAIL FROM:<sender@example.com>
				// This is the SMTP command that specifies the sender's email address.
				// Used for the Return-Path header
				if conn.isInTransaction() {
					conn.sendResponse(response.FailNestedMailCmd)
					continue
				}
				content := cmdMAIL.content(cmd)
				conn.MailFrom, err = conn.parsePath(content, conn.parser.MailFrom)
				if err != nil {
					s.log().Error("MAIL parse error", "data", "["+string(input[10:])+"]", "err", err)
					conn.sendResponse(err)
					continue
				}
				if conn.parser.NullPath {
					// bounce has empty from address
					conn.MailFrom = mail.Address{}
				}
				// TODO run Backend hook
				conn.sendResponse(response.SuccessMailCmd)
				continue

			case cmdRCPT.match(cmd):
				// Client: RCPT TO:<recipient@example.com>
				// This is the SMTP command that specifies the recipient's email address.
				if len(conn.RcptTo) > rfc5321.LimitRecipients {
					conn.sendResponse(response.ErrorTooManyRecipients)
					break
				}
				content := cmdRCPT.content(cmd)
				to, err := conn.parsePath(content, conn.parser.RcptTo)
				if err != nil {
					s.log().Error("RCPT parse error", "data", "["+string(input[8:])+"]", "err", err)
					conn.sendResponse(err.Error())
					break
				}

				// TODO run Backend hook
				conn.RcptTo = append(conn.RcptTo, to)
				conn.sendResponse(response.SuccessRcptCmd)
				continue

			case cmdRSET.match(cmd):
				// Client: RSET
				// The client then decides to abort this transaction and sends the RSET command.
				conn.resetTransaction()
				conn.sendResponse(response.SuccessResetCmd)
				continue

			case cmdVRFY.match(cmd):
				// Client: VRFY user@example.com
				//The SMTP VRFY command is designed to verify the existence of a mailbox or user on a mail serve
				//Due to these security concerns, most modern SMTP servers have disabled or severely restricted the VRFY command.
				conn.sendResponse(response.SuccessVerifyCmd)
				continue

			case cmdNOOP.match(cmd):
				// Client: NOOP
				// Its primary purpose is to:
				// - Check if the server is still alive and responsive.
				// - Keep the connection alive during periods of inactivity.
				// - Test the server's response.
				conn.sendResponse(response.SuccessNoopCmd)
				continue

			case cmdQUIT.match(cmd):
				// Client: QUIT
				// The QUIT command is the standard way for an SMTP client to gracefully close a connection to an SMTP server.
				conn.sendResponse(response.SuccessQuitCmd)
				conn.kill()
				return

			case cmdDATA.match(cmd):
				// Client: DATA
				// The DATA command in SMTP is used to initiate the transfer of the email message content,
				if len(conn.RcptTo) == 0 {
					conn.sendResponse(response.FailNoRecipientsDataCmd)
					break
				}
				conn.sendResponse(response.SuccessDataCmd)
				conn.state = ConnData

			case cmdSTARTTLS.match(cmd):
				// Client: STARTTLS
				// Server: 220 2.0.0 Ready to start TLS
				// [TLS handshake occurs here]

				if s.TLSConfig == nil {
					conn.sendResponse(response.FailCommandNotImplemented)
					continue
				}

				conn.sendResponse(response.SuccessStartTLSCmd)
				conn.state = ConnStartTLS
				continue
			default:
				conn.errors++
				if conn.errors >= MaxUnrecognizedCommands {
					conn.sendResponse(response.FailMaxUnrecognizedCmd)
					conn.kill()
				} else {
					conn.sendResponse(response.FailUnrecognizedCmd)
				}
			}

		case ConnData:

			// intentionally placed the limit 1MB above so that reading does not return with an error
			// if the connection goes a little over. Anything above will err
			conn.bufin.setLimit(s.MaxSize + 1024000) // This a hard limit.

			n, err := conn.Data.ReadFrom(conn.smtpReader.DotReader())
			if n > s.MaxSize {
				err = fmt.Errorf("maximum DATA size exceeded (%d)", s.MaxSize)
			}
			if err != nil {
				s.log().Error("error reading data", "err", err)
				if errors.Is(err, LineLimitExceeded) {
					conn.sendResponse(response.FailReadLimitExceededDataCmd, " ", LineLimitExceeded.Error())
					conn.kill()
				} else if errors.Is(err, MessageSizeExceeded) {
					conn.sendResponse(response.FailMessageSizeExceeded, " ", MessageSizeExceeded.Error())
					conn.kill()
				} else {
					conn.sendResponse(response.FailReadErrorDataCmd, " ", err.Error())
					conn.kill()
				}
				s.log().Warn("Error reading data", "ip", conn.RemoteAddr, "err", err)
				conn.resetTransaction()
				continue
			}

			res := s.backend.Process(conn.Envelope)
			if res.Class() == 2 {
				conn.messagesSent++
			}
			conn.sendResponse(res)
			conn.state = ConnCmd
			if s.isShuttingDown() {
				conn.state = ConnShutdown
			}
			conn.resetTransaction()
			continue

		case ConnStartTLS:
			if conn.TLS { // already in tls mode...
				s.log().Warn("Failed TLS start, tls is alreade active", "ip", conn.RemoteAddr)
				conn.state = ConnCmd
				continue
			}

			if s.TLSConfig == nil { // no tls config available
				s.log().Warn("Failed TLS start, no tls config", "ip", conn.RemoteAddr)
				conn.state = ConnCmd
				continue
			}

			err := conn.upgradeTLS(s.TLSConfig)
			if err != nil {
				s.log().Warn("Failed TLS handshake", "ip", conn.RemoteAddr, "err", err)
				conn.state = ConnCmd
				continue
			}
			advertiseTLS = ""
			conn.resetTransaction()
			conn.state = ConnCmd
			continue

		case ConnShutdown:
			// shutdown state
			conn.sendResponse(response.ErrorShutdown)
			conn.kill()
		}

		if conn.bufErr != nil {
			s.log().Debug("connection could not buffer a response", "err", conn.bufErr)
			return
		}
		// flush the response buffer
		if conn.bufout.Buffered() > 0 {

			s.log().Debug(fmt.Sprintf("Writing response to connection: %s", conn.response.String()))

			err := s.flushResponse(conn)
			if err != nil {
				s.log().Debug("error writing response", "err", err)
				return
			}
		}

	}
}

func (s *Server) log() *slog.Logger {
	if s.logger == nil {
		return noopLogger()
	}
	return s.logger
}
