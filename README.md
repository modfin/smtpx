
# SMTPX

> A lightweight SMTP server package written in Go that feels familiar.


## Installation

```bash
go get github.com/modfin/smtpx
```

## Usage

One idea for smtpx is to have it feel familiar and have the same feel as the standard library has for `net/http`.

```go

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

			// Parses headers while keeping encoding.
			// For human-readable decoded format, use mail.Headers()
			headers, err := mail.HeadersLiteral()

			// Error handling ignored for brevity
			fmt.Println("From:", headers.From())
			fmt.Println("To:", headers.To())
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



```


## Tribute

### A Fork
`smtpx` started out as a fork of `go-guerrilla` but there is very little left from the original project.
The goal is quite different for `smtpx`, and pretty much everything has been rewritten in 
a more go-like way focused on have `smtpx` work as a package and not a stand-alone server.   

https://github.com/phires/go-guerrilla (original https://github.com/flashmob/go-guerrilla)

### Other projects

There are a few other smtp servers written in go, so please have look before using `smtpx`.

- https://github.com/emersion/go-smtp
- https://github.com/phires/go-guerrilla
- https://github.com/foxcpp/maddy


### License
MIT
