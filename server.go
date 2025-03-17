package brevx

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/crholm/brevx/responses"
	"io"
	"log/slog"
	"net"
	"net/mail"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Server listens for SMTP clients on the port specified in its config
type Server struct {
	Logger *slog.Logger

	// Middlewares will be run in the order they are specified before backend is called
	// m1 -> m2 -> Handler
	//               V
	// m1 <- m2 <- Handler
	Middlewares []Middleware

	// Handler will be receiving envelopes after the Data command
	Handler Handler

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
	// TODO: Implment this
	Timeout int

	// MaxClients controls how many maximum clients we can handle at once.
	// Defaults to defaultMaxClients
	// TODO: Implment this
	MaxClients int

	// XClientOn when using a proxy such as Nginx, XCLIENT command is used to pass the
	// original connection's IP address & connection's HELO
	XClientOn bool
	ProxyOn   bool

	// MaxRecipients is the maximum number of recipients allowed in a single RCPT command
	// Defaults to defaultMaxRecipients = 100
	MaxRecipients int

	// MaxUnrecognizedCommands is the maximum number of unrecognized commands allowed before the server terminates
	// the connection, defaults to defaultMaxUnrecognizedCommands = 5
	MaxUnrecognizedCommands int

	listener         net.Listener
	closedListener   chan struct{}
	wgConnections    sync.WaitGroup
	countConnections atomic.Int64

	state int
}

