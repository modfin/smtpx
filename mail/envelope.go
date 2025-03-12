package mail

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/phires/go-guerrilla/mail/rfc5321"
	"mime"
	"net"
	"net/textproto"
)

// Dec  A WordDecoder decodes MIME headers containing RFC 2047 encoded-words.
// Used by the MimeHeaderDecode function.
// It's exposed public so that an alternative decoder can be set, eg Gnu iconv
// by importing the mail/inconv package.
// Another alternative would be to use https://godoc.org/golang.org/x/text/encoding
var Dec mime.WordDecoder

func init() {
	// use the default decoder, without Gnu inconv. Import the mail/inconv package to use iconv.
	Dec = mime.WordDecoder{}
}

// Address encodes an email address of the form `<user@host>`
type Address struct {
	// User is local part
	User string
	// Host is the domain
	Host string
	// ADL is at-domain list if matched
	ADL []string
	// PathParams contains any ESTMP parameters that were matched
	PathParams [][]string
	// NullPath is true if <> was received
	NullPath bool
	// Quoted indicates if the local-part needs quotes
	Quoted bool
	// IP stores the IP Address, if the Host is an IP
	IP net.IP
	// DisplayName is a label before the address (RFC5322)
	DisplayName string
	// DisplayNameQuoted is true when DisplayName was quoted
	DisplayNameQuoted bool
}

func (a *Address) String() string {
	var local string
	if a.IsEmpty() {
		return ""
	}
	if a.User == "postmaster" && a.Host == "" {
		return "postmaster"
	}
	if a.Quoted {
		var sb bytes.Buffer
		sb.WriteByte('"')
		for i := 0; i < len(a.User); i++ {
			if a.User[i] == '\\' || a.User[i] == '"' {
				// escape
				sb.WriteByte('\\')
			}
			sb.WriteByte(a.User[i])
		}
		sb.WriteByte('"')
		local = sb.String()
	} else {
		local = a.User
	}
	if a.Host != "" {
		if a.IP != nil {
			return fmt.Sprintf("%s@[%s]", local, a.Host)
		}
		return fmt.Sprintf("%s@%s", local, a.Host)
	}
	return local
}

func (a *Address) IsEmpty() bool {
	return a.User == "" && a.Host == ""
}

func (a *Address) IsPostmaster() bool {
	if a.User == "postmaster" {
		return true
	}
	return false
}

// NewAddress takes a string of an RFC 5322 address of the
// form "Gogh Fir <gf@example.com>" or "foo@example.com".
func NewAddress(str string) (*Address, error) {
	var ap rfc5321.RFC5322
	l, err := ap.Address([]byte(str))
	if err != nil {
		return nil, err
	}
	if len(l.List) == 0 {
		return nil, errors.New("no email address matched")
	}
	a := new(Address)
	addr := &l.List[0]
	a.User = addr.LocalPart
	a.Quoted = addr.LocalPartQuoted
	a.Host = addr.Domain
	a.IP = addr.IP
	a.DisplayName = addr.DisplayName
	a.DisplayNameQuoted = addr.DisplayNameQuoted
	a.NullPath = addr.NullPath
	return a, nil
}

// Envelope of Email represents a single SMTP message.
type Envelope struct {
	ClientId uint64

	// Remote IP address
	RemoteAddr net.Addr

	// Message sent in EHLO command
	Helo string

	// TLS is true if the email was received using a TLS connection
	TLS bool
	// ESMTP: true if EHLO was used
	ESMTP bool

	// Sender
	MailFrom Address

	// Recipients
	RcptTo []Address

	// Data stores the header and message body
	Data bytes.Buffer
}

func NewEnvelope(remoteAddr net.Addr, clientID uint64) *Envelope {
	return &Envelope{
		ClientId:   clientID,
		RemoteAddr: remoteAddr,
	}
}

// ParseHeaders parses the headers from Envelope
func ParseHeaders(e *Envelope) (textproto.MIMEHeader, error) {
	header, _, found := bytes.Cut(e.Data.Bytes(), []byte{'\n', '\n'}) // the first two new-lines chars are the End Of Header

	if !found {
		return nil, errors.New("could not find headers")
	}
	headerReader := textproto.NewReader(bufio.NewReader(bytes.NewBuffer(header)))
	return headerReader.ReadMIMEHeader()
}

// String converts the email to string.
// Typically, you would want to use the compressor guerrilla.Processor for more efficiency, or use NewReader
func (e *Envelope) String() string {
	return e.Data.String()
}

