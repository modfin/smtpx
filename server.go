package guerrilla

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/mail"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	Logger  *slog.Logger
	Backend Backend

	// TLSConfig will be used when TLS is enabled
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
}

func (c *Server) setDefaults() error {
	if c.Logger == nil {
		c.Logger = noopLogger()
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

	if c.closedListener == nil {
		c.closedListener = make(chan struct{})
	}

	return nil
}

type allowedHosts struct {
	table      map[string]bool // host lookup table
	wildcards  []string        // host wildcard list (* is used as a wildcard)
	sync.Mutex                 // guard access to the map
}

type command string

var (
	cmdHELO     command = "HELO "
	cmdEHLO     command = "EHLO"
	cmdHELP     command = "HELP"
	cmdXCLIENT  command = "XCLIENT "
	cmdMAIL     command = "MAIL FROM:"
	cmdRCPT     command = "RCPT TO:"
	cmdRSET     command = "RSET"
	cmdVRFY     command = "VRFY"
	cmdNOOP     command = "NOOP"
	cmdQUIT     command = "QUIT"
	cmdDATA     command = "DATA"
	cmdSTARTTLS command = "STARTTLS"
	cmdPROXY    command = "PROXY "
)

func (c command) match(cmd string) bool {
	return strings.HasPrefix(strings.ToUpper(cmd), string(c))
}

func (c command) content(in string) string {
	return in[len(c):] // since we accept mixed cases here...
}

// ListenAndServe begin accepting SMTP clients. Will block unless there is an error or Server.Shutdown() is called
func (s *Server) ListenAndServe() error {
	err := s.setDefaults()
	if err != nil {
		return err
	}

	log := s.log().With("inf", s.Addr)

	var clientID uint64

	s.listener, err = net.Listen("tcp", s.Addr)
	if err != nil {
		s.state = ServerStateStartError
		return fmt.Errorf("cannot listen on %s, err %w ", s.Addr, err)
	}

	log.Info("Listening on TCP")
	s.state = ServerStateRunning

	for {
		log.Debug("Waiting for a new connection", "next_id", clientID+1)
		conn, err := s.listener.Accept()
		clientID++
		if err != nil {
			log.Info("Server has stopped accepting new clients", "connections", s.countConnections.Load())
			s.state = ServerStateStopped
			// wait for all connections to finish, this might be dangerous and a deadline should be set
			s.wgConnections.Wait() // wait for all connections to finish
			close(s.closedListener)
			return nil
		}

		log.Debug("Accepted new connection", "ip", conn.RemoteAddr())

		s.wgConnections.Add(1)
		s.countConnections.Add(1)
		go func(conn net.Conn, clientID uint64) {
			defer s.wgConnections.Done()
			defer s.countConnections.Add(-1)
			defer conn.Close()
			s.handleConn(newConnection(conn, clientID, s.Logger))

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
	log := s.log().With("id", conn.ClientId(), "ip", conn.RemoteAddr)

	log.Info("Handle connection")
	defer log.Info("Close connection")

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
			log.Warn("Failed TLS handshake", "err", err)
			// Server requires TLS, but can't handshake
			conn.kill()
			return
		}
	}
	if s.TLSConfig == nil {
		// STARTTLS turned off, don't advertise it
		advertiseTLS = ""
	}

	for conn.isAlive() {
		if conn.bufErr != nil {
			log.Debug("connection could not buffer a response", "err", conn.bufErr)
			return
		}

		switch conn.state {
		case ConnGreeting:
			conn.sendResponse(greeting)
			conn.state = ConnCmd
			continue

		case ConnCmd:
			// TODO set readlimit ... // TODO avoid DoS
			cmd, err := conn.readCommand()
			log.Debug("Client: " + cmd)
			if err == io.EOF {
				log.Warn("Client closed the connection", "err", err)
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Warn("Timeout", "err", err)
				return
			}
			if errors.Is(err, LineLimitExceeded) {
				conn.sendResponse(response.FailLineTooLong)
				conn.kill()
				return
			}
			if err != nil {
				log.Warn("Read error", "err", err)
				conn.kill()
				return
			}
			if s.isShuttingDown() {
				conn.state = ConnShutdown
				continue
			}

			switch {
			case cmdHELO.match(cmd):
				// Client: HELO example.com
				// The client sends the HELO command, followed by its own fully qualified domain name (FQDN) or IP address.
				// HELO is the older "Hello" command, used in basic SMTP sessions
				//  (as opposed to the extended ESMTP sessions initiated by EHLO).
				//
				// helo = "HELO" SP Domain CRLF
				// HELO example.com\r\n
				// HELO 192.168.1.10\r\n
				content := cmdHELO.content(cmd)
				conn.Helo = strings.TrimSpace(string(content)) // TODO parse domain or IP address

				conn.resetTransaction()
				conn.sendResponse(helo)
				continue

			case cmdEHLO.match(cmd):
				// Client: EHLO example.com
				// The client sends the EHLO command, followed by its own fully qualified domain name (FQDN) or IP address.
				// Client is saying "Hello, I am example.com, and I would like to establish an ESMTP connection."
				//
				// ehlo = "EHLO" SP ( Domain / address-literal ) CRLF
				//  - SP is a single space
				//  - Domain: A fully qualified domain name (FQDN).
				//  - address-literal: An IP address enclosed in square brackets
				// EHLO mail.example.com\r\n
				// EHLO [192.168.1.10]\r\n
				// EHLO [IPv6:2001:0db8:85a3:0000:0000:8a2e:0370:7334]\r\n
				content := cmdHELO.content(cmd)
				conn.Helo = strings.TrimSpace(string(content))

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

			case cmdXCLIENT.match(cmd) && s.XClientOn:
				// Client: XCLIENT ADDR=192.168.1.10 NAME=client.example.com PROTO=ESMTP AUTH=user@example.com
				// The XCLIENT command is another Extended SMTP (ESMTP) command, but it's not standardized in the
				// official RFCs. It's used by some mail servers, primarily Postfix, to provide client information to
				// the server before the MAIL FROM command. This is particularly useful in situations where a proxy or
				// load balancer is involved.
				content := cmdXCLIENT.content(cmd)
				toks := strings.Fields(content)
				for _, tok := range toks {
					key, val, found := strings.Cut(tok, "=")
					if found {
						if val == "[UNAVAILABLE]" {
							continue
						}
						if key == "ADDR" {
							ip := net.ParseIP(string(val))
							conn.RemoteAddr = &net.TCPAddr{IP: ip}
						}
						if key == "HELO" {
							conn.Helo = val
						}
					}
				}
				conn.sendResponse(response.SuccessMailCmd)
				continue

			case cmdPROXY.match(cmd) && s.ProxyOn:
				// Client: PROXY TCP4 remote.host.example.com 192.168.1.10 192.168.1.20 5000 6000
				// PROXY
				// - TCP4: Protocol version.
				// - remote.host.example.com: The hostname of the connecting client.
				// - 192.168.1.10: The client's IP address.
				// - 192.168.1.20: The proxy's IP address.
				// - 5000: The client's source port.
				// - 6000: The proxy's destination port.
				content := strings.TrimSpace(cmdPROXY.content(cmd))
				toks := strings.Fields(content)
				log.Debug("PROXY", "command", content)

				switch len(toks) {
				case 5:
					ip := net.ParseIP(toks[1])
					conn.RemoteAddr = &net.TCPAddr{IP: ip}
					conn.sendResponse(greeting)
					continue
				case 6:
					ip := net.ParseIP(toks[2])
					conn.RemoteAddr = &net.TCPAddr{IP: ip}
					conn.sendResponse(greeting)
					continue
				default:
					log.Error("PROXY parse error, expected 5 or 6 parts", "data", content)
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
				conn.MailFrom, err = mail.ParseAddress(string(content))
				if err != nil {
					log.Error("MAIL parse error", "data", "["+string(content)+"]", "err", err)
					conn.sendResponse(err)
					continue
				}

				// Hook to Backend to check if it alloed
				err = s.Backend.Mail(conn.Envelope, conn.MailFrom)
				if err != nil { // indicates that we should abort
					log.Error("MAIL hook error", "data", "["+string(content)+"]", "err", err)
					conn.sendResponse(response.RejectedSenderMailCmd)
					conn.kill()
					continue
				}

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
				to, err := mail.ParseAddress(string(content))
				if err != nil {
					log.Error("RCPT parse error", "data", "["+string(content)+"]", "err", err)
					conn.sendResponse(response.FailSyntaxError)
					break
				}

				// Hook to Backend to check if ut i is allowed
				err = s.Backend.Rcpt(conn.Envelope, to)
				if err != nil { // indicates that we should abort
					log.Error("MAIL hook error", "data", "["+string(content)+"]", "err", err)
					conn.sendResponse(response.RejectedRcptCmd)
					conn.kill()
					continue
				}

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
			// TODO make a limit to readers...
			//conn.bufin.setLimit(s.MaxSize + 1024000) // This a hard limit.

			n, err := conn.Data.ReadFrom(conn.in.DotReader())
			if n > s.MaxSize {
				err = fmt.Errorf("maximum DATA size exceeded (%d)", s.MaxSize)
			}
			if err != nil {
				log.Error("error reading data", "err", err)
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
				log.Warn("Error reading data", "err", err)
				conn.resetTransaction()
				continue
			}

			res := s.Backend.Process(conn.Envelope)
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
				log.Warn("Failed TLS start, tls is alreade active")
				conn.state = ConnCmd
				continue
			}

			if s.TLSConfig == nil { // no tls config available
				log.Warn("Failed TLS start, no tls config")
				conn.state = ConnCmd
				continue
			}

			err := conn.upgradeTLS(s.TLSConfig)
			if err != nil {
				log.Warn("Failed TLS handshake", "err", err)
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

	}
}

func (s *Server) log() *slog.Logger {
	if s.Logger == nil {
		return noopLogger()
	}
	return s.Logger
}
