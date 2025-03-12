package brevx

import (
	"github.com/crholm/brevx/envelope"
	"net/mail"
)

// Backend is process received mail. Depending on the implementation, they can store mail in the database,
// write to a file, check for spam, re-transmit to another Server, etc.
// Must return an SMTP message (i.e. "250 OK") and a boolean indicating
// whether the message was processed successfully.
type Backend interface {

	// Mail is a hook from smtp server to Backend if from is allowed.
	Mail(e *envelope.Envelope, from *mail.Address) error

	// Rcpt is hook from smtp server to Backend if to is allowed.
	Rcpt(e *envelope.Envelope, to *mail.Address) error

	// Process processes then saves the mail envelope
	Process(*envelope.Envelope) Result
}

func BackendFunc(processor func(*envelope.Envelope) Result) Backend {
	return ProcessFunc(processor)
}

// ProcessFunc is a function that processes then saves the mail envelope
type ProcessFunc func(*envelope.Envelope) Result

// Process Make ProcessWith will satisfy the Processor interface
func (f ProcessFunc) Process(e *envelope.Envelope) Result {
	return f(e)
}

func (f ProcessFunc) Mail(e *envelope.Envelope, from *mail.Address) error {
	return nil
}

func (f ProcessFunc) Rcpt(e *envelope.Envelope, to *mail.Address) error {
	return nil
}

var NoopBackend Backend = BackendFunc(func(e *envelope.Envelope) Result {
	return NewResult(250, "OK")
})