// PopRcpt removes the last email address that was pushed to the envelope
func (e *Envelope) PopRcpt() Address {
	ret := e.RcptTo[len(e.RcptTo)-1]
	e.RcptTo = e.RcptTo[:len(e.RcptTo)-1]
	return ret
}

const (
	statePlainText = iota
	stateStartEncodedWord
	stateEncodedWord
	stateEncoding
	stateCharset
	statePayload
	statePayloadEnd
)

// MimeHeaderDecode converts 7 bit encoded mime header strings to UTF-8
func MimeHeaderDecode(str string) string {
	// optimized to only create an output buffer if there's need to
	// the `out` buffer is only made if an encoded word was decoded without error
	// `out` is made with the capacity of len(str)
	// a simple state machine is used to detect the start & end of encoded word and plain-text
	state := statePlainText
	var (
		out        []byte
		wordStart  int  // start of an encoded word
		wordLen    int  // end of an encoded
		ptextStart = -1 // start of plan-text
		ptextLen   int  // end of plain-text
	)
	for i := 0; i < len(str); i++ {
		switch state {
		case statePlainText:
			if ptextStart == -1 {
				ptextStart = i
			}
			if str[i] == '=' {
				state = stateStartEncodedWord
				wordStart = i
				wordLen = 1
			} else {
				ptextLen++
			}
		case stateStartEncodedWord:
			if str[i] == '?' {
				wordLen++
				state = stateCharset
			} else {
				wordLen = 0
				state = statePlainText
				ptextLen++
			}
		case stateCharset:
			if str[i] == '?' {
				wordLen++
				state = stateEncoding
			} else if str[i] >= 'a' && str[i] <= 'z' ||
				str[i] >= 'A' && str[i] <= 'Z' ||
				str[i] >= '0' && str[i] <= '9' || str[i] == '-' {
				wordLen++
			} else {
				// error
				state = statePlainText
				ptextLen += wordLen
				wordLen = 0
			}
		case stateEncoding:
			if str[i] == '?' {
				wordLen++
				state = statePayload
			} else if str[i] == 'Q' || str[i] == 'q' || str[i] == 'b' || str[i] == 'B' {
				wordLen++
			} else {
				// abort
				state = statePlainText
				ptextLen += wordLen
				wordLen = 0
			}

		case statePayload:
			if str[i] == '?' {
				wordLen++
				state = statePayloadEnd
			} else {
				wordLen++
			}

		case statePayloadEnd:
			if str[i] == '=' {
				wordLen++
				var err error
				out, err = decodeWordAppend(ptextLen, out, str, ptextStart, wordStart, wordLen)
				if err != nil && out == nil {
					// special case: there was an error with decoding and `out` wasn't created
					// we can assume the encoded word as plaintext
					ptextLen += wordLen //+ 1 // add 1 for the space/tab
					wordLen = 0
					wordStart = 0
					state = statePlainText
					continue
				}
				if skip := hasEncodedWordAhead(str, i+1); skip != -1 {
					i = skip
				} else {
					out = makeAppend(out, len(str), []byte{})
				}
				ptextStart = -1
				ptextLen = 0
				wordLen = 0
				wordStart = 0
				state = statePlainText
			} else {
				// abort
				state = statePlainText
				ptextLen += wordLen
				wordLen = 0
			}

		}
	}

	if out != nil && ptextLen > 0 {
		out = makeAppend(out, len(str), []byte(str[ptextStart:ptextStart+ptextLen]))
		ptextLen = 0
	}

	if out == nil {
		// best case: there was nothing to encode
		return str
	}
	return string(out)
}

func decodeWordAppend(ptextLen int, out []byte, str string, ptextStart int, wordStart int, wordLen int) ([]byte, error) {
	if ptextLen > 0 {
		out = makeAppend(out, len(str), []byte(str[ptextStart:ptextStart+ptextLen]))
	}
	d, err := Dec.Decode(str[wordStart : wordLen+wordStart])
	if err == nil {
		out = makeAppend(out, len(str), []byte(d))
	} else if out != nil {
		out = makeAppend(out, len(str), []byte(str[wordStart:wordLen+wordStart]))
	}
	return out, err
}

func makeAppend(out []byte, size int, in []byte) []byte {
	if out == nil {
		out = make([]byte, 0, size)
	}
	out = append(out, in...)
	return out
}

func hasEncodedWordAhead(str string, i int) int {
	for ; i+2 < len(str); i++ {
		if str[i] != ' ' && str[i] != '\t' {
			return -1
		}
		if str[i+1] == '=' && str[i+2] == '?' {
			return i
		}
	}
	return -1
}
