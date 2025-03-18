package middleware

import (
	"fmt"
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"time"
)

func AddReturnPath(next smtpx.HandlerFunc) smtpx.HandlerFunc {
	return func(e *envelope.Envelope) smtpx.Response {
		_ = e.PrependHeader("Return-Path", fmt.Sprintf("<%s>", e.MailFrom.Address))
		return next(e)
	}
}

func AddDeliveredHeaders() smtpx.Middleware {
	return func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(e *envelope.Envelope) smtpx.Response {
			if len(e.RcptTo) == 1 {
				_ = e.PrependHeader("Delivered-To", e.RcptTo[0].Address)
			}
			return next(e)
		}
	}
}

func AddReceivedHeaders(hostname string) smtpx.Middleware {
	return func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(e *envelope.Envelope) smtpx.Response {

			protocol := "SMTP"
			if e.ESMTP {
				protocol = "E" + protocol
			}
			if e.TLS {
				protocol = protocol + "S"
			}

			clientId := e.ConnectionId()
			envelopeId := e.EnvelopeId()

			id := fmt.Sprintf("%d-%s@%s", clientId, envelopeId, hostname)

			received := fmt.Sprintf("from %s (%s [%s])\r\n", e.Helo, e.Helo, e.RemoteAddr.String())
			received += fmt.Sprintf("  by %s with %s id %s\r\n", hostname, protocol, id)
			if len(e.RcptTo) == 1 {
				received += fmt.Sprintf("  for <%s>\r\n", e.RcptTo[0].Address)
			}
			received += fmt.Sprintf("  %s", time.Now().In(time.UTC).Format(time.RFC1123Z))
			// save the result

			_ = e.PrependHeader("Received", received)

			return next(e)
		}
	}
}
