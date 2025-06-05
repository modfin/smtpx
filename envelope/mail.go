package envelope

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/text/transform"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"net/url"
	"strings"
)

type Headers struct {
	textproto.MIMEHeader
}

type RawHeaders struct {
	textproto.MIMEHeader
}

func (h RawHeaders) From() (*mail.Address, error) {
	from := h.Get("From")
	return mail.ParseAddress(from)
}
func (h RawHeaders) To() ([]*mail.Address, error) {
	to := h.Get("To")
	return mail.ParseAddressList(to)
}

// NewMail returns a new mail struct. It takes in a byte slice of the entire email contant. ie. header + body
// eg
//
//	From: sender@example.com
//	To: recipient@example.com
//	Subject: Test Email
//	Content-Type: text/plain
//
//	This is a test email body.
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

// Headers parses the headers from Envelope to a human-readable format.
func (e *Mail) Headers() (Headers, error) {
	headerReader := textproto.NewReader(bufio.NewReader(bytes.NewBuffer(e.RawHeaders)))

	h, err := headerReader.ReadMIMEHeader()
	if errors.Is(err, io.EOF) {
		err = nil
	}

	// If UTF8 is true, there should not be any need for decoding...
	// And there are no charset endoding blocks =?charset?[b/q]?<data>?=
	if e.UTF8 && !bytes.Contains(e.RawHeaders, []byte("=?")) {
		return Headers{h}, err
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
	return Headers{h}, err
}

// HeadersLiteral parses the headers from Envelope without decoding the values.
// This is more useful for e.g. parsing of email addresses.
//
// Example: The following From header will give the following result in its literal and decoded form using Headers().
// Notice that the decoded form is not a valid RFC 5322 address.
//
// From: =?iso-8859-1?Q?Lastname=2C_=F6?= <o.Lastname@company.com>
// Literal: =?iso-8859-1?Q?Lastname=2C_=F6?= <o.Lastname@company.com>
// Decoded: Lastname, ö <o.Lastname@company.com>
func (e *Mail) HeadersLiteral() (RawHeaders, error) {
	headerReader := textproto.NewReader(bufio.NewReader(bytes.NewBuffer(e.RawHeaders)))

	h, err := headerReader.ReadMIMEHeader()
	if errors.Is(err, io.EOF) {
		err = nil
	}

	return RawHeaders{h}, err
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

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]

		if boundary == "" {
			return nil, errors.New("no boundary in Content-Type params")
		}

		mr := multipart.NewReader(body, params["boundary"])

		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
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

		return content, nil
	}

	content.Body, err = io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return content, nil

}

type Content struct {
	Headers  textproto.MIMEHeader
	Body     []byte
	Children []Content
}

func (c *Content) filename() (string, error) {

	header := c.Headers.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(header)
	if err != nil {
		return "", fmt.Errorf("failed to parse Content-Disposition: %w", err)
	}
	name := params["filename"]
	if name == "" {
		_, params, err = mime.ParseMediaType(c.Headers.Get("Content-Type"))
		if err != nil {
			return "", fmt.Errorf("failed to parse Content-Tyep: %w", err)
		}
		name = params["name"]
	}

	if name == "" {
		return "", errors.New("no filename in Content-Disposition nor name in Content-Type params")
	}

	if !strings.Contains(header, "filename*") { // if not  RFC5987, percent decode
		name, err = url.QueryUnescape(name)
		if err != nil {
			return "", fmt.Errorf("failed to unescape filename: %w", err)
		}
	}
	return name, nil
}

func (c *Content) Encoding() string {
	enc := c.Headers.Get("Content-Transfer-Encoding")
	if enc == "" {
		enc = "7bit"
	}
	return enc
}

