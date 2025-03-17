package brevx

import (
	"bufio"
	"errors"
	"io"
	"net/textproto"
)

type SMTPReaderError string

func (e SMTPReaderError) Error() string {
	return string(e)
}

const LimitError SMTPReaderError = "read limit reached"

// Copy past from io.LimitReader, with limit error replaces
type limitedReader struct {
	R io.Reader // underlying reader
	N int64     // max bytes remaining
}

func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, errors.Join(io.EOF, LimitError)
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return
}

type smtpReader struct {
	n     int64 // max byte read
	raw   io.Reader
	limit *limitedReader
	buf   *bufio.Reader

	*textproto.Reader
}

// NewSMTPReader setup a new smtpReader with n byte read limit
func NewSMTPReader(r io.Reader, n int64) *smtpReader {

	limit := &limitedReader{R: r, N: n}
	buf := bufio.NewReader(limit)
	proto := textproto.NewReader(buf)

	return &smtpReader{
		n:     n,
		raw:   r,
		limit: limit,
		buf:   buf,

		Reader: proto,
	}
}

func (r *smtpReader) Limit() int64 {
	return r.n
}

func (r *smtpReader) ResetLimit() {
	r.limit.N = r.n
}
