package envelope

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"strings"
	"testing"
)

//// Test MimeHeader decoding, not using iconv
//func TestMimeHeaderDecode(t *testing.T) {
//
//	/*
//		Normally this would fail if not using iconv
//		str := MimeHeaderDecode("=?ISO-2022-JP?B?GyRCIVo9dztSOWJAOCVBJWMbKEI=?=")
//		if i := strings.Index(str, "【女子高生チャ"); i != 0 {
//			t.Error("expecting 【女子高生チャ, got:", str)
//		}
//	*/
//
//	str := MimeHeaderDecode("=?utf-8?B?55So5oi34oCcRXBpZGVtaW9sb2d5IGluIG51cnNpbmcgYW5kIGg=?=  =?utf-8?B?ZWFsdGggY2FyZSBlQm9vayByZWFkL2F1ZGlvIGlkOm8=?=  =?utf-8?B?cTNqZWVr4oCd5Zyo572R56uZ4oCcU1BZ5Lit5paH5a6Y5pa5572R56uZ4oCd?=  =?utf-8?B?55qE5biQ5Y+36K+m5oOF?=")
//	if i := strings.Index(str, "用户“Epidemiology in nursing and health care eBook read/audio id:oq3jeek”在网站“SPY中文官方网站”的帐号详情"); i != 0 {
//		t.Error("\nexpecting \n用户“Epidemiology in nursing and h ealth care eBook read/audio id:oq3jeek”在网站“SPY中文官方网站”的帐号详情\n got:\n", str)
//	}
//	str = MimeHeaderDecode("=?ISO-8859-1?Q?Andr=E9?= Pirard <PIRARD@vm1.ulg.ac.be>")
//	if strings.Index(str, "André Pirard") != 0 {
//		t.Error("expecting André Pirard, got:", str)
//	}
//}
//
//// TestMimeHeaderDecodeNone tests strings without any encoded words
//func TestMimeHeaderDecodeNone(t *testing.T) {
//	// in the best case, there will be nothing to decode
//	str := MimeHeaderDecode("Andre Pirard <PIRARD@vm1.ulg.ac.be>")
//	if strings.Index(str, "Andre Pirard") != 0 {
//		t.Error("expecting Andre Pirard, got:", str)
//	}
//}

func TestAddressPostmaster(t *testing.T) {
	addr := &mail.Address{Name: "postmaster"}
	str := addr.String()
	if str != `"postmaster" <@>` {
		t.Error("it was not postmaster,", str)
	}
}

//func TestAddressNull(t *testing.T) {
//	addr := &Address{NullPath: true}
//	str := addr.String()
//	if str != "" {
//		t.Error("it was not empty", str)
//	}
//}

func TestNewAddress(t *testing.T) {

	addr, err := mail.ParseAddress("<hoop>")
	if err == nil {
		t.Error("there should be an error:", err)
	}

	addr, err = mail.ParseAddress(`Gogh Fir <tesst@test.com>`)
	if err != nil {
		t.Error("there should be no error:", addr.String(), err)
	}
}

func TestQuotedAddress(t *testing.T) {

	str := `<"  yo-- man wazz'''up? surprise \surprise, this is POSSIBLE@fake.com "@example.com>`
	//str = `<"post\master">`
	addr, err := mail.ParseAddress(str)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	str = addr.String()
	// in this case, string should remove the unnecessary escape
	if strings.Contains(str, "\\surprise") {
		t.Error("there should be no \\surprise:", err)
	}

}

func TestAddressWithIP(t *testing.T) {
	str := `<"  yo-- man wazz'''up? surprise \surprise, this is POSSIBLE@fake.com "@[64.233.160.71]>`
	addr, err := mail.ParseAddress(str)
	if err != nil {
		t.Error("there should be no error:", err)
	} else if addr == nil {
		t.Error("expecting the address host to be an IP")
	}
	//fmt.Println("name:", addr.Name)
	//fmt.Println("address:", addr.Address)
}

