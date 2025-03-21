package main

import (
	"fmt"
	"github.com/modfin/smtpx/envelope"
	"strings"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {

	mail, err := envelope.NewMail([]byte(exampleEmail), true)
	check(err)

	headers, err := mail.Headers()
	check(err)
	for k, v := range headers {
		fmt.Printf("%s: %s\n", k, v)
	}

	// From: [sender@example.com]
	// To: [recipient@example.com]
	// Subject: [Multipart Email with Spécial Chàracters]
	// Date: [Mon, 15 May 2023 10:00:00 -0700]
	// Mime-Version: [1.0]
	// Content-Type: [multipart/mixed; boundary="outer-boundary"]

	//  Content is tree
	content, err := mail.Body()
	check(err)

	content.Walk(func(c *envelope.Content, level int) error {

		fmt.Print(strings.Repeat("│ ", level))
		fmt.Printf("Content-Type: %s\n", c.Headers.Get("Content-Type"))
		if c.Leaf() {
			fmt.Printf("%s╰ len: %d\n", strings.Repeat("│ ", level), len(c.Body))
		}
		return nil
	})
	// Content-Type: multipart/mixed; boundary="outer-boundary"
	// │ Content-Type: multipart/related; boundary="related-boundary"
	// │ │ Content-Type: multipart/alternative; boundary="inner-boundary"
	// │ │ │ Content-Type: text/plain; charset="UTF-8"
	// │ │ │ ╰ len: 219
	// │ │ │ Content-Type: text/html; charset="UTF-8"
	// │ │ │ ╰ len: 345
	// │ │ Content-Type: image/png
	// │ │ ╰ len: 308
	// │ Content-Type: application/pdf
	// │ ╰ len: 178
	//

	// Flatten and filter things that has content, ie, all multipart parts is removed
	for _, c := range content.Flatten() {
		fmt.Printf("Content-Type: %s\n", c.Headers.Get("Content-Type"))
	}
	// Content-Type: text/plain; charset="UTF-8"
	// Content-Type: text/html; charset="UTF-8"
	// Content-Type: image/png
	// Content-Type: application/pdf

}

var exampleEmail = `From: sender@example.com
To: recipient@example.com
Subject: =?UTF-8?Q?Multipart_Email_with_Sp=C3=A9cial_Ch=C3=A0racters?=
Date: Mon, 15 May 2023 10:00:00 -0700
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="outer-boundary"

--outer-boundary
Content-Type: multipart/related; boundary="related-boundary"

--related-boundary
Content-Type: multipart/alternative; boundary="inner-boundary"

--inner-boundary
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: 7bit

This is the plain text version of the email.
It contains a simple message without any formatting.
Here are some non-ASCII characters: ñ, é, ß, 你好
(An inline image is displayed in the HTML version of this email.)

--inner-boundary
Content-Type: text/html; charset="UTF-8"
Content-Transfer-Encoding: 7bit

<html>
<body>
<h1>HTML Version with Inline Image</h1>
<p>This is the <strong>HTML version</strong> of the email.</p>
<p>It contains <em>formatted text</em> and can include other HTML elements.</p>
<p>Non-ASCII characters: ñ, é, ß, 你好</p>
<p>Here's an inline image:</p>
<img src="cid:unique-image-id" alt="Inline image" />
</body>
</html>

--inner-boundary--

--related-boundary
Content-Type: image/png
Content-Transfer-Encoding: base64
Content-ID: <unique-image-id>
Content-Disposition: inline; filename="image.png"

iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAACXBIWXMAAAsTAAALEwEAmpwYAAAA
B3RJTUUH1QEHDxEhOnxCRgAAAAd0RVh0QXV0aG9yAKmuzEgAAAAMdEVYdERlc2NyaXB0aW9uABMJ
ISMAAAAKdEVYdENvcHlyaWdodACsD8w6AAAADnRFWHRDcmVhdGlvbiB0aW1lADX3DwkAAAAJdEVY
dFNvZnR3YXJlAF1w/zoAAAALdEVYdERpc2NsYWltZXIAt8C0jwAAAAh0RVh0V2FybmluZwDAG+aH

--related-boundary--

--outer-boundary
Content-Type: application/pdf
Content-Disposition: attachment; filename="document.pdf"
Content-Transfer-Encoding: base64

JVBERi0xLjMNCiXi48/TDQoNCjEgMCBvYmoNCjw8DQovVHlwZSAvQ2F0YWxvZw0KL091dGxpbmVzIDIgMCBSDQov
UGFnZXMgMyAwIFINCj4+DQplbmRvYmoNCg0KMiAwIG9iag0KPDwNCi9UeXBlIC9PdXRsaW5lcw0KL0NvdW50IDAn

--outer-boundary--`
