package middleware

import (
	"github.com/crholm/brevx"
	"github.com/crholm/brevx/envelope"
	"github.com/crholm/brevx/middleware/authres"
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
	nextHandler := func(e *envelope.Envelope) brevx.Response {
		return brevx.NewResponse(250, "OK")
	}

	// Create the middleware
	slog.SetLogLoggerLevel(slog.LevelDebug)
	middleware := AddAuthenticationResult("example.com", slog.Default())

	// Apply the middleware to our next handler
	handler := middleware(nextHandler)

	// Call the handler with our envelope
	resp := handler(e)

	// Check that the response is passed through correctly
	if resp.Code() != 250 {
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
