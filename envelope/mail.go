package envelope

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"mime"
	"net/textproto"
)

type Mail struct {
	UTF8 bool

	RawHeaders []byte
	RawBody    []byte
}

// Headers parses the headers from Envelope
func (e *Mail) Headers() (textproto.MIMEHeader, error) {
	headerReader := textproto.NewReader(bufio.NewReader(bytes.NewBuffer(e.RawHeaders)))

	h, err := headerReader.ReadMIMEHeader()
	if errors.Is(err, io.EOF) {
		err = nil
	}

	// If UTF8 is true, there should not be any need for decoding...
	// And there are no charset endoding blocks =?charset?[b/q]?<data>?=
	if e.UTF8 && !bytes.Contains(e.RawHeaders, []byte("=?")) {
		return h, err
	}

	dec := &mime.WordDecoder{
		CharsetReader: charsetReader,
	}

	// decode all headers
	for k, vv := range h {
		var vv2 []string
		for _, v := range vv {
			v2, err := dec.DecodeHeader(v) // This will end up being some random endo, need to conver it to UTF8
			if err != nil {
				v2 = v
			}
			vv2 = append(vv2, v2)
		}
		h[k] = vv2
	}
	return h, err
}

// Body returns the email body
func (e *Mail) Body() []byte {
	return e.RawBody
}
