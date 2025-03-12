package envelope

//// Dec A WordDecoder decodes MIME headers containing RFC 2047 encoded-words.
//// Used by the MimeHeaderDecode function.
//// It's exposed public so that an alternative decoder can be set, eg Gnu iconv
//// by importing the mail/inconv package.
//// Another alternative would be to use https://godoc.org/golang.org/x/text/encoding
//var Dec mime.WordDecoder
//
//func init() {
//	// use the default decoder, without Gnu inconv. Import the mail/inconv package to use iconv.
//	Dec = mime.WordDecoder{}
//}
//
//// Address encodes an email address of the form `<user@host>`
//type Address struct {
//	// User is local part
//	User string
//	// Host is the domain
//	Host string
//	// ADL is at-domain list if matched
//	ADL []string
//	// PathParams contains any ESTMP parameters that were matched
//	PathParams [][]string
//	// NullPath is true if <> was received
//	NullPath bool
//	// Quoted indicates if the local-part needs quotes
//	Quoted bool
//	// IP stores the IP Address, if the Host is an IP
//	IP net.IP
//	// DisplayName is a label before the address (RFC5322)
//	DisplayName string
//	// DisplayNameQuoted is true when DisplayName was quoted
//	DisplayNameQuoted bool
//}
//
//func (a *Address) String() string {
//	var local string
//	if a.IsEmpty() {
//		return ""
//	}
//	if a.User == "postmaster" && a.Host == "" {
//		return "postmaster"
//	}
//	if a.Quoted {
//		var sb bytes.Buffer
//		sb.WriteByte('"')
//		for i := 0; i < len(a.User); i++ {
//			if a.User[i] == '\\' || a.User[i] == '"' {
//				// escape
//				sb.WriteByte('\\')
//			}
//			sb.WriteByte(a.User[i])
//		}
//		sb.WriteByte('"')
//		local = sb.String()
//	} else {
//		local = a.User
//	}
//	if a.Host != "" {
//		if a.IP != nil {
//			return fmt.Sprintf("%s@[%s]", local, a.Host)
//		}
//		return fmt.Sprintf("%s@%s", local, a.Host)
//	}
//	return local
//}
//
//func (a *Address) IsEmpty() bool {
//	return a.User == "" && a.Host == ""
//}
//
//func (a *Address) IsPostmaster() bool {
//	if a.User == "postmaster" {
//		return true
//	}
//	return false
//}
//
//// NewAddress takes a string of an RFC 5322 address of the
//// form "Gogh Fir <gf@example.com>" or "foo@example.com".
//func NewAddress(str string) (*Address, error) {
//
//	var ap rfc5321.RFC5322
//	l, err := ap.Address([]byte(str))
//	if err != nil {
//		return nil, err
//	}
//	if len(l.List) == 0 {
//		return nil, errors.New("no email address matched")
//	}
//	a := new(Address)
//	addr := &l.List[0]
//	a.User = addr.LocalPart
//	a.Quoted = addr.LocalPartQuoted
//	a.Host = addr.Domain
//	a.IP = addr.IP
//	a.DisplayName = addr.DisplayName
//	a.DisplayNameQuoted = addr.DisplayNameQuoted
//	a.NullPath = addr.NullPath
//	return a, nil
//}
