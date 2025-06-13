package envelope

import (
	"net/textproto"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMailBodyAndParseContent(t *testing.T) {
	// Test case 1: Simple plain text email
	t.Run("Plain Text Email", func(t *testing.T) {
		rawEmail := []byte(`From: sender@example.com
To: recipient@example.com
Subject: Test Email
Content-Type: text/plain

This is a test email body.`)

		mail, err := NewMail(rawEmail, false)
		require.NoError(t, err)

		content, err := mail.Body()
		require.NoError(t, err)

		assert.Equal(t, "text/plain", content.Headers.Get("Content-Type"))
		assert.Equal(t, "This is a test email body.", string(content.Body))
		assert.Empty(t, content.Children)
	})

	// Test case 2: Multipart email
	t.Run("Multipart Email", func(t *testing.T) {
		rawEmail := []byte(`From: sender@example.com
To: recipient@example.com
Subject: Multipart Test Email
Content-Type: multipart/mixed; boundary="boundary123"

--boundary123
Content-Type: text/plain

This is the plain text part.
--boundary123
Content-Type: text/html

<html><body>This is the HTML part.</body></html>
--boundary123--`)

		mail, err := NewMail(rawEmail, false)
		require.NoError(t, err)

		content, err := mail.Body()
		require.NoError(t, err)

		assert.Equal(t, "multipart/mixed; boundary=\"boundary123\"", content.Headers.Get("Content-Type"))
		assert.Empty(t, content.Body)
		require.Len(t, content.Children, 2)

		assert.Equal(t, "text/plain", content.Children[0].Headers.Get("Content-Type"))
		assert.Equal(t, "This is the plain text part.", string(content.Children[0].Body))

		assert.Equal(t, "text/html", content.Children[1].Headers.Get("Content-Type"))
		assert.Equal(t, "<html><body>This is the HTML part.</body></html>", string(content.Children[1].Body))
	})

	// Test case 3: Nested multipart email
	t.Run("Nested Multipart Email", func(t *testing.T) {
		rawEmail := []byte(`From: sender@example.com
To: recipient@example.com
Subject: Nested Multipart Test Email
Content-Type: multipart/mixed; boundary="outer"

--outer
Content-Type: multipart/alternative; boundary="inner"

--inner
Content-Type: text/plain

Plain text version
--inner
Content-Type: text/html

<html><body>HTML version</body></html>
--inner--
--outer
Content-Type: application/pdf
Content-Disposition: attachment; filename="test.pdf"

PDF content here
--outer--`)

		mail, err := NewMail(rawEmail, false)
		require.NoError(t, err)

		content, err := mail.Body()
		require.NoError(t, err)

		assert.Equal(t, "multipart/mixed; boundary=\"outer\"", content.Headers.Get("Content-Type"))
		assert.Empty(t, content.Body)
		require.Len(t, content.Children, 2)

		// Check the nested multipart/alternative part
		assert.Equal(t, "multipart/alternative; boundary=\"inner\"", content.Children[0].Headers.Get("Content-Type"))
		assert.Empty(t, content.Children[0].Body)
		require.Len(t, content.Children[0].Children, 2)

		assert.Equal(t, "text/plain", content.Children[0].Children[0].Headers.Get("Content-Type"))
		assert.Equal(t, "Plain text version", string(content.Children[0].Children[0].Body))

		assert.Equal(t, "text/html", content.Children[0].Children[1].Headers.Get("Content-Type"))
		assert.Equal(t, "<html><body>HTML version</body></html>", string(content.Children[0].Children[1].Body))

		// Check the attachment part
		assert.Equal(t, "application/pdf", content.Children[1].Headers.Get("Content-Type"))
		assert.Equal(t, "attachment; filename=\"test.pdf\"", content.Children[1].Headers.Get("Content-Disposition"))
		assert.Equal(t, "PDF content here", string(content.Children[1].Body))
	})
}

func TestContentIsAttachment(t *testing.T) {
	testCases := []struct {
		name     string
		headers  textproto.MIMEHeader
		expected bool
	}{
		{
			name: "Attachment",
			headers: textproto.MIMEHeader{
				"Content-Disposition": []string{"attachment"},
			},
			expected: true,
		},
		{
			name: "Attachment with filename",
			headers: textproto.MIMEHeader{
				"Content-Disposition": []string{"attachment; filename=\"document.pdf\""},
			},
			expected: true,
		},
		{
			name: "Attachment with bad headers",
			headers: textproto.MIMEHeader{
				"Content-Disposition": []string{"attachment; file"},
			},
			expected: true,
		},
		{
			name: "Inline",
			headers: textproto.MIMEHeader{
				"Content-Disposition": []string{"inline"},
			},
			expected: false,
		},
		{
			name: "No Content-Disposition",
			headers: textproto.MIMEHeader{
				"Content-Type": []string{"text/plain"},
			},
			expected: false,
		},
		{
			name: "Mixed case",
			headers: textproto.MIMEHeader{
				"Content-Disposition": []string{"AttAchMent"},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := &Content{
				Headers: tc.headers,
			}
			result := content.IsAttachment()
			assert.Equal(t, tc.expected, result, "IsAttachment() returned unexpected result")
		})
	}
}

func TestAttachmentName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Simple filename",
			input:    `attachment; filename="example.pdf"`,
			expected: "example.pdf",
			wantErr:  false,
		},
		{
			name:     "Filename with spaces",
			input:    `attachment; filename="my document.pdf"`,
			expected: "my document.pdf",
			wantErr:  false,
		},
		{
			name:     "UTF-8 encoded filename",
			input:    `attachment; filename*=UTF-8''%E8%AF%95%E9%AA%8C.pdf`,
			expected: "试验.pdf",
			wantErr:  false,
		},
		{
			name:     "Percent-encoding filename",
			input:    `attachment; filename="my%20file.pdf"`,
			expected: "my file.pdf",
			wantErr:  false,
		},
		{
			name:    "Invalid format",
			input:   `attachment; filename`,
			wantErr: true,
		},
		{
			name:    "Empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := &Content{
				Headers: textproto.MIMEHeader{
					"Content-Disposition": []string{tt.input},
				},
			}

			a := &AttachmentPart{c: content}

			got, err := a.Filename()
			if (err != nil) != tt.wantErr {
				t.Errorf("Filename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("Filename() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestContent_IsInline(t *testing.T) {
	tests := []struct {
		name        string
		disposition string
		want        bool
	}{
		{
			name:        "Inline disposition",
			disposition: "inline",
			want:        true,
		},
		{
			name:        "Inline disposition with parameters",
			disposition: `inline; filename="test.pdf"`,
			want:        true,
		},
		{
			name:        "Inline disposition uppercase",
			disposition: "INLINE",
			want:        true,
		},
		{
			name:        "Attachment disposition",
			disposition: "attachment",
			want:        false,
		},
		{
			name:        "Attachment disposition with parameters",
			disposition: `attachment; filename="test.pdf"`,
			want:        false,
		},
		{
			name:        "Empty disposition",
			disposition: "",
			want:        false,
		},
		{
			name:        "Invalid disposition",
			disposition: "invalid-type",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Content{
				Headers: textproto.MIMEHeader{
					"Content-Disposition": []string{tt.disposition},
				},
			}
			if got := c.IsInline(); got != tt.want {
				t.Errorf("Content.IsInline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContent_AsInline(t *testing.T) {
	tests := []struct {
		name        string
		disposition string
		wantInline  bool
		wantErr     bool
	}{
		{
			name:        "Valid inline content",
			disposition: "inline",
			wantInline:  true,
			wantErr:     false,
		},
		{
			name:        "Inline with filename",
			disposition: `inline; filename="test.jpg"`,
			wantInline:  true,
			wantErr:     false,
		},
		{
			name:        "Attachment content",
			disposition: "attachment",
			wantInline:  false,
			wantErr:     true,
		},
		{
			name:        "No disposition",
			disposition: "",
			wantInline:  false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Content{
				Headers: textproto.MIMEHeader{
					"Content-Disposition": []string{tt.disposition},
				},
			}
			got, err := c.AsInline()
			if (err != nil) != tt.wantErr {
				t.Errorf("Content.AsInline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got != nil) != tt.wantInline {
				t.Errorf("Content.AsInline() = %v, want %v", got, tt.wantInline)
			}
		})
	}
}

func TestInlinePart_Filename(t *testing.T) {
	tests := []struct {
		name         string
		disposition  string
		wantFilename string
		wantErr      bool
	}{
		{
			name:         "Inline with filename",
			disposition:  `inline; filename="test.jpg"`,
			wantFilename: "test.jpg",
			wantErr:      false,
		},
		{
			name:         "Inline without filename",
			disposition:  "inline",
			wantFilename: "",
			wantErr:      true,
		},
		{
			name:         "Inline with UTF-8 filename",
			disposition:  `inline; filename*=UTF-8''%E6%B5%8B%E8%AF%95.jpg`,
			wantFilename: "测试.jpg",
			wantErr:      false,
		},
		{
			name:         "Inline with percent-encoded filename",
			disposition:  `inline; filename="test%20file.jpg"`,
			wantFilename: "test file.jpg",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Content{
				Headers: textproto.MIMEHeader{
					"Content-Disposition": []string{tt.disposition},
				},
			}
			inlinePart, err := c.AsInline()
			if err != nil {
				t.Fatalf("Failed to create InlinePart: %v", err)
			}

			got, err := inlinePart.Filename()
			if (err != nil) != tt.wantErr {
				t.Errorf("InlinePart.Filename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantFilename {
				t.Errorf("InlinePart.Filename() = %v, want %v", got, tt.wantFilename)
			}
		})
	}
}

func TestFormPart_Name(t *testing.T) {
	tests := []struct {
		name        string
		disposition string
		wantName    string
		wantErr     bool
	}{
		{
			name:        "Valid form part name",
			disposition: `form-data; name="fieldName"`,
			wantName:    "fieldName",
			wantErr:     false,
		},
		{
			name:        "Form part with name and filename",
			disposition: `form-data; name="uploadFile"; filename="test.jpg"`,
			wantName:    "uploadFile",
			wantErr:     false,
		},
		{
			name:        "Form part with quoted name",
			disposition: `form-data; name="field Name With Spaces"`,
			wantName:    "field Name With Spaces",
			wantErr:     false,
		},
		{
			name:        "Missing name parameter",
			disposition: `form-data; filename="test.jpg"`,
			wantName:    "",
			wantErr:     true,
		},
		{
			name:        "Empty name parameter",
			disposition: `form-data; name=""`,
			wantName:    "",
			wantErr:     true,
		},
		{
			name:        "Invalid Content-Disposition",
			disposition: "invalid disposition",
			wantName:    "",
			wantErr:     true,
		},
		{
			name:        "Empty Content-Disposition",
			disposition: "",
			wantName:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formPart := &FormPart{
				c: &Content{
					Headers: textproto.MIMEHeader{
						"Content-Disposition": []string{tt.disposition},
					},
				},
			}

			gotName, err := formPart.Name()
			if (err != nil) != tt.wantErr {
				t.Errorf("FormPart.Name() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotName != tt.wantName {
				t.Errorf("FormPart.Name() = %v, want %v", gotName, tt.wantName)
			}
		})
	}
}

func TestContent_Decode(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		input    []byte
		want     []byte
		wantErr  bool
	}{
		{
			name:     "7bit encoding",
			encoding: "7bit",
			input:    []byte("This is a 7bit encoded text."),
			want:     []byte("This is a 7bit encoded text."),
			wantErr:  false,
		},
		{
			name:     "8bit encoding",
			encoding: "8bit",
			input:    []byte("This is an 8bit encoded text with special characters: ñ, é, ß"),
			want:     []byte("This is an 8bit encoded text with special characters: ñ, é, ß"),
			wantErr:  false,
		},
		{
			name:     "binary encoding",
			encoding: "binary",
			input:    []byte{0x00, 0x01, 0x02, 0x03, 0xFF},
			want:     []byte{0x00, 0x01, 0x02, 0x03, 0xFF},
			wantErr:  false,
		},
		{
			name:     "quoted-printable encoding",
			encoding: "quoted-printable",
			input:    []byte("This is a quoted-printable encoded text with special characters: =C3=B1, =C3=A9, =C3=9F"),
			want:     []byte("This is a quoted-printable encoded text with special characters: ñ, é, ß"),
			wantErr:  false,
		},
		{
			name:     "base64 encoding",
			encoding: "base64",
			input:    []byte("VGhpcyBpcyBhIGJhc2U2NCBlbmNvZGVkIHRleHQu"),
			want:     []byte("This is a base64 encoded text."),
			wantErr:  false,
		},
		{
			name:     "invalid base64 encoding",
			encoding: "base64",
			input:    []byte("This is not a valid base64 encoded text"),
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "unknown encoding",
			encoding: "unknown",
			input:    []byte("This is some text with unknown encoding."),
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Content{
				Headers: textproto.MIMEHeader{
					"Content-Type":              []string{"text/plain"},
					"Content-Transfer-Encoding": []string{tt.encoding},
				},
				Body: tt.input,
			}
			got, err := c.Decode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Content.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Content.Decode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContent_Decode_Charsets(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		encoding    string
		input       []byte
		expected    string
		wantErr     bool
	}{
		{
			name:        "UTF-8 with 7bit encoding",
			contentType: "text/plain; charset=utf-8",
			encoding:    "7bit",
			input:       []byte("Hello, 世界"),
			expected:    "Hello, 世界",
		},
		{
			name:        "ISO-8859-1 with quoted-printable encoding",
			contentType: "text/plain; charset=iso-8859-1",
			encoding:    "quoted-printable",
			input:       []byte("H=E9llo, w=F6rld"),
			expected:    "Héllo, wörld",
		},
		{
			name:        "UTF-8 with base64 encoding",
			contentType: "text/plain; charset=utf-8",
			encoding:    "base64",
			input:       []byte("SGVsbG8sIHdvcmxk"),
			expected:    "Hello, world",
		},
		{
			name:        "Windows-1252 with 8bit encoding",
			contentType: "text/plain; charset=windows-1252",
			encoding:    "8bit",
			input:       []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F, 0x2C, 0x20, 0xF6, 0x72, 0x6C, 0x64},
			expected:    "Hello, örld",
		},
		{
			name:        "UTF-16BE with base64 encoding",
			contentType: "text/plain; charset=utf-16be",
			encoding:    "base64",
			input:       []byte("AGgAZQBsAGwAbwAsACAAdwBvAHIAbABk"),
			expected:    "hello, world",
		},
		{
			name:        "Invalid charset",
			contentType: "text/plain; charset=invalid",
			encoding:    "7bit",
			input:       []byte("Hello, world"),
			expected:    "Hello, world",
			wantErr:     false,
		},
		{
			name:        "Invalid encoding",
			contentType: "text/plain; charset=utf-8",
			encoding:    "invalid",
			input:       []byte("Hello, world"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Content{
				Headers: textproto.MIMEHeader{
					"Content-Type":              {tt.contentType},
					"Content-Transfer-Encoding": {tt.encoding},
				},
				Body: tt.input,
			}

			decoded, err := c.Decode()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, string(decoded))
			}
		})
	}
}

func TestHeadersLiteral_NotDecoding(t *testing.T) {
	tests := []struct {
		name            string
		value           string
		expectedRaw     string
		expectedDecoded string
		expectedEmail   string
	}{
		{
			name:            "Latin1",
			value:           "=?iso-8859-1?Q?Lastname=2C_=F6?= <o.Lastname@company.com>",
			expectedRaw:     "=?iso-8859-1?Q?Lastname=2C_=F6?= <o.Lastname@company.com>",
			expectedDecoded: "\"Lastname, ö\" <o.Lastname@company.com>",
			expectedEmail:   "o.Lastname@company.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawEmail := []byte(`From: ` + tt.value + `
To: recipient@example.com
Subject: Test Email
Content-Type: text/plain

This is a test email body.`)

			m, err := NewMail(rawEmail, false)
			require.NoError(t, err)

			// Test HeadersLiteral - should not decode
			headersLiteral, err := m.Headers(WithLiteral())
			assert.NoError(t, err)

			address, err := headersLiteral.From()

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEmail, address.Address)

			// Test Headers - should decode
			headers, err := m.Headers()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedDecoded, headers.Get("From"))
		})
	}
}
