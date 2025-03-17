package smtpx

import (
	"github.com/modfin/smtpx/envelope"
)

type Middleware func(next HandlerFunc) HandlerFunc
type HandlerFunc func(*envelope.Envelope) Response

// Handler is process received mail. Depending on the implementation, they can store mail in the database,
// write to a file, check for spam, re-transmit to another Server, etc.
//
// Returning nil will translate to 250 OK Response
// Returning a non 2xx Response will abort the transaction
type Handler interface {
	// Data processes then saves the mail envelope
	// Success should return nil or responses.SuccessMessageQueued
	Data(*envelope.Envelope) Response
}

func NewHandler(handler HandlerFunc) Handler {
	return dataFunc(handler)
}

// DataFunc is a function that processes then saves the mail envelope
type dataFunc func(*envelope.Envelope) Response

// Data Make ProcessWith will satisfy the Processor interface
func (f dataFunc) Data(e *envelope.Envelope) Response {
	return f(e)
}

var NoopBackend Handler = NewHandler(func(e *envelope.Envelope) Response {
	return nil
})
