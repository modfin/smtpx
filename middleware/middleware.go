package middleware

import (
	"fmt"
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"github.com/modfin/smtpx/utils"
	"strings"
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

// RecipientDomainsWhitelist Check if the recipient domain, extracted from "RCPT TO" command, is in the whitelist
// Example usage: server.Use(middleware.RecipientDomainsWhitelist("example.com", "other-domain.com"))
func FilterRecipientDomains(domain ...string) smtpx.Middleware {
	var set = map[string]bool{}
	for _, d := range domain {
		set[strings.ToLower(d)] = true
	}
	return func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(e *envelope.Envelope) smtpx.Response {
			for _, r := range e.RcptTo {
				domain := strings.ToLower(utils.DomainOfEmail(r))
				if set[domain] {
					return next(e)
				}
			}
			return smtpx.NewResponse(550, "Recipient domain not allowed")
		}
	}
}

// RecipientDomainsWhitelist Check if the recipient domain, extracted from "RCPT TO" command, is in the whitelist
// Example usage: server.Use(middleware.RecipientDomainsWhitelist("example.com", "other-domain.com"))
// if any domain in RCPT TO is in whitelist the middleware will continue
// if no domain in RCPT TO is in whitelist the middleware will return stats code 550
// envelope.Envelope.RcptTo is not filtered and will contain all that was recviced in RCPT TO
func RecipientDomainsWhitelist(domain ...string) smtpx.Middleware {
	var set = map[string]bool{}
	for _, d := range domain {
		set[strings.ToLower(d)] = true
	}
	return func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(e *envelope.Envelope) smtpx.Response {
			for _, r := range e.RcptTo {
				domain := strings.ToLower(utils.DomainOfEmail(r))
				if set[domain] {
					return next(e)
				}
			}
			return smtpx.NewResponse(550, "Recipient domain not allowed")
		}
	}
}

// SenderDomainsWhitelist Check if the sender domain, extracted from "MAIL FROM" command is in the whitelist
// example usage: server.Use(middleware.SenderDomainsWhitelist("example.com", "other-domain.com"))
// if domain is not in the whitelist, the middleware will stop and return stats code 550 to email client
func SenderDomainsWhitelist(domain ...string) smtpx.Middleware {

	var set = map[string]bool{}
	for _, d := range domain {
		set[strings.ToLower(d)] = true
	}

	return func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(e *envelope.Envelope) smtpx.Response {
			domain := strings.ToLower(utils.DomainOfEmail(e.MailFrom))
			if !set[domain] {
				return smtpx.NewResponse(550, "Sender domain not allowed")
			}

			return next(e)
		}
	}
}
