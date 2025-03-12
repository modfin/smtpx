package brevx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/crholm/brevx/envelope"
	"github.com/crholm/brevx/mocks"
	"log/slog"
	"net"
	"net/smtp"
	"os"
	"sync"
	"testing"
	"time"
)

func StartTLSServer(inf string, t *testing.T) (<-chan *envelope.Envelope, *Server, *x509.CertPool) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	hostname := "example.com"

	rootCert, rootKey, err := mocks.GenerateRootCA()

	if err != nil {
		t.Fatal(err)
	}

	tlscfg, err := mocks.CreateTLSConfigWithCA(hostname, rootCert, rootKey)
	if err != nil {
		t.Fatal(err)
	}

	mails := make(chan *envelope.Envelope, 10)
	s := &Server{
		Hostname: hostname,
		Logger:   logger,
		Addr:     inf,
		Backend: BackendFunc(func(e *envelope.Envelope) Result {
			mails <- e
			return NewResult(250, "Added to spool")
		}),
		TLSConfig: tlscfg,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil {
			fmt.Println(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	return mails, s, mocks.RootCAPool(rootCert)
}

func StartServer(inf string) (<-chan *envelope.Envelope, *Server) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	mails := make(chan *envelope.Envelope, 10)
	s := &Server{
		Logger: logger,
		Addr:   inf,
		Backend: BackendFunc(func(e *envelope.Envelope) Result {
			mails <- e
			return NewResult(250, "OK")
		}),
	}

	go func() {
		if err := s.ListenAndServe(); err != nil {
			fmt.Println(err)
		}
	}()

	return mails, s
}

func TestTLS(t *testing.T) {
	inf := ":2525"
	mails, s, certPool := StartTLSServer(inf, t)

	from := "sender@example.com"
	to := "recipient@example.com"
	// Send an email to the server
	msg := fmt.Sprint("Subject: Test Email\n")
	msg += "From: sender@example.com\n"
	msg += "To: recipient@example.com\n"
	msg += "\n\n"
	msg += "Hello, this is a test email.\n"

	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		ServerName: s.Hostname,
	}

	// Connect to the SMTP server
	conn, err := net.Dial("tcp", inf)
	if err != nil {
		t.Fatalf("failed to connect to SMTP server: %v", err)
	}

	// Create an SMTP client
	c, err := smtp.NewClient(conn, s.Hostname)
	if err != nil {
		t.Fatalf("failed to create SMTP client: %v", err)
	}
	defer c.Close()

	if err := c.StartTLS(tlsConfig); err != nil {
		t.Fatalf("failed to start TLS: %v", err)
	}

	// Set the sender and recipient
	if err := c.Mail(from); err != nil {
		t.Fatalf("failed to set sender: %v", err)
	}
	if err := c.Rcpt(to); err != nil {
		t.Fatalf("failed to set recipient: %v", err)
	}

	w, err := c.Data()
	if err != nil {
		t.Fatalf("failed to start data transfer: %v", err)
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		t.Fatalf("failed to write data: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close data transfer: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatal(err)
	}

	e := <-mails

	if e.MailFrom.Address != "sender@example.com" {
		t.Fatalf("expected %s, got %s", "sender@example.com", e.MailFrom.Address)
	}

	if len(e.RcptTo) != 1 {
		t.Fatalf("expected 1 recipient, got %d", len(e.RcptTo))
	}

	if e.RcptTo[0].Address != "recipient@example.com" {
		t.Fatalf("expected %s, got %s", "recipient@example.com", e.RcptTo[0].Address)
	}

	if e.Data.String() != msg {
		t.Fatalf("expected %s, got %s", msg, e.Data.String())
	}

	// Shut down the server
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

}

func TestStartStop(t *testing.T) {

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	s := &Server{
		Logger: logger,
		Backend: BackendFunc(func(e *envelope.Envelope) Result {
			return NewResult(250, "OK")
		}),
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := s.ListenAndServe(); err != nil {
			t.Fatal(err)
		}
		wg.Done()
	}()

	time.Sleep(100 * time.Millisecond)

	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}
func TestStartStopTimout(t *testing.T) {
	inf := ":2525"
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	s := Server{
		Logger: logger,
		Backend: BackendFunc(func(e *envelope.Envelope) Result {
			return NewResult(250, "OK")
		}),
		Addr: inf,
	}

	//ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := s.ListenAndServe(); err != nil {
			t.Fatal(err)
		}
		wg.Done()
	}()

	// Allow the server to start
	time.Sleep(100 * time.Millisecond)

	go func() {
		conn, err := net.Dial("tcp", inf)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(500 * time.Millisecond)
		conn.Close()
	}()

	// Allow the client to connect
	time.Sleep(100 * time.Millisecond)

	fmt.Println("shutting down")
	timeout, cancelTimeout := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancelTimeout()
	if err := s.Shutdown(timeout); err == nil {
		t.Fatal("expected error")
	}
	wg.Wait()
}

