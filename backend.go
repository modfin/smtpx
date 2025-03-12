package guerrilla

import (
	"github.com/phires/go-guerrilla/mail"
)

// Backend is process received mail. Depending on the implementation, they can store mail in the database,
// write to a file, check for spam, re-transmit to another Server, etc.
// Must return an SMTP message (i.e. "250 OK") and a boolean indicating
// whether the message was processed successfully.
type Backend interface {
	// Process processes then saves the mail envelope
	Process(*mail.Envelope) Result

	// Mail is a hook from smtp server to backend if from is allowed.
	Mail(from mail.Address) error

	// Rcpt is hook from smtp server to backend if to is allowed.
	Rcpt(to mail.Address) error
}

func BackendFunc(processor func(*mail.Envelope) Result) Backend {
	return ProcessFunc(processor)
}

// ProcessWith Signature of Processor
type ProcessFunc func(*mail.Envelope) Result

// Process Make ProcessWith will satisfy the Processor interface
func (f ProcessFunc) Process(e *mail.Envelope) Result {
	return f(e)
}

func (f ProcessFunc) Mail(from mail.Address) error {
	return nil
}

func (f ProcessFunc) Rcpt(to mail.Address) error {
	return nil
}

var NoopBackend Backend = BackendFunc(func(e *mail.Envelope) Result {
	return NewResult(250, "OK")
})
