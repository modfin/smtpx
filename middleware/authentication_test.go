package middleware

import (
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"github.com/modfin/smtpx/middleware/authres"
	"log/slog"
	"net"
	"net/mail"
	"strings"
	"testing"
)

func TestAddAuthenticationResult(t *testing.T) {
	// Create a mock envelope
	e := &envelope.Envelope{
		RemoteAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1")},
		Helo:       "test.com",
		MailFrom:   &mail.Address{Address: "sender@example.com"},
		Data:       &envelope.Data{},
	}
	_, err := e.Data.WriteString("From: sender@example.com\r\nSubject: Test\r\n\r\nTest message")
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock next handler that just returns a success response
	nextHandler := func(e *envelope.Envelope) smtpx.Response {
		return smtpx.NewResponse(250, "OK")
	}

	// Create the middleware
	slog.SetLogLoggerLevel(slog.LevelDebug)
	middleware := AddAuthenticationResult("example.com", slog.Default())

	// Apply the middleware to our next handler
	handler := middleware(nextHandler)

	// Call the handler with our envelope
	resp := handler(e)

	// Check that the response is passed through correctly
	if resp.StatusCode() != 250 {
		t.Errorf("Expected response {250 OK}, got %v", resp)
	}

	// Check that the Authentication-Results header was added
	m, err := e.Mail()
	if err != nil {
		t.Errorf("Expected envelope to be a mail message, got %v", err)
	}
	header, err := m.Headers()
	if err != nil {
		t.Errorf("Expected Authentication-Result header to be added, but it wasn't")
	}

	headerVal := header.Get("Authentication-Results")
	// Verify the header contains the hostname
	if !strings.Contains(headerVal, "example.com") {
		t.Errorf("Expected Authentication-Result to contain hostname, got %s", headerVal)
	}

	// Verify the header contains SPF results
	if !strings.Contains(headerVal, "spf=") {
		t.Errorf("Expected Authentication-Result to contain SPF results, got %s", headerVal)
	}

	id, res, err := authres.Parse(headerVal)
	if err != nil {
		t.Errorf("Expected no error when parsing header, got: %v", err)
	}
	if id != "example.com" {
		t.Errorf("Expected identifier to be %q, but got %q", "example.com", id)
	}
	if len(res) != 2 {
		t.Errorf("Expected number of results to be %v, but got %v", 2, len(res))
	}
}

func TestSPFCheck(t *testing.T) {
	tests := []struct {
		name       string
		ip         string
		helo       string
		from       string
		wantResult authres.ResultValue
		wantReason string
	}{
		{
			name:       "SPF Pass",
			ip:         "89.160.105.250",
			helo:       "modfin.se",
			from:       "test@modfin.se",
			wantResult: authres.ResultPass,
			wantReason: "matched ip",
		},
		{
			name:       "SPF Nested Pass",
			ip:         "198.2.128.2",
			helo:       "modularfinance.se",
			from:       "test@modularfinance.se",
			wantResult: authres.ResultPass,
			wantReason: "matched ip",
		},
		{
			name:       "SPF Nested MX Pass",
			ip:         "46.21.101.51",
			helo:       "modularfinance.se",
			from:       "test@modularfinance.se",
			wantResult: authres.ResultPass,
			wantReason: "matched mx",
		},
		{
			name:       "SPF Fail",
			ip:         "127.0.0.1",
			helo:       "example.com",
			from:       "test@example.com",
			wantResult: authres.ResultFail,
			wantReason: "matched all",
		},
		{
			name:       "SPF SoftFail",
			ip:         "127.0.0.1",
			helo:       "modfin.se",
			from:       "test@modfin.se",
			wantResult: authres.ResultSoftFail,
			wantReason: "matched all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &envelope.Envelope{
				RemoteAddr: &net.TCPAddr{IP: net.ParseIP(tt.ip), Port: 12345},
				Helo:       tt.helo,
				MailFrom:   &mail.Address{Address: tt.from},
			}

			result := spfCheck(e)

			if result.Value != tt.wantResult {
				t.Errorf("spfCheck() result = %v, want %v", result.Value, tt.wantResult)
			}

			if result.Reason != tt.wantReason {
				t.Errorf("spfCheck() reason = %v, want %v", result.Reason, tt.wantReason)
			}
		})
	}
}
