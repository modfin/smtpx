package guerrilla

import (
	"context"
	"fmt"
	"github.com/crholm/brevx/envelope"
	"log/slog"
	"net"
	"net/smtp"
	"os"
	"sync"
	"testing"
	"time"
)

func TestStartStop(t *testing.T) {

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	s := Server{
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