func TestSendEmail(t *testing.T) {
	// Create a new server

	envelopes := make(chan *envelope.Envelope, 1)

	s := Server{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
		Backend: BackendFunc(func(e *envelope.Envelope) Result {
			envelopes <- e
			return NewResult(250, "OK")
		}),
		Addr: ":2525",
	}

	// Start the server in a goroutine
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := s.ListenAndServe(); err != nil {
			t.Fatal(err)
		}
		wg.Done()
	}()

	// Wait for the server to start
	time.Sleep(100 * time.Millisecond)

	// Send an email to the server
	msg := fmt.Sprint("Subject: Test Email\n")
	msg += "From: sender@example.com\n"
	msg += "To: recipient@example.com\n"
	msg += "\n\n"
	msg += "Hello, this is a test email.\n"

	auth := smtp.Auth(nil)
	err := smtp.SendMail("localhost:2525", auth, "sender@example.com", []string{"recipient@example.com"}, []byte(msg))
	if err != nil {
		t.Fatal(err)
	}

	e := <-envelopes

	if e.MailFrom.Address != "sender@example.com" {
		t.Fatalf("expected %s, got %s", "sender@example.com", e.MailFrom.Address)
	}

	if len(e.RcptTo) != 1 {
		t.Fatalf("expected 1 recipient, got %d", len(e.RcptTo))
	}

	if e.RcptTo[0].Address != "recipient@example.com" {
		t.Fatalf("expected %s, got %s", "recipient@example.com", e.RcptTo[0].Address)
	}

	if e.Data.String() != msg {
		t.Fatalf("expected %s, got %s", msg, e.Data.String())
	}

	// Shut down the server
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestTLSMultiple(t *testing.T) {
	inf := ":2525"
	mails, s, certPool := StartTLSServer(inf, t)

	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		ServerName: s.Hostname,
	}

	// Connect to the SMTP server
	conn, err := net.Dial("tcp", inf)
	if err != nil {
		t.Fatalf("failed to connect to SMTP server: %v", err)
	}

	// Create an SMTP client
	c, err := smtp.NewClient(conn, s.Hostname)
	if err != nil {
		t.Fatalf("failed to create SMTP client: %v", err)
	}
	defer c.Close()

	if err := c.StartTLS(tlsConfig); err != nil {
		t.Fatalf("failed to start TLS: %v", err)
	}

	// Sending Email 1

	from := "sender@example.com"
	to := "recipient@example.com"
	// Send an email to the server
	msg := fmt.Sprint("Subject: Test Email\n")
	msg += "From: sender@example.com\n"
	msg += "To: recipient@example.com\n"
	msg += "\n\n"
	msg += "Hello, this is a test email.\n"

	if err := c.Mail(from); err != nil {
		t.Fatalf("failed to set sender: %v", err)
	}
	if err := c.Rcpt(to); err != nil {
		t.Fatalf("failed to set recipient: %v", err)
	}

	w, err := c.Data()
	if err != nil {
		t.Fatalf("failed to start data transfer: %v", err)
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		t.Fatalf("failed to write data: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close data transfer: %v", err)
	}

	// Sending Email 2
	if err := c.Reset(); err != nil {
		t.Fatalf("failed to reset connection: %v", err)
	}

	from2 := "sender-2@example.com"
	to2 := "recipient-2@example.com"
	// Send an email to the server
	msg2 := fmt.Sprint("Subject: Test Email 2\n")
	msg2 += "From: sender-2@example.com\n"
	msg2 += "To: recipient-2@example.com\n"
	msg2 += "\n\n"
	msg2 += "Hello, this is a test email.\n"

	if err := c.Mail(from2); err != nil {
		t.Fatalf("failed to set sender: %v", err)
	}
	if err := c.Rcpt(to2); err != nil {
		t.Fatalf("failed to set recipient: %v", err)
	}

	w, err = c.Data()
	if err != nil {
		t.Fatalf("failed to start data transfer: %v", err)
	}

	_, err = w.Write([]byte(msg2))
	if err != nil {
		t.Fatalf("failed to write data: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close data transfer: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatal(err)
	}

	// Check the first email sent
	e := <-mails

	if e.MailFrom.Address != "sender@example.com" {
		t.Fatalf("expected %s, got %s", "sender@example.com", e.MailFrom.Address)
	}
	if len(e.RcptTo) != 1 {
		t.Fatalf("expected 1 recipient, got %d", len(e.RcptTo))
	}
	if e.RcptTo[0].Address != "recipient@example.com" {
		t.Fatalf("expected %s, got %s", "recipient@example.com", e.RcptTo[0].Address)
	}
	if e.Data.String() != msg {
		t.Fatalf("expected %s, got %s", msg, e.Data.String())
	}

	// Check the second email sent
	e = <-mails

	if e.MailFrom.Address != from2 {
		t.Fatalf("expected %s, got %s", from2, e.MailFrom.Address)
	}
	if len(e.RcptTo) != 1 {
		t.Fatalf("expected 1 recipient, got %d", len(e.RcptTo))
	}
	if e.RcptTo[0].Address != to2 {
		t.Fatalf("expected %s, got %s", "recipient@example.com", e.RcptTo[0].Address)
	}
	if e.Data.String() != msg2 {
		t.Fatalf("expected %s, got %s", msg2, e.Data.String())
	}

	// Shut down the server
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

}
