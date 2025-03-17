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

	m, err := e.Mail()
	if err != nil {
		t.Error("cannot parse mail:", err)
		return
	}

	headers, err := m.Headers()
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
	header += "Subject: Large Headers Test"
	fmt.Printf("Headers length: %dKiB / %dMiB\n", len(header)/1024, len(header)/1024/1024)

	e.Data.WriteString(fmt.Sprintf("%s\n\nHello Test", header))

	addHead := "Delivered-To: " + to.String() + "\n"
	addHead += "Received: from " + e.Helo + " (" + e.Helo + "  [" + e.RemoteAddr.String() + "])\n"
	e.Data.PrependString(addHead)

	r := e.Data.Reader()

	data, _ := io.ReadAll(r)
	if len(data) != e.Data.Len() {
		t.Error("e.Len() is incorrect, it shown ", e.Data.Len(), " but we wanted ", len(data))
	}
	m, err := e.Mail()
	if err != nil {
		t.Error("cannot parse mail:", err)
		return
	}

	headers, err := m.Headers()

	if err != nil && !errors.Is(err, io.EOF) {
		t.Error("cannot parse headers:", err)
		return
	}
	if headers.Get("Subject") != "Large Headers Test" {
		t.Error("Subject expecting: Test, got:", headers.Get("Subject"))
	}

}

