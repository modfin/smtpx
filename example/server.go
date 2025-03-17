package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"github.com/modfin/smtpx/middleware"
	"github.com/modfin/smtpx/responses"
	"golang.org/x/crypto/acme/autocert"
	"log/slog"
	"os"
	"os/signal"
	"time"
)

func main() {

	hostname := "example.com"
	certManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache("acme-cache"),
		HostPolicy: autocert.HostWhitelist(hostname),
	}

	slog.SetLogLoggerLevel(slog.LevelDebug)

	server := smtpx.Server{
		Logger: slog.Default(),

		// Hostname is used for HELO command, and is required for TLS
		Hostname: hostname,
		Addr:     ":25",

		// TLSConfig is optional and is used for STARTTLS command
		TLSConfig: &tls.Config{ // example using autocert for TLS
			GetCertificate: certManager.GetCertificate,
		},

		// Middlewares are executed in order
		Middlewares: []smtpx.Middleware{
			middleware.Logger(slog.Default()),
			middleware.Recover,
			middleware.AddReturnPath,
		},

		// Handler for incoming emails
		// smtpx.NewHandler creates a handler from a function
		Handler: smtpx.NewHandler(func(e *envelope.Envelope) smtpx.Response {

			fmt.Println("Command MAIL", e.MailFrom)
			fmt.Println("Command RCPT", e.RcptTo)
			fmt.Println("Command DATA", e.Data.String())

			// Opening the envelope and getting the mail
			// ie. Splitting the Data section of a email into header and body
			mail, err := e.Mail()
			if err != nil {
				return responses.FailSyntaxError
			}

			// Parses headers canonical headers and decodes them to UTF8 in case of RFC 2047 MIME encoding
			headers, err := mail.Headers()

			fmt.Println("From:", headers.Get("From"))
			fmt.Println("To:", headers.Get("To"))
			fmt.Println("Subject:", headers.Get("Subject"))
			fmt.Println("Body:", string(mail.Body()))

			return smtpx.NewResponse(250, "OK")
		}),
	}

	// Graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, os.Kill)
		<-sig

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			slog.Default().Error("failed to shutdown server", "err", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		slog.Default().Error("failed to start server", "err", err)
	}
}