func TestEnvelope(t *testing.T) {

	e := NewEnvelope(&net.TCPAddr{IP: net.ParseIP("127.0.0.1")}, 22)

	e.Helo = "helo.example.com"
	e.MailFrom = &mail.Address{Name: "test", Address: "test@example.com"}
	e.TLS = true
	e.RemoteAddr = &net.TCPAddr{IP: net.ParseIP("222.111.233.121")}
	to := &mail.Address{Address: "test@example.com"}
	e.RcptTo = append(e.RcptTo, to)
	if to.String() != "<test@example.com>" {
		t.Error("to does not equal test@example.com, it was:", to.String())
	}
	_, err := e.Data.WriteString("Subject: Test\n\nThis is a test nbnb nbnb hgghgh nnnbnb nbnbnb nbnbn.")
	if err != nil {
		t.Error("could not write headers1:", err)
		return
	}

	addHead := "Delivered-To: " + to.String() + "\n"
	addHead += "Received: from " + e.Helo + " (" + e.Helo + "  [" + e.RemoteAddr.String() + "])\n"
	_, err = e.Data.WriteString(fmt.Sprintf("%s\n\nHello Test", addHead))
	if err != nil {
		t.Error("could not write headers2:", err)
		return
	}

	r := e.Data.Reader()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Error("co", err)
	}
	if len(data) != e.Data.Len() {
		t.Error("e.Len() is incorrect, it shown ", e.Data.Len(), " but we wanted ", len(data))
	}
	headers, err := e.Headers()
	if err != nil && !errors.Is(err, io.EOF) {
		t.Error("cannot parse headers:", err)
		return
	}
	if headers.Get("Subject") != "Test" {
		t.Error("Subject expecting: Test, got:", headers.Get("Subject"))
	}

}

// TestEnvelopeLargeHeader is a test function that tests the behavior of the Envelope struct when handling large headers.
//
// It creates a new Envelope instance with a remote address and client ID, sets various properties of the Envelope,
// adds a recipient email address, and writes a large header (about 11MiB) to the Data buffer. It then creates a delivery header
// and parses the headers of the Envelope. Finally, it checks if the Subject of the Envelope is correct.
//
// Parameters:
// - t: A testing.T object for running the test and reporting any failures.
//
// Returns: None.
func TestEnvelopeLargeHeader(t *testing.T) {

	e := NewEnvelope(&net.TCPAddr{IP: net.ParseIP("127.0.0.1")}, 22)

	e.Helo = "helo.example.com"
	e.MailFrom = &mail.Address{Name: "test", Address: "test@example.com"}
	e.TLS = true
	e.RemoteAddr = &net.TCPAddr{IP: net.ParseIP("222.111.233.121")}
	to := &mail.Address{Address: "test@example.com"}
	e.RcptTo = append(e.RcptTo, to)
	if to.String() != "<test@example.com>" {
		t.Error("to does not equal test@example.com, it was:", to.String())
	}
	header := ""
	for i := 0; i < 17; i++ {
		header += fmt.Sprintf("%sX-Dummy%d: %s\n", header, i, strings.Repeat("n", 68))
	}
	header += "Subject: Large Header Test"
	fmt.Printf("Header length: %dKiB / %dMiB\n", len(header)/1024, len(header)/1024/1024)

	e.Data.WriteString(fmt.Sprintf("%s\n\nHello Test", header))

	addHead := "Delivered-To: " + to.String() + "\n"
	addHead += "Received: from " + e.Helo + " (" + e.Helo + "  [" + e.RemoteAddr.String() + "])\n"
	e.Data.PrependString(addHead)

	r := e.Data.Reader()

	data, _ := io.ReadAll(r)
	if len(data) != e.Data.Len() {
		t.Error("e.Len() is incorrect, it shown ", e.Data.Len(), " but we wanted ", len(data))
	}
	headers, err := e.Headers()

	if err != nil && !errors.Is(err, io.EOF) {
		t.Error("cannot parse headers:", err)
		return
	}
	if headers.Get("Subject") != "Large Header Test" {
		t.Error("Subject expecting: Test, got:", headers.Get("Subject"))
	}

}

//func TestEncodedWordAhead(t *testing.T) {
//	str := "=?ISO-8859-1?Q?Andr=E9?= Pirard <PIRARD@vm1.ulg.ac.be>"
//	if hasEncodedWordAhead(str, 24) != -1 {
//		t.Error("expecting no encoded word ahead")
//	}
//
//	str = "=?ISO-8859-1?Q?Andr=E9?= ="
//	if hasEncodedWordAhead(str, 24) != -1 {
//		t.Error("expecting no encoded word ahead")
//	}
//
//	str = "=?ISO-8859-1?Q?Andr=E9?= =?ISO-8859-1?Q?Andr=E9?="
//	if hasEncodedWordAhead(str, 24) == -1 {
//		t.Error("expecting an encoded word ahead")
//	}
//
//}