func TestMIMEHeaderDecoding(t *testing.T) {

	type testCase struct {
		input string
		exp   string
	}
	testCases := []testCase{
		// Basic UTF-8 Base64 encoded header
		{
			input: "Subject: =?UTF-8?B?VGVzdCB3aXRoIMOpIGFuZCDkuK3lj7g=?=",
			exp:   "Test with é and 中司",
		},
		// UTF-8 Quoted-Printable encoded header
		{
			input: "Subject: =?UTF-8?Q?Test_with_=C3=A9_and_=E4=B8=AD=E6=96=87?=",
			exp:   "Test with é and 中文",
		},
		// Multiple encoded words
		{
			input: "Subject: =?UTF-8?B?VGVzdA==?= =?UTF-8?B?IHdpdGgg?= =?UTF-8?B?w6k=?=",
			exp:   "Test with é",
		},
		// Mixed encoding types (B and Q)
		{
			input: "Subject: =?UTF-8?B?VGVzdA==?= =?UTF-8?Q?_with_=C3=A9?=",
			exp:   "Test with é",
		},
		// ISO-8859-1 encoding
		{
			input: "Subject: =?ISO-8859-1?Q?Pr=FCfung?=",
			exp:   "Prüfung",
		},
		// Headers with regular text mixed with encoded words
		{
			input: "Subject: Hello =?UTF-8?B?w7xtbGF1dA==?= test",
			exp:   "Hello ümlaut test",
		},
		// From header with name and email
		{
			input: "Subject: =?UTF-8?B?5byg5LiJ?= <zhang@example.com>",
			exp:   "张三 <zhang@example.com>",
		},
		// Multiple headers
		{
			input: "Subject: =?UTF-8?B?5ryi5a2X?=\r\nFrom: =?UTF-8?B?5ZGo5LqM?= <wang@example.com>",
			exp:   "漢字",
		},
		// Japanese text in UTF-8
		{
			input: "Subject: =?UTF-8?B?5pel5pys6Kqe44Gu44OG44K544OI?=",
			exp:   "日本語のテスト",
		},
		// Cyrillic text in UTF-8 (Russian)
		{
			input: "Subject: =?UTF-8?B?0J/RgNC40LLQtdGCINC80LjRgCE=?=",
			exp:   "Привет мир!",
		},
		// Korean text in UTF-8
		{
			input: "Subject: =?UTF-8?B?7ZWc6rWt7Ja0IOyduOymneuLiOuLpA==?=",
			exp:   "한국어 인증니다",
		},
		// Arabic text in UTF-8
		{
			input: "Subject: =?UTF-8?B?2KfYrtiq2KjYp9ixINin2YTZhti1?=",
			exp:   "اختبار النص",
		},
		// Hebrew text in UTF-8
		{
			input: "Subject: =?UTF-8?B?15HXk9eZ16fXldeqINei15HXqNeZ16o=?=",
			exp:   "בדיקות עברית",
		},
		// Thai text in UTF-8
		{
			input: "Subject: =?UTF-8?B?4LiX4LiU4Liq4Lit4Lia4LiA4Liy4Lij4LiX4LiU4Liq4Lit4Lia4LiA4Liy4Lij4LiX4LiU4Liq4Lit4Lia?=",
			exp:   "ทดสอบ\u0E00ารทดสอบ\u0E00ารทดสอบ",
		},
		// Q encoding with underscores for spaces
		{
			input: "Subject: =?UTF-8?Q?This_is_a_test_with_spaces?=",
			exp:   "This is a test with spaces",
		},
		// Q encoding with hexadecimal values
		{
			input: "Subject: =?UTF-8?Q?Euro_symbol:_=E2=82=AC?=",
			exp:   "Euro symbol: €",
		},
		// Long header that might be wrapped
		{
			input: "Subject: =?UTF-8?B?VGhpcyBpcyBhIHZlcnkgbG9uZyBzdWJqZWN0IGxpbmUgdGhhdCBtaWdodCBiZSB3cmFwcGVkIGluIHRoZSBlbWFpbCBoZWFkZXIu?=",
			exp:   "This is a very long subject line that might be wrapped in the email header.",
		},
		// Multiple character sets
		{
			input: "Subject: =?UTF-8?B?VVRGLTg=?= and =?ISO-8859-1?Q?ISO-8859-1?=",
			exp:   "UTF-8 and ISO-8859-1",
		},
		// Emoji in UTF-8
		{
			input: "Subject: =?UTF-8?B?8J+Ygg==?= =?UTF-8?B?IA==?= =?UTF-8?B?8J+Yhg==?= A =?UTF-8?B?8J+Ygw==?=",
			exp:   "😂 😆 A 😃",
		},
		// Special characters that need encoding
		{
			input: "Subject: =?UTF-8?Q?Special_chars:_=5B=3C=3E=40=2C=3B=3A=5C=2F=22=28=29=5D?=",
			exp:   "Special chars: [<>@,;:\\/\"()]",
		},
		// Accented characters in ISO-8859-1
		{
			input: "Subject: =?ISO-8859-1?Q?=E1=E9=ED=F3=FA=F1=E7?=",
			exp:   "áéíóúñç",
		},
		// Unprintable ASCII control characters
		{
			input: "Subject: Control =?UTF-8?Q?chars:_=01=02=03=1F?=",
			exp:   "Control chars: \x01\x02\x03\x1F",
		},
		// Chinese (Traditional) characters
		{
			input: "Subject: =?UTF-8?B?5Lit5paH6Ieq5YuV6Kqe6KiCIC0g57mB57S/5YyW?=",
			exp:   "中文自動語訂 - 繁紿化",
		},
		// Vietnamese characters
		{
			input: "Subject: =?UTF-8?B?VMOgaSBsaeG7h3UgVmnhu4d0IE5hbQ==?=",
			exp:   "Tài liệu Việt Nam",
		},
		// Mixed plain text and encoded text
		{
			input: "Subject: Plain text and =?UTF-8?B?ZW5jb2RlZCB0ZXh0?= mixed",
			exp:   "Plain text and encoded text mixed",
		},
		// Line folding in headers with encoded content
		{
			input: "Subject: =?UTF-8?B?TGluZSBm?=\r\n =?UTF-8?B?b2xkaW5n?= test",
			exp:   "Line folding test",
		},
		// Multiple different headers
		{
			input: "Subject: =?UTF-8?B?VGVzdA==?=\r\nTo: =?UTF-8?B?VXNlcg==?= <user@example.com>\r\nFrom: =?UTF-8?B?U2VuZGVy?= <sender@example.com>",
			exp:   "Test",
		},
		// Case differences in encoding type and charset
		{
			input: "Subject: =?utf-8?b?dGVzdCBsb3dlcmNhc2U=?=",
			exp:   "test lowercase",
		},
		// Non-standard but sometimes encountered format without spaces
		{
			input: "Subject:=?UTF-8?B?Tm8gc3BhY2VzIGJlZm9yZSBlbmNvZGVkIHdvcmRz?=",
			exp:   "No spaces before encoded words",
		},

		{
			input: "Subject: =?ISO-8859-1?B?SG9sYSBjb20gZXN06/M=?=",
			exp:   "Hola com estëó",
		},
		{
			input: "Subject: =?ISO-8859-1?Q?Pr=FCfung=20der=20=DCbertragung?=",
			exp:   "Prüfung der Übertragung",
		},
		{
			input: "Subject: =?ISO-8859-1?Q?J=F8rgen_Hansen?= <jorgen@example.com>",
			exp:   "Jørgen Hansen <jorgen@example.com>",
		},
		{
			input: "Subject: =?ISO-8859-1?B?VHLpbmluZw==?= =?ISO-8859-1?Q?_de_fran=E7ais?=",
			exp:   "Tréning de français",
		},

		// TODO someting to add all charsets in how we parse headers...
		// ISO-8859-2 (Latin-2, Central European)
		//{
		//	input: "Subject: =?ISO-8859-2?B?UG9sc2tpZSB6bmFraTog5eXt7A==?=",
		//	exp:   "Polskie znaki: ĺĺíě",
		//},
		//{
		//	input: "Subject: =?ISO-8859-2?Q?=C8esk=E9_znaky?=",
		//	exp:   "České znaky",
		//},
		//{
		//	input: "Subject: =?ISO-8859-2?B?THVrYXMgTXJhenZh?= <lukas@example.cz>",
		//	exp:   "Lukas Mrazva <lukas@example.cz>",
		//},

		//ISO-8859-3 (Latin-3, South European)
		//{
		//	input: "Subject: =?ISO-8859-3?Q?T=FCrk=E7e_mesaj=FD?=",
		//	exp:   "Türkçe mesajŭ",
		//},
		//{
		//	input: "Subject: =?ISO-8859-3?B?TWFsdGEgYW5kIEVzcGVyYW50byDmtqM=?=",
		//	exp:   "Malta and Esperanto ĝħş",
		//},

		// ISO-8859-4 (Latin-4, North European)
		//{
		//	input: "Subject: =?ISO-8859-4?Q?Latvie=F0u_valoda?=",
		//	exp:   "Latviešu valoda",
		//},
		//{
		//	input: "Subject: =?ISO-8859-4?B?TGlldHV2acWzIGthbGJh?=",
		//	exp:   "Lietuvių kalba",
		//},

		//// ISO-8859-5 (Cyrillic)
		//{
		//	input: "Subject: =?ISO-8859-5?B?UHVza2luOiDXwdDb0N/Q1A==?=",
		//	exp:   "Puskin: Стихи",
		//},
		//{
		//	input: "Subject: =?ISO-8859-5?Q?=D0=C0=D1=D1=D2=C9=D9_=D4=CF=D2=CD=C1=D4?=",
		//	exp:   "РУССКИЙ ФОРМАТ",
		//},
		//
		//// ISO-8859-6 (Arabic)
		//{
		//	input: "Subject: =?ISO-8859-6?B?2KfZhNi52LHYqNmK2Kk=?=",
		//	exp:   "العربية",
		//},
		//{
		//	input: "Subject: =?ISO-8859-6?Q?=C7=E1=DA=D1=C8=ED=C9?=",
		//	exp:   "العربية",
		//},
		//
		//// ISO-8859-7 (Greek)
		//{
		//	input: "Subject: =?ISO-8859-7?B?RWxsZ25pa+EgZ2wmIzk0MztzczM=?=",
		//	exp:   "Ellhnikά glώssα",
		//},
		//{
		//	input: "Subject: =?ISO-8859-7?Q?=C5=EB=EB=E7=ED=E9=EA=DC?=",
		//	exp:   "Ελληνικά",
		//},
		//
		//// ISO-8859-8 (Hebrew)
		//{
		//	input: "Subject: =?ISO-8859-8?B?5eXp6unt?=",
		//	exp:   "עברית",
		//},
		//{
		//	input: "Subject: =?ISO-8859-8?Q?=F9=E1=F8=E9=FA?=",
		//	exp:   "שלרית",
		//},
		//
		//// ISO-8859-9 (Latin-5, Turkish)
		//{
		//	input: "Subject: =?ISO-8859-9?B?VMO8cmvDp2Ugw5ZybmVrIE1ldGlu?=",
		//	exp:   "Türkçe Örnek Metin",
		//},
		//{
		//	input: "Subject: =?ISO-8859-9?Q?T=FCrk=E7e=20=D6rnek=20Metin?=",
		//	exp:   "Türkçe Örnek Metin",
		//},
		//
		//// ISO-8859-15 (Latin-9, update of Latin-1 with € symbol)
		//{
		//	input: "Subject: =?ISO-8859-15?B?RXVybyBzeW1ib2w6IKM=?=",
		//	exp:   "Euro symbol: €",
		//},
		//{
		//	input: "Subject: =?ISO-8859-15?Q?Euro_symbol=3A_=A4?=",
		//	exp:   "Euro symbol: €",
		//},
		//
		//// Windows-1250 (Central European)
		//{
		//	input: "Subject: =?WINDOWS-1250?B?UG9sc2tpZSB6bmFraTog5eXt7A==?=",
		//	exp:   "Polskie znaki: ąęłńś",
		//},
		//{
		//	input: "Subject: =?WINDOWS-1250?Q?=C8esk=E9_znaky?=",
		//	exp:   "České znaky",
		//},
		//
		//// Windows-1251 (Cyrillic)
		//{
		//	input: "Subject: =?WINDOWS-1251?B?0KDRg9GB0YHQutC40LkgItCi0LXQutGB0YIi?=",
		//	exp:   "Русский \"Текст\"",
		//},
		//{
		//	input: "Subject: =?WINDOWS-1251?Q?=D0=F3=F1=F1=EA=E8=E9_=F2=E5=EA=F1=F2?=",
		//	exp:   "Русский текст",
		//},
		//
		//// Windows-1252 (Western European)
		//{
		//	input: "Subject: =?WINDOWS-1252?B?RnJhbuehaXMgZXQgQWxsZW1hbmQ=?=",
		//	exp:   "Français et Allemand",
		//},
		//{
		//	input: "Subject: =?WINDOWS-1252?Q?Fran=E7ais_et_Espagnol_=A1Hola!?=",
		//	exp:   "Français et Espagnol ¡Hola!",
		//},
		//
		//// Windows-1253 (Greek)
		//{
		//	input: "Subject: =?WINDOWS-1253?B?RWxsaG5pa+EgZ2zOjHNzYQ==?=",
		//	exp:   "Ellhnikά glώssa",
		//},
		//
		//// Windows-1254 (Turkish)
		//{
		//	input: "Subject: =?WINDOWS-1254?B?VMO8cmvDp2UgRGVzdGXEn2k=?=",
		//	exp:   "Türkçe Desteği",
		//},
		//
		//// Windows-1255 (Hebrew)
		//{
		//	input: "Subject: =?WINDOWS-1255?B?5eXp6ent?=",
		//	exp:   "עברית",
		//},
		//
		//// Windows-1256 (Arabic)
		//{
		//	input: "Subject: =?WINDOWS-1256?B?2KfZhNi52LHYqNmK2Kk=?=",
		//	exp:   "العربية",
		//},
		//
		//// Windows-1257 (Baltic)
		//{
		//	input: "Subject: =?WINDOWS-1257?B?TGF0dmlldJV1IHZhbG9kYQ==?=",
		//	exp:   "Latviešu valoda",
		//},
		//
		//// KOI8-R (Russian Cyrillic)
		//{
		//	input: "Subject: =?KOI8-R?B?0snFydTBztnOxc7JINDPz87J18XOyc8=?=",
		//	exp:   "Привет русский текст",
		//},
		//{
		//	input: "Subject: =?KOI8-R?Q?=D0=CB=CF=D7=CF_=D2=D5=D3=D3=CB=C9=CA?=",
		//	exp:   "Слово русский",
		//},
		//
		//// KOI8-U (Ukrainian Cyrillic)
		//{
		//	input: "Subject: =?KOI8-U?B?0snFydTBztkg8MXOyc7J0SDHxcjT09fFzsnPINfFzsnPgg==?=",
		//	exp:   "Привіт український текст ії",
		//},
		//
		//// Shift-JIS (Japanese)
		{
			input: "Subject: =?SHIFT-JIS?B?k/qWe4zqg2WDjINYg2eDYJN5DA==?=",
			exp:   "日本語テレストチ土",
		},
		//{
		//	input: "Subject: =?SHIFT-JIS?Q?=93=FA=96=7B=8C=EA=83=65=83=58=83=67=83=60=93=99?=",
		//	exp:   "日本語テストです",
		//},
		//
		//// EUC-JP (Japanese)
		//{
		//	input: "Subject: =?EUC-JP?B?xvzL3LjspcalraW5pcg=?=",
		//	exp:   "日本語テスト",
		//},
		//
		//// EUC-KR (Korean)
		//{
		//	input: "Subject: =?EUC-KR?B?sbi6z7nRwM3GrCDGyLzayer4sMkg7JWM66qF7JWE?=",
		//	exp:   "한국어 테스트 메시지입니다",
		//},
		//
		//// GB2312/GBK (Simplified Chinese)
		//{
		//	input: "Subject: =?GB2312?B?1tC5+rXExbM=?=",
		//	exp:   "中文测试",
		//},
		//{
		//	input: "Subject: =?GB2312?Q?=D6=D0=CE=C4=B2=E2=CA=D4?=",
		//	exp:   "中文测试",
		//},
		//
		//// Big5 (Traditional Chinese)
		//{
		//	input: "Subject: =?BIG5?B?pPs2OaluOUo=?=",
		//	exp:   "中文測試",
		//},
		//{
		//	input: "Subject: =?BIG5?Q?=A4=A4=A4=E5=B4=FA=B8=D5?=",
		//	exp:   "中文測試",
		//},
		//
		//// Mixed charsets in a single header
		//{
		//	input: "Subject: =?ISO-8859-1?Q?Fran=E7ais?= and =?WINDOWS-1251?B?0KDRg9GB0YHQutC40Lk=?= and =?BIG5?B?pPs2OQ==?=",
		//	exp:   "Français and Русский and 中文",
		//},
		//
		//// Realistic headers with mix of plain and encoded parts
		//{
		//	input: "Subject: Re: =?ISO-8859-2?Q?Odpov=ECd:?= Your inquiry about =?WINDOWS-1250?Q?produktov=E9_katalogy?=",
		//	exp:   "Re: Odpovĕd: Your inquiry about produktové katalogy",
		//},
		//
		//// Headers with spaces between encoded words
		//{
		//	input: "Subject: =?ISO-8859-1?Q?Caf=E9?= =?ISO-8859-1?Q?_des?= =?ISO-8859-1?Q?_Amis?=",
		//	exp:   "Café des Amis",
		//},
	}

	for i, tc := range testCases {

		body := tc.input + "\r\n\r\n" + "content"

		e := Envelope{
			Data: &Data{},
		}
		e.Data.WriteString(body)

		m, err := e.Mail()
		if err != nil {
			t.Error("cannot parse mail:", err)
			return
		}

		headers, err := m.Headers()
		if err != nil {
			t.Errorf("Test case %d failed with error: %v", i+1, err)
			continue
		}

		decoded := headers.Get("Subject")

		if decoded != tc.exp {
			t.Errorf("Test case %d failed:\nInput:    %s\nExpected: %s\nGot:      %s",
				i+1, tc.input, tc.exp, decoded)
		}
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
