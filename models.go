package brevx

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

var (
	LineLimitExceeded   = errors.New("maximum line length exceeded")
	MessageSizeExceeded = errors.New("maximum message size exceeded")
)

// we need to adjust the limit, so we embed io.LimitedReader
type adjustableLimitedReader struct {
	R *io.LimitedReader
}

// bolt this on so we can adjust the limit
func (alr *adjustableLimitedReader) setLimit(n int64) {
	alr.R.N = n
}

// Returns a specific error when a limit is reached, that can be differentiated
// from an EOF error from the standard io.Reader.
func (alr *adjustableLimitedReader) Read(p []byte) (n int, err error) {
	n, err = alr.R.Read(p)
	if err == io.EOF && alr.R.N <= 0 {
		// return our custom error since io.Reader returns EOF
		err = LineLimitExceeded
	}
	return
}

// allocate a new adjustableLimitedReader
func newAdjustableLimitedReader(r io.Reader, n int64) *adjustableLimitedReader {
	lr := &io.LimitedReader{R: r, N: n}
	return &adjustableLimitedReader{lr}
}

// This is a bufio.Reader what will use our adjustable limit reader
// We 'extend' buffio to have the limited reader feature
type smtpBufferedReader struct {
	*bufio.Reader
	alr *adjustableLimitedReader
}

// Delegate to the adjustable limited reader
func (sbr *smtpBufferedReader) setLimit(n int64) {
	sbr.alr.setLimit(n)
}

// Set a new reader & use it to reset the underlying reader
func (sbr *smtpBufferedReader) Reset(r io.Reader) {
	sbr.alr = newAdjustableLimitedReader(r, CommandLineMaxLength)
	sbr.Reader.Reset(sbr.alr)
}

// Allocate a new SMTPBufferedReader
func newSMTPBufferedReader(rd io.Reader) *smtpBufferedReader {
	alr := newAdjustableLimitedReader(rd, CommandLineMaxLength)
	s := &smtpBufferedReader{bufio.NewReader(alr), alr}
	return s
}

// Result represents a response to an SMTP connection after receiving DATA.
// The String method should return an SMTP message ready to send back to the
// connection, for example `250 OK: Message received`.
type Result interface {
	fmt.Stringer
	// Code should return the SMTP code associated with this response, ie. `250`
	Code() int
	Class() int
}

type result struct {
	code int
	str  string
}

func (r result) String() string {
	var clazz string
	switch r.code / 100 {
	case 2:
		clazz = "OK"
	case 4:
		clazz = "Temporary failure"
	case 5:
		clazz = "Permanent failure"
	}
	return fmt.Sprintf("%d %s: %s", r.code, clazz, r.str)
}

func (r result) Code() int {
	return r.code
}
func (r result) Class() int {
	return r.code / 100
}

func NewResult(code int, str string) Result {
	return result{code: code, str: str}
}

func NewResultFromError(err error) Result {
	return result{code: 500, str: err.Error()}
}
