package envelope

import (
	"context"
	"fmt"
	"github.com/modfin/smtpx/utils"
	"net"
	"net/mail"
	"net/textproto"
)

// Envelope of Email represents a single SMTP message.
type Envelope struct {
	ctx context.Context

	// Remote IP address
	RemoteAddr net.Addr

	// Message sent in EHLO command
	Helo string

	// TLS is true if the email was received using a TLS connection
	TLS bool

	// UTF8
	UTF8 bool

	// ESMTP: true if EHLO was used
	ESMTP bool

	// Sender
	MailFrom *mail.Address

	// Recipients
	RcptTo []*mail.Address

	// Data stores the header and message body
	Data *Data
}

func (e *Envelope) Context() context.Context {
	if e.ctx == nil {
		e.ctx = context.Background()
	}
	return e.ctx
}
func (e *Envelope) WithContext(ctx context.Context) {
	e.ctx = ctx
}

func (e *Envelope) ConnectionId() uint64 {
	ctx := e.Context()
	u, _ := ctx.Value("connection-id").(uint64)
	return u
}
func (e *Envelope) EnvelopeId() string {
	ctx := e.Context()
	u, _ := ctx.Value("envelope-id").(string)
	return u
}

func (e *Envelope) GetError() error {
	ctx := e.Context()
	u, _ := ctx.Value("err").(error)
	return u
}

func (e *Envelope) SetError(err error) {
	e.ctx = context.WithValue(e.Context(), "err", err)
}

func NewEnvelope(remoteAddr net.Addr, connectionId uint64) *Envelope {
	ctx := context.WithValue(context.Background(), "connection-id", connectionId)
	ctx = context.WithValue(ctx, "envelope-id", utils.XID())

	return &Envelope{
		ctx:        ctx,
		RemoteAddr: remoteAddr,
		Data:       &Data{},
	}
}

// PrependHeader adds a header to Data in the envelope, operates on the Data buffer
func (e *Envelope) PrependHeader(key, value string) error {
	_, err := e.Data.PrependString(fmt.Sprintf("%s: %s\r\n", textproto.CanonicalMIMEHeaderKey(key), value))
	return err
}

// Mail will "Open" the envelope and return the mail inside it. Ie the Header and Body
func (e *Envelope) Mail() (*Mail, error) {
	return NewMail(e.Data.Bytes(), e.UTF8)
}