func (c *Content) Decode() ([]byte, error) {
	enc := c.Encoding()

	ct := c.Headers.Get("Content-Type")
	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Content-Type: %w", err)
	}
	charset := strings.ToLower(params["charset"])

	if charset == "" {
		charset = "utf-8"
	}

	var toUTF8 func(r io.Reader) io.Reader
	toUTF8 = func(r io.Reader) io.Reader {
		return r
	}

	if charset != "utf-8" {

		m, ok := charsetEncodings[charset]
		if !ok {
			m, ok = charsetEncodings[charsetAliases[charset]]
		}
		if m != nil {
			toUTF8 = func(r io.Reader) io.Reader {
				return transform.NewReader(r, m.NewDecoder())
			}
		}
	}

	switch enc {

	case "quoted-printable":

		data, err := io.ReadAll(toUTF8(quotedprintable.NewReader(bytes.NewReader(c.Body))))
		return data, err
	case "base64":

		data, err := base64.StdEncoding.DecodeString(string(c.Body))
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %w", err)
		}

		data, err = io.ReadAll(toUTF8(bytes.NewReader(data)))

		return data, err
	case "7bit", "8bit", "binary":
		data, err := io.ReadAll(toUTF8(bytes.NewReader(c.Body)))
		return data, err
	default:
		return nil, fmt.Errorf("unknown encoding: %s", enc)
	}

}

func (c *Content) Walk(fn func(*Content, int) error) error {
	var rec func(c *Content, level int) error
	rec = func(c *Content, level int) error {
		if err := fn(c, level); err != nil {
			return err
		}
		for _, child := range c.Children {

			if err := rec(&child, level+1); err != nil {
				return err
			}

		}
		return nil
	}
	return rec(c, 0)
}

func (c *Content) Flatten() []*Content {

	if c.Leaf() {
		return []*Content{{
			Headers:  c.Headers,
			Body:     c.Body,
			Children: nil,
		}}
	}
	var res []*Content
	for _, child := range c.Children {
		res = append(res, child.Flatten()...)
	}

	return res
}

func (c *Content) Leaf() bool {
	return len(c.Children) == 0 && !strings.HasPrefix(c.Headers.Get("Content-Type"), "multipart/")
}

func (c *Content) IsForm() bool {
	_type, _, _ := mime.ParseMediaType(c.Headers.Get("Content-Disposition"))
	return strings.ToLower(_type) == "form-data"
}

func (c *Content) AsForm() (*FormPart, error) {
	if !c.IsForm() {
		return nil, errors.New("not a form")
	}
	return &FormPart{c}, nil
}

func (c *Content) IsInline() bool {
	_type, _, _ := mime.ParseMediaType(c.Headers.Get("Content-Disposition"))
	return strings.ToLower(_type) == "inline"
}

func (c *Content) AsInline() (*InlinePart, error) {
	if !c.IsInline() {
		return nil, errors.New("not an inline")
	}

	return &InlinePart{c}, nil
}

func (c *Content) IsAttachment() bool {
	_type, _, _ := mime.ParseMediaType(c.Headers.Get("Content-Disposition"))
	return strings.ToLower(_type) == "attachment"
}

func (c *Content) AsAttachment() (*AttachmentPart, error) {
	if !c.IsAttachment() {
		return nil, errors.New("not an attachment")
	}

	return &AttachmentPart{c}, nil
}

type FormPart struct {
	c *Content
}

func (a *FormPart) Filename() (string, error) {
	return a.c.filename()
}

func (a *FormPart) Name() (string, error) {
	header := a.c.Headers.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(header)
	if err != nil {
		return "", fmt.Errorf("failed to parse Content-Disposition: %w", err)
	}
	name := params["name"]
	if name == "" {
		return "", errors.New("no name in Content-Disposition params")
	}

	return name, nil
}

type InlinePart struct {
	c *Content
}

func (a *InlinePart) Filename() (string, error) {
	return a.c.filename()
}

type AttachmentPart struct {
	c *Content
}

func (a *AttachmentPart) Filename() (string, error) {
	return a.c.filename()
}
