package envelope

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
)

func NewMail(data []byte, utf8 bool) (*Mail, error) {
	header, body, found := bytes.Cut(data, []byte("\r\n\r\n"))
	if !found {
		header, body, found = bytes.Cut(data, []byte("\n\n"))
	}
	if !found {
		return nil, errors.New("could not find body")
	}
	return &Mail{RawHeaders: header, RawBody: body, UTF8: utf8}, nil
}

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

type Content struct {
	Headers textproto.MIMEHeader
	Body    []byte

	Children []Content
}

func (e *Mail) Body() (*Content, error) {
	h, err := e.Headers()
	if err != nil {
		return nil, err
	}
	head := textproto.MIMEHeader{}
	head.Set("Content-Type", h.Get("Content-Type"))

	enc := h.Get("Content-Transfer-Encoding")
	if enc != "" {
		head.Set("Content-Transfer-Encoding", enc)
	}
	dis := h.Get("Content-Disposition")
	if dis != "" {
		head.Set("Content-Disposition", dis)
	}
	id := h.Get("Content-ID")
	if id != "" {
		head.Set("Content-ID", id)
	}

	return parseContent(head, bytes.NewReader(e.RawBody))
}

// Body returns the email body
func parseContent(headers textproto.MIMEHeader, body io.Reader) (*Content, error) {

	mediaType, params, err := mime.ParseMediaType(headers.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	content := &Content{Headers: headers}

	switch mediaType {
	case "multipart/mixed", "multipart/alternative", "multipart/related", "multipart/signed":
		mr := multipart.NewReader(body, params["boundary"])

		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				return content, nil
			}
			if err != nil {
				return nil, fmt.Errorf("failed to get part: %w", err)
			}
			child, err := parseContent(p.Header, p)
			if err != nil {
				return nil, err
			}
			content.Children = append(content.Children, *child)
		}

	default:

		content.Body, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body: %w", err)
		}

		return content, nil
	}
}
