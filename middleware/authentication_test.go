package middleware

import (
	"fmt"
	"github.com/crholm/brevx"
	"github.com/crholm/brevx/envelope"
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
	middleware := AddAuthenticationResult("example.com")

	// Apply the middleware to our next handler
	handler := middleware(nextHandler)

	// Call the handler with our envelope
	resp := handler(e)

	// Check that the response is passed through correctly
	if resp.Code() != 250 {
		t.Errorf("Expected response {250 OK}, got %v", resp)
	}

	fmt.Println(e.Data.String())
	// Check that the Authentication-Results header was added
	header, err := e.Headers()
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
}
