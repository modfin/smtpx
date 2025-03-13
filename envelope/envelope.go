package envelope

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/crholm/brevx/utils"
	"io"
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

func (e *Envelope) ClientId() uint64 {
	ctx := e.Context()
	u, _ := ctx.Value("client-id").(uint64)
	return u
}
func (e *Envelope) EnvelopeId() string {
	ctx := e.Context()
	u, _ := ctx.Value("envelope-id").(string)
	return u
}

func NewEnvelope(remoteAddr net.Addr, clientID uint64) *Envelope {
	ctx := context.WithValue(context.Background(), "client-id", clientID)
	ctx = context.WithValue(ctx, "envelope-id", utils.XID())

	return &Envelope{
		ctx:        ctx,
		RemoteAddr: remoteAddr,
		Data:       &Data{},
	}
}

// AddHeader adds a header to the envelope, operates on the Data buffer
func (e *Envelope) AddHeader(key, value string) error {
	_, err := e.Data.PrependString(fmt.Sprintf("%s: %s\r\n", textproto.CanonicalMIMEHeaderKey(key), value))
	return err
}

// Headers parses the headers from Envelope
func (e *Envelope) Headers() (textproto.MIMEHeader, error) {
	header, _, found := bytes.Cut(e.Data.Bytes(), []byte{'\n', '\n'}) // the first two new-lines chars are the End Of Header

	if !found {
		return nil, errors.New("could not find headers")
	}
	headerReader := textproto.NewReader(bufio.NewReader(bytes.NewBuffer(header)))

	h, err := headerReader.ReadMIMEHeader()
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return h, err
}

// Body returns the email body
func (e *Envelope) Body() ([]byte, error) {
	_, body, found := bytes.Cut(e.Data.Bytes(), []byte{'\n', '\n'}) // the first two new-lines chars are the End Of Header

	if !found {
		return nil, errors.New("could not find body")
	}
	return body, nil
}

func HeaderSubject(headers textproto.MIMEHeader) (*mail.Address, error) {
	from := headers.Get("Subject")
	return mail.ParseAddress(from)
}
func HeaderFrom(headers textproto.MIMEHeader) (*mail.Address, error) {
	from := headers.Get("From")
	return mail.ParseAddress(from)
}
func HeaderTo(headers textproto.MIMEHeader) ([]*mail.Address, error) {
	from := headers.Get("To")
	return mail.ParseAddressList(from)
}
func HeaderCc(headers textproto.MIMEHeader) ([]*mail.Address, error) {
	cc := headers.Get("Cc")
	return mail.ParseAddressList(cc)
}