func (s *Server) Use(middleware ...Middleware) {
	s.Middlewares = append(s.Middlewares, middleware...)
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

	if c.MaxRecipients == 0 {
		c.MaxRecipients = defaultMaxRecipients
	}

	if c.MaxUnrecognizedCommands == 0 {
		c.MaxUnrecognizedCommands = defaultMaxUnrecognizedCommands
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
	cmdHELO     command = "HELO"
	cmdEHLO     command = "EHLO"
	cmdHELP     command = "HELP"
	cmdXCLIENT  command = "XCLIENT"
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

var commands = []command{cmdHELO, cmdEHLO, cmdXCLIENT, cmdMAIL, cmdRCPT, cmdRSET, cmdVRFY, cmdNOOP, cmdQUIT, cmdDATA, cmdSTARTTLS, cmdPROXY, cmdHELP}

func (c command) match(cmd string) bool {
	return strings.HasPrefix(strings.ToUpper(cmd), string(c))
}

func (c command) content(in string) string {
	return strings.TrimSpace(in[len(c):]) // since we accept mixed cases here...
}

// ListenAndServe begin accepting SMTP clients. Will block unless there is an error or Server.Shutdown() is called
func (s *Server) ListenAndServe() error {
	err := s.setDefaults()
	if err != nil {
		return err
	}

	log := s.log().With("inf", s.Addr)

	var connectionId uint64

	s.listener, err = net.Listen("tcp", s.Addr)
	if err != nil {
		s.state = ServerStateStartError
		return fmt.Errorf("cannot listen on %s, err %w ", s.Addr, err)
	}

	log.Info("Listening on TCP")
	s.state = ServerStateRunning

	for {
		log.Debug("Waiting for a new connection", "next_id", connectionId+1)
		conn, err := s.listener.Accept()
		connectionId++
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

			s.handleConn(newConnection(conn, s.MaxSize, clientID, s.Logger))

		}(conn, connectionId)
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

	conn.log.Info("Handle connection")
	defer conn.log.Info("Close connection")

	// Initial greeting
	greeting := fmt.Sprintf("220 %s SMTP %s(%s) #%d  %s",
		s.Hostname, Name, Version, conn.ID, time.Now().Format(time.RFC3339))

	helo := fmt.Sprintf("250 %s Hello", s.Hostname)
	// ehlo is a multi-line reply and need additional \r\n at the end
	ehlo := fmt.Sprintf("250-%s Hello\r\n", s.Hostname)

	// Extended feature advertisements
	messageSize := fmt.Sprintf("250-SIZE %d\r\n", s.MaxSize)
	extPipelining := "250-PIPELINING\r\n"
	extTLS := "250-STARTTLS\r\n"
	extEnhancedStatusCodes := "250-ENHANCEDSTATUSCODES\r\n"
	extUFF8 := "250-SMTPUTF8\r\n"
	// The last line doesn't need \r\n since string will be printed as a new line.
	// Also, Last line has no dash -
	help := "250 HELP"

	if s.TLSAlwaysOn && s.TLSConfig != nil {
		if err := conn.upgradeTLS(s.TLSConfig); err == nil {
			extTLS = ""
		} else {
			conn.log.Warn("Failed TLS handshake", "err", err)
			// Server requires TLS, but can't handshake
			conn.kill()
			return
		}
	}
	if s.TLSConfig == nil {
		// STARTTLS turned off, don't advertise it
		extTLS = ""
	}

	for conn.isAlive() {
		if conn.bufErr != nil {
			conn.log.Debug("connection could not buffer a response", "err", conn.bufErr)
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
			conn.log.Debug("Client: " + cmd)
			if err == io.EOF {
				conn.log.Warn("Client closed the connection", "err", err)
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				conn.log.Warn("Timeout", "err", err)
				return
			}
			if errors.Is(err, LimitError) {
				conn.sendResponse(responses.FailLineTooLong)
				conn.kill()
				return
			}
			if err != nil {
				conn.log.Warn("Could not read command", "err", err)
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
				conn.resetTransaction()

				content := cmdHELO.content(cmd)
				conn.Helo = strings.TrimSpace(content) // TODO parse domain or IP address

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

				conn.resetTransaction()

				content := cmdHELO.content(cmd)
				conn.Helo = content
				conn.ESMTP = true

				conn.sendResponse(ehlo,
					messageSize,
					extPipelining,
					extTLS,
					extEnhancedStatusCodes,
					extUFF8,
					help)
				continue

			case cmdHELP.match(cmd):
				// Client: HELP
				// Server: 214-Supported commands:
				// Server: 214-HELO EHLO MAIL RCPT DATA RSET NOOP QUIT VRFY EXPN HELP
				// Server: 214 End of HELP info
				coms := make([]string, 0, len(commands))
				for _, c := range commands {
					c := string(c)
					c, _, _ = strings.Cut(c, " ")
					coms = append(coms, c)
				}

				conn.sendResponse(
					"214-Supported commands:\r\n",
					fmt.Sprintf("214-%s\r\n", strings.Join(coms, " ")),
					"214 End of HELP info")
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
				conn.sendResponse(responses.SuccessMailCmd)
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
				conn.log.Debug("PROXY", "command", content)

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
					conn.log.Debug("PROXY, parse error, expected 5 or 6 parts", "data", content)
					conn.sendResponse(responses.FailSyntaxError)
					continue
				}

			case cmdMAIL.match(cmd):
				// Client: MAIL FROM:<sender@example.com>
				// This is the SMTP command that specifies the sender's email address.
				// Used for the Return-Path header
				if conn.isInTransaction() {
					conn.sendResponse(responses.FailNestedMailCmd)
					conn.errors++
					continue
				}
				content := cmdMAIL.content(cmd)
				addr, charser, _ := strings.Cut(content, " ")
				if charser == CharsetUtf8 {
					conn.charset = CharsetUtf8
					conn.Envelope.UTF8 = true
				}
				conn.MailFrom, err = mail.ParseAddress(addr)
				if err != nil {
					conn.log.Debug("MAIL, parse error", "data", "["+string(content)+"]", "err", err)
					conn.sendResponse(responses.RejectedSenderMailCmd)
					conn.errors++
					continue
				}

				// TODO CMD add hook system

				conn.sendResponse(responses.SuccessMailCmd)
				continue

			case cmdRCPT.match(cmd):
				// Client: RCPT TO:<recipient@example.com>
				// This is the SMTP command that specifies the recipient's email address.
				if len(conn.RcptTo) > s.MaxRecipients {
					conn.sendResponse(responses.ErrorTooManyRecipients)
					conn.errors++
					continue
				}
				content := cmdRCPT.content(cmd)
				to, err := mail.ParseAddress(content)
				if err != nil {
					conn.log.Debug("RCPT, parse error", "data", content, "err", err)
					conn.sendResponse(responses.FailSyntaxError)
					conn.errors++
					continue
				}

				// TODO CMD add hook system

				conn.RcptTo = append(conn.RcptTo, to)
				conn.sendResponse(responses.SuccessRcptCmd)
				continue

			case cmdRSET.match(cmd):
				// Client: RSET
				// The client then decides to abort this transaction and sends the RSET command.
				conn.resetTransaction()
				conn.sendResponse(responses.SuccessResetCmd)
				continue

			case cmdVRFY.match(cmd):
				// Client: VRFY user@example.com
				//The SMTP VRFY command is designed to verify the existence of a mailbox or user on a mail serve
				//Due to these security concerns, most modern SMTP servers have disabled or severely restricted the VRFY command.
				conn.sendResponse(responses.SuccessVerifyCmd)
				continue

			case cmdNOOP.match(cmd):
				// Client: NOOP
				// Its primary purpose is to:
				// - Check if the server is still alive and responsive.
				// - Keep the connection alive during periods of inactivity.
				// - Test the server's response.
				conn.sendResponse(responses.SuccessNoopCmd)
				continue

			case cmdQUIT.match(cmd):
				// Client: QUIT
				// The QUIT command is the standard way for an SMTP client to gracefully close a connection to an SMTP server.
				conn.sendResponse(responses.SuccessQuitCmd)
				conn.kill()
				return

			case cmdDATA.match(cmd):
				// Client: DATA
				// The DATA command in SMTP is used to initiate the transfer of the email message content,
				if len(conn.RcptTo) == 0 {
					conn.sendResponse(responses.FailNoRecipientsDataCmd)
					break
				}
				conn.sendResponse(responses.SuccessDataCmd)
				conn.state = ConnData

			case cmdSTARTTLS.match(cmd):
				// Client: STARTTLS
				// Server: 220 2.0.0 Ready to start TLS
				// [TLS handshake occurs here]

				if s.TLSConfig == nil {
					conn.sendResponse(responses.FailCommandNotImplemented)
					continue
				}

				conn.sendResponse(responses.SuccessStartTLSCmd)
				conn.state = ConnStartTLS
				continue
			default:
				conn.errors++
				if conn.errors >= s.MaxUnrecognizedCommands {
					conn.sendResponse(responses.FailMaxUnrecognizedCmd)
					conn.kill()
				} else {
					conn.sendResponse(responses.FailUnrecognizedCmd)
				}
			}

		case ConnData:

			_, err := conn.Envelope.Data.ReadFrom(conn.in.DotReader())

			if errors.Is(err, LimitError) {
				conn.log.Debug("DATA, to much data sent", "err", err)
				conn.sendResponse(responses.FailMessageSizeExceeded, " ", LimitError.Error())
				conn.kill()
				continue
			}

			if err != nil {
				conn.log.Warn("DATA, error reading data", "err", err)
				conn.sendResponse(responses.FailReadErrorDataCmd, " ", err.Error())
				conn.kill()
				continue
			}

			middlewares := append([]Middleware{}, s.Middlewares...)
			slices.Reverse(middlewares)

			var start HandlerFunc = s.Handler.Data
			for _, middleware := range middlewares {
				if middleware == nil {
					continue
				}
				start = middleware(start)
			}
			resp := start(conn.Envelope)

			if resp == nil {
				resp = responses.SuccessMessageAccepted
			}
			if resp.Class() != responses.ClassSuccess { // indicates that we should abort
				conn.log.Debug("DATA, processing failed", "response", resp.String())
				conn.sendResponse(resp)
				conn.errors++
				continue
			}
			if resp.Class() == responses.ClassSuccess {
				conn.messagesSent++
			}

			conn.sendResponse(resp)

			conn.state = ConnCmd
			if s.isShuttingDown() {
				conn.state = ConnShutdown
			}
			conn.resetTransaction()

			continue

		case ConnStartTLS:
			if conn.TLS { // already in tls mode...
				conn.log.Warn("TLS, Failed TLS start, tls is alreade active")
				conn.state = ConnCmd
				continue
			}

			if s.TLSConfig == nil { // no tls config available
				conn.log.Warn("TLS, Failed TLS start, no tls config")
				conn.state = ConnCmd
				continue
			}

			err := conn.upgradeTLS(s.TLSConfig)
			if err != nil {
				conn.log.Warn("TLS, Failed TLS handshake", "err", err)
				conn.state = ConnCmd
				continue
			}
			extTLS = ""
			conn.resetTransaction()
			conn.state = ConnCmd
			continue

		case ConnShutdown:
			// shutdown state
			conn.sendResponse(responses.ErrorShutdown)
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
