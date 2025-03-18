package smtpx

import (
	"crypto/tls"
	"fmt"
	"github.com/modfin/smtpx/envelope"
	"log/slog"
	"net"
	"sync"
	"time"
)

// ClientState indicates which part of the SMTP transaction a given connection is in.
type ClientState int

const (
	// ConnGreeting The connection has connected, and is awaiting our first response
	ConnGreeting = iota
	// ConnCmd We have responded to the connection's connection and are awaiting a command
	ConnCmd
	// ConnData We have received the sender and recipient information
	ConnData
	// ConnStartTLS We have agreed with the connection to secure the connection over TLS
	ConnStartTLS
	// Server will shut down, connection to shutdown on next command turn
	ConnShutdown
)

const CharsetDefault = "default"
const CharsetUtf8 = "SMTPUTF8"

type connection struct {
	*envelope.Envelope

	ID uint64

	ConnectedAt time.Time
	KilledAt    time.Time

	// Number of errors encountered during session with this connection
	errors  int
	state   ClientState
	charset string

	messagesSent int

	bufErr error

	in *smtpReader

	connGuard sync.Mutex
	conn      net.Conn

	log *slog.Logger
}

// newConnection allocates a new connection.
func newConnection(conn net.Conn, maxMessageSize int64, connectionId uint64, logger *slog.Logger) *connection {

	env := envelope.NewEnvelope(conn.RemoteAddr(), connectionId)
	c := &connection{
		conn: conn,

		Envelope:    env,
		ConnectedAt: time.Now(),
		charset:     CharsetDefault,
		in:          NewSMTPReader(conn, maxMessageSize),
		log: logger.With(
			"connection-id", env.ConnectionId(),
			// This is change when a multiple emails are sent.
			// Will stack with regular slog implementation
			//"envelope-id", env.EnvelopeId(),

			"remote-ip", conn.RemoteAddr()),
	}
	return c
}

const commandSuffix = "\r\n"

// Reads from the connection until a \n terminator is encountered,
// or until a timeout occurs.
func (conn *connection) readCommand() (string, error) {
	//var input string
	// In command state, stop reading at line breaks
	// TODO, maybe enforce max len of line ( a smaller number then maxMessageSize)
	cmd, err := conn.in.ReadLine()
	if err != nil {
		return "", err
	}
	return cmd, nil
}

// sendResponse adds a response to be written on the next turn
// the response gets buffered
func (c *connection) sendResponse(r ...interface{}) {

	var out string
	if c.bufErr != nil {
		c.bufErr = nil
	}
	for _, item := range r {
		switch v := item.(type) {
		case error:
			out += v.Error()
		case fmt.Stringer:
			out += v.String()
		case string:
			out += v
		}
	}

	c.log.Debug(("Server: " + out))

	_, c.bufErr = c.conn.Write([]byte(out + commandSuffix))

	if c.bufErr != nil {
		c.log.Error("could not write to c.bufout", "err", c.bufErr)
		return
	}

}

// resetTransaction resets the SMTP transaction, ready for the next email (doesn't disconnect)
// Transaction ends on:
// -HELO/EHLO/REST command
// -End of DATA command
// TLS handshake
func (c *connection) resetTransaction() {
	c.Envelope = envelope.NewEnvelope(c.RemoteAddr, c.ConnectionId())
	c.in.ResetLimit()

	c.log.Debug("transaction reset")
}

// isInTransaction returns true if the connection is inside a transaction.
// A transaction starts after a MAIL command gets issued by the connection.
// Call resetTransaction to end the transaction
func (c *connection) isInTransaction() bool {
	return c.MailFrom != nil && c.MailFrom.Address != ""
}

// kill flags the connection to close on the next turn
func (c *connection) kill() {
	c.KilledAt = time.Now()
}

// isAlive returns true if the connection is to close on the next turn
func (c *connection) isAlive() bool {
	return c.KilledAt.IsZero()
}

// setTimeout adjust the timeout on the connection, goroutine safe
func (c *connection) setTimeout(t time.Duration) (err error) {
	defer c.connGuard.Unlock()
	c.connGuard.Lock()
	if c.conn != nil {
		err = c.conn.SetDeadline(time.Now().Add(t * time.Second))
	}
	return
}

// closeConn closes a connection connection, , goroutine safe
func (c *connection) closeConn() {
	defer c.connGuard.Unlock()
	c.connGuard.Lock()
	_ = c.conn.Close()
	c.conn = nil
}

// UpgradeToTLS upgrades a connection connection to TLS
func (c *connection) upgradeTLS(tlsConfig *tls.Config) error {
	// wrap c.conn in a new TLS Server side connection
	tlsConn := tls.Server(c.conn, tlsConfig)
	// Call handshake here to get any handshake error before reading starts
	err := tlsConn.Handshake()
	if err != nil {
		return err
	}
	// convert tlsConn to net.Conn
	c.conn = net.Conn(tlsConn)

	c.in = NewSMTPReader(c.conn, c.in.Limit())
	c.TLS = true
	return err
}
