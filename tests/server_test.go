package tests

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"github.com/modfin/smtpx/middleware"
	"github.com/modfin/smtpx/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const hostname = "example.com"

func StartTLSServer(inf string, t *testing.T, middlewares ...smtpx.Middleware) (<-chan *envelope.Envelope, *smtpx.Server, *x509.CertPool) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	rootCert, rootKey, err := mocks.GenerateRootCA()

	if err != nil {
		t.Fatal(err)
	}

	tlscfg, err := mocks.CreateTLSConfigWithCA(hostname, rootCert, rootKey)
	if err != nil {
		t.Fatal(err)
	}

	mails := make(chan *envelope.Envelope, 10)
	s := &smtpx.Server{
		Hostname:    hostname,
		Logger:      logger,
		Addr:        inf,
		Middlewares: middlewares,
		Handler: smtpx.NewHandler(func(e *envelope.Envelope) smtpx.Response {
			mails <- e
			return nil
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

func StartServer(inf string) (<-chan *envelope.Envelope, *smtpx.Server) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	mails := make(chan *envelope.Envelope, 10)
	s := &smtpx.Server{
		Logger: logger,
		Addr:   inf,
		Handler: smtpx.NewHandler(func(e *envelope.Envelope) smtpx.Response {
			mails <- e
			return smtpx.NewResponse(250, "OK")
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
	s := &smtpx.Server{
		Logger: logger,
		Handler: smtpx.NewHandler(func(e *envelope.Envelope) smtpx.Response {
			return smtpx.NewResponse(250, "OK")
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
	s := smtpx.Server{
		Logger: logger,
		Handler: smtpx.NewHandler(func(e *envelope.Envelope) smtpx.Response {
			return smtpx.NewResponse(250, "OK")
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

	s := smtpx.Server{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
		Handler: smtpx.NewHandler(func(e *envelope.Envelope) smtpx.Response {
			envelopes <- e
			return smtpx.NewResponse(250, "OK")
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

	// Terminate the connection
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

func TestSMTPServer(t *testing.T) {
	// Setup - generate a test CA and server certificate
	addr := ":2525"
	// Create an SMTP server for testing
	inbox, server, certPool := StartTLSServer(addr, t)
	defer server.Shutdown(context.Background())

	// Get the server address
	host := server.Hostname

	// Test cases
	t.Run("Basic Email Sending", func(t *testing.T) {
		err := SendEmailCannedWithCA(certPool, host, addr, "from@example.com",
			[]string{"to@example.com"}, "Test Subject", "Test Body")
		require.NoError(t, err, "Should send email successfully")
		e := <-inbox
		require.Equal(t, "from@example.com", e.MailFrom.Address)

		m, err := e.Mail()
		require.NoError(t, err)

		p, err := m.Headers()
		require.NoError(t, err)
		require.Equal(t, "Test Subject", p.Get("Subject"))
	})

	t.Run("Invalid Sender", func(t *testing.T) {
		// Test with invalid sender format
		err := SendEmailCannedWithCA(certPool, host, addr, "not-an-email",
			[]string{"to@example.com"}, "Test Subject", "Test Body")

		require.Error(t, err, "Should reject invalid sender format")
	})

	t.Run("Invalid Recipient", func(t *testing.T) {
		// Test with invalid recipient format
		err := SendEmailCannedWithCA(certPool, host, addr, "from@example.com",
			[]string{"not-an-email"}, "Test Subject", "Test Body")

		require.Error(t, err, "Should reject invalid recipient format")
	})

	t.Run("Multiple Recipients", func(t *testing.T) {
		// Test sending to multiple recipients
		err := SendEmailCannedWithCA(certPool, host, addr, "from@example.com",
			[]string{"to1@example.com", "to2@example.com"},
			"Test Subject", "Test Body")

		require.NoError(t, err, "Should handle multiple recipients")
		<-inbox
	})

	t.Run("Large Message Body", func(t *testing.T) {
		// Generate a large message body (1MB)
		// 32byte * 32 * 1024 = 1MB
		largeBody := strings.Repeat("Lorem ipsum dolor sit amet dore ", 32*1_024*9)

		err := SendEmailCannedWithCA(certPool, host, addr, "from@example.com",
			[]string{"to1@example.com"}, "Large Email", largeBody)
		require.NoError(t, err, "Should handle large message bodies")
		e := <-inbox
		m, err := e.Mail()
		require.NoError(t, err)

		body := m.RawBody
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(largeBody), strings.TrimSpace(string(body)))
	})

	t.Run("Large Message Body reset", func(t *testing.T) {
		// Generate a large message body (1MB)
		// 32byte * 32 * 1024 = 1MB
		largeBody := strings.Repeat("Lorem ipsum dolor sit amet dore ", 32*1_024*9)

		msg := fmt.Sprintf("From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"\r\n"+
			"%s", "from@example.com", "to@example.com", "subject", largeBody)

		tlsConfig := &tls.Config{
			RootCAs:    certPool,
			ServerName: host,
		}

		// Connect to the SMTP server (without TLS initially)
		c, err := smtp.Dial(addr)
		require.NoError(t, err, "Should connect to SMTP server successfully")
		defer c.Close()

		// Check if the server supports STARTTLS
		if ok, _ := c.Extension("STARTTLS"); ok {
			err = c.StartTLS(tlsConfig)
			require.NoError(t, err, "Should start TLS successfully")
		}

		for i := 0; i < 5; i++ {
			t.Log("Sending email:", i)
			err = c.Reset()
			require.NoError(t, err, "Should reset connection successfully")

			err = c.Mail("from@example.com")
			require.NoError(t, err, "Should set mail successfully")

			err = c.Rcpt("to@example.com")
			require.NoError(t, err, "Should set recipient successfully")

			w, err := c.Data()
			require.NoError(t, err, "Should set data successfully")

			_, err = w.Write([]byte(msg))
			require.NoError(t, err, "Should write data successfully")

			t.Log("Sent email successfully")

			err = w.Close()
			require.NoError(t, err, "Should close data successfully")
			t.Log("Closed data successfully")

			e := <-inbox
			t.Log("Received email successfully")

			m, err := e.Mail()
			require.NoError(t, err)

			body := m.RawBody
			require.Equal(t, strings.TrimSpace(largeBody), strings.TrimSpace(string(body)))
		}

	})

	t.Run("To Large Message Body", func(t *testing.T) {

		toLargeBody := strings.Repeat("Lorem ipsum dolor sit amet dore ", 32*1_024*20) // 20MB
		err := SendEmailCannedWithCA(certPool, host, addr, "from@example.com",
			[]string{"to1@example.com"}, "Large Email", toLargeBody)
		require.Error(t, err, "Should not handle to large body")
	})
}

func TestMiddlewareReturnPath(t *testing.T) {
	// Setup - generate a test CA and server certificate
	addr := ":2525"
	// Create an SMTP server for testing
	inbox, server, certPool := StartTLSServer(addr, t, middleware.AddReturnPath)
	defer server.Shutdown(context.Background())

	// Get the server address
	host := server.Hostname
	// Test cases
	t.Run("Basic", func(t *testing.T) {
		err := SendEmailCannedWithCA(certPool, host, addr, "from@example.com",
			[]string{"to@example.com"}, "Test Subject", "Test Body")
		require.NoError(t, err, "Should send email successfully")

		e := <-inbox
		m, err := e.Mail()

		require.NoError(t, err, "Should get mail successfully")

		head, err := m.Headers()
		require.NoError(t, err, "Should get headers successfully")

		assert.Equal(t, "<from@example.com>", head.Get("Return-Path"))
	})
}

func TestMiddlewareNilCheck(t *testing.T) {
	// Setup - generate a test CA and server certificate
	addr := ":2525"
	// Create an SMTP server for testing

	nilCheck := func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(e *envelope.Envelope) smtpx.Response {
			res := next(e)
			require.NotNil(t, res, "next should never return nil, nil shall be converted to 250 Message Accepted")
			return nil
		}
	}

	inbox, server, certPool := StartTLSServer(addr, t, nilCheck, nilCheck, nilCheck)
	defer server.Shutdown(context.Background())

	// Get the server address
	host := server.Hostname
	// Test cases
	t.Run("Basic", func(t *testing.T) {
		err := SendEmailCannedWithCA(certPool, host, addr, "from@example.com",
			[]string{"to@example.com"}, "Test Subject", "Test Body")
		require.NoError(t, err, "Should send email successfully")

		select {
		case <-inbox:
		case <-time.After(time.Second):
			t.Error("Should receive email successfully")
		}
	})
}

func TestMiddlewareSenderDomain(t *testing.T) {
	// Setup - generate a test CA and server certificate
	addr := ":2525"
	// Create an SMTP server for testing
	inbox, server, certPool := StartTLSServer(addr, t, middleware.SenderDomainsWhitelist("example.com"))
	defer server.Shutdown(context.Background())

	// Get the server address
	host := server.Hostname
	// Test cases
	t.Run("Success", func(t *testing.T) {
		err := SendEmailCannedWithCA(certPool, host, addr, "from@example.com",
			[]string{"to@example.com"}, "Test Subject", "Test Body")
		require.NoError(t, err, "Should send email successfully")

		select {
		case <-inbox:
		case <-time.After(time.Second):
			t.Error("Should receive email successfully")
		}
	})
	t.Run("Fail", func(t *testing.T) {
		err := SendEmailCannedWithCA(certPool, host, addr, "from@blacklist.com",
			[]string{"to@example.com"}, "Test Subject", "Test Body")

		require.Error(t, err, "Should not send email successfully")
		require.ErrorContains(t, err, "Sender domain not allowed")

	})

	t.Run("FailThenSuccess", func(t *testing.T) {

		msg := fmt.Sprintf("From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"\r\n"+
			"%s", "from@from.com", "to@to.com", "subject", "body")

		c, err := ConnWithCA(certPool, host, addr)
		defer c.Close()

		/// Failing from blacklist

		// Set the sender
		err = c.Mail("from@blacklist.com")
		require.NoError(t, err, "Should set sender successfully")

		// Set the recipient
		err = c.Rcpt("to@example.com")
		require.NoError(t, err, "Should set recipient successfully")

		// Send the email body
		w, err := c.Data()
		require.NoError(t, err, "Should set data successfully")

		_, err = w.Write([]byte(msg))
		require.NoError(t, err, "Should write data successfully")

		err = w.Close()
		require.Error(t, err, "Sender should fail heare")
		require.ErrorContains(t, err, "Sender domain not allowed")

		/// Resetting and trying again with valid domain

		err = c.Reset()
		require.NoError(t, err, "Should reset successfully")

		err = c.Mail("from@example.com")
		require.NoError(t, err, "Should set sender successfully")

		err = c.Rcpt("to@example.com")
		require.NoError(t, err, "Should set recipient successfully")

		w, err = c.Data()
		require.NoError(t, err, "Should set data successfully")

		_, err = w.Write([]byte(msg))
		require.NoError(t, err, "Should write data successfully")

		err = w.Close()
		require.NoError(t, err, "Should close data successfully")

		err = c.Quit()
		require.NoError(t, err, "Should quit successfully")

		select {
		case <-inbox:
		case <-time.After(time.Second):
			t.Error("Should receive email successfully")

		}

	})
}

func TestMiddlewareReceivedHeader(t *testing.T) {
	// Setup - generate a test CA and server certificate
	addr := ":2525"
	// Create an SMTP server for testing
	inbox, server, certPool := StartTLSServer(addr, t, middleware.AddReceivedHeaders(hostname))
	defer server.Shutdown(context.Background())

	// Get the server address
	host := server.Hostname
	// Test cases
	t.Run("Basic", func(t *testing.T) {
		err := SendEmailCannedWithCA(certPool, host, addr, "from@example.com",
			[]string{"to@example.com"}, "Test Subject", "Test Body")
		require.NoError(t, err, "Should send email successfully")

		e := <-inbox
		m, err := e.Mail()

		require.NoError(t, err, "Should get mail successfully")
		head, err := m.Headers()
		require.NoError(t, err, "Should get headers successfully")
		fmt.Println(head.Get("Received"))
		assert.Contains(t, head.Get("Received"), "to@example.com")
		assert.Contains(t, head.Get("Received"), "localhost")
		assert.Contains(t, head.Get("Received"), "127.0.0.1")
		assert.Contains(t, head.Get("Received"), "example.com")
	})
}

func TestMiddlewareOrder(t *testing.T) {

	var res []string
	adder := func(i int) smtpx.Middleware {
		return func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
			return func(e *envelope.Envelope) smtpx.Response {
				res = append(res, fmt.Sprintf("pre-%d", i))
				defer func() { res = append(res, fmt.Sprintf("post-%d", i)) }()
				return next(e)
			}
		}
	}

	stop := func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(e *envelope.Envelope) smtpx.Response {
			res = append(res, "stopped")
			return smtpx.NewResponse(550, "Stopped for no good reason")
		}
	}

	// Setup - generate a test CA and server certificate
	addr := ":2525"
	// Create an SMTP server for testing
	inbox, server, certPool := StartTLSServer(addr, t)
	defer server.Shutdown(context.Background())
	//MIME encoded-word syntax (RFC 2047) to represent non-ASCII characters.
	t.Run("Basic", func(t *testing.T) {
		res = nil
		server.Middlewares = append([]smtpx.Middleware{}, adder(0), adder(1), adder(2))
		// Test cases
		err := SendEmailCannedWithCA(certPool, server.Hostname, addr, "from@example.com",
			[]string{"to@example.com"}, "Test Subject", "Test Body")
		require.NoError(t, err, "Should send email successfully")
		<-inbox

		assert.Equal(t, []string{"pre-0", "pre-1", "pre-2", "post-2", "post-1", "post-0"}, res)

		res = nil

		err = SendEmailCannedWithCA(certPool, server.Hostname, addr, "from@example.com",
			[]string{"to@example.com"}, "Test Subject", "Test Body")
		require.NoError(t, err, "Should send email successfully")
		<-inbox

		assert.Equal(t, []string{"pre-0", "pre-1", "pre-2", "post-2", "post-1", "post-0"}, res)

	})

	t.Run("Early_Return", func(t *testing.T) {
		res = nil
		server.Middlewares = append([]smtpx.Middleware{}, adder(0), stop, adder(2))

		err := SendEmailCannedWithCA(certPool, server.Hostname, addr, "from@example.com",
			[]string{"to@example.com"}, "Test Subject", "Test Body")
		require.ErrorContains(t, err, "Stopped for no good reason")

		assert.Equal(t, []string{"pre-0", "stopped", "post-0"}, res)
	})
}

func TestEncoding(t *testing.T) {

	// Setup - generate a test CA and server certificate
	addr := ":2525"
	// Create an SMTP server for testing
	inbox, server, certPool := StartTLSServer(addr, t)
	defer server.Shutdown(context.Background())

	t.Run("UTF8", func(t *testing.T) {
		// Test cases
		err := SendEmailCannedWithCA(certPool, server.Hostname, addr, "from@example.com",
			[]string{"to@example.com"}, "漢字", "Test Body")
		require.NoError(t, err, "Should send email successfully")
		e := <-inbox
		m, err := e.Mail()

		require.NoError(t, err, "Should get mail successfully")

		h, err := m.Headers()
		require.NoError(t, err, "Should get headers successfully")
		assert.Equal(t, "漢字", h.Get("Subject"))

	})

	t.Run("Basic", func(t *testing.T) {
		// Test cases
		err := SendEmailCannedWithCA(certPool, server.Hostname, addr, "from@example.com",
			[]string{"to@example.com"}, "=?UTF-8?B?VGVzdCB3aXRoIMOpIGFuZCDkuK3lj7g=?=", "Test Body")
		require.NoError(t, err, "Should send email successfully")
		e := <-inbox
		m, err := e.Mail()

		require.NoError(t, err, "Should get mail successfully")

		h, err := m.Headers()
		require.NoError(t, err, "Should get headers successfully")
		assert.Equal(t, "Test with é and 中司", h.Get("Subject"))

	})

}

// SendEmailCannedWithCA sends an email using SMTP with a custom CA certificate
func SendEmailCannedWithCA(certPool *x509.CertPool, host string, addr string, from string, to []string, subject, body string) error {

	// Construct the message with headers and body
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", from, to, subject, body)

	return SendEmailWithCA(certPool, host, addr, from, to, msg)
}

// SendEmailCannedWithCA sends an email using SMTP with a custom CA certificate
func SendEmailWithCA(certPool *x509.CertPool, host string, addr string, from string, to []string, content string) error {
	c, err := ConnWithCA(certPool, host, addr)
	defer c.Close()

	// Set the sender
	if err = c.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set the recipient
	for _, recipient := range to {
		if err = c.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	// Send the email body
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("failed to open message writer: %w", err)
	}

	_, err = w.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("failed to close message writer: %w", err)
	}

	// Send the QUIT command and close the connection
	return c.Quit()
}

// SendEmailCannedWithCA sends an email using SMTP with a custom CA certificate
func ConnWithCA(certPool *x509.CertPool, host string, addr string) (*smtp.Client, error) {
	// Create a cert pool and add our CA

	// Configure TLS with our cert pool
	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		ServerName: host,
	}

	// Connect to the SMTP server (without TLS initially)
	c, err := smtp.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	// Check if the server supports STARTTLS
	if ok, _ := c.Extension("STARTTLS"); ok {
		// Start TLS
		if err = c.StartTLS(tlsConfig); err != nil {
			return nil, fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	return c, nil
}
