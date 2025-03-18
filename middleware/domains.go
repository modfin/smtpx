package middleware

import (
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"github.com/modfin/smtpx/utils"
	"net/mail"
	"strings"
)

// FilterRecipientDomains Check if the recipient domain, extracted from "RCPT TO" command, is in the whitelist
// Example usage: server.Use(middleware.RecipientDomainsWhitelist("example.com", "other-domain.com"))
// if any domain in RCPT TO is in whitelist the middleware will continue
// if no domain in RCPT TO is in whitelist the middleware will return stats code 550
// if no domains was provided to RecipientDomainsWhitelist, all domains are allowed
// envelope.Envelope.RcptTo is filtered and will only contain emails with allowed domains
func FilterRecipientDomains(domain ...string) smtpx.Middleware {
	var set = map[string]bool{}
	for _, d := range domain {
		set[strings.ToLower(d)] = true
	}
	return func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(e *envelope.Envelope) smtpx.Response {
			if len(set) == 0 {
				return next(e)
			}
			var res []*mail.Address
			for _, r := range e.RcptTo {
				if set[utils.DomainOfEmail(r)] {
					res = append(res, r)
				}
			}
			e.RcptTo = res

			if len(res) > 0 {
				return next(e)
			}

			return smtpx.NewResponse(550, "Recipient domain not allowed")
		}
	}
}

// RecipientDomainsWhitelist Check if the recipient domain, extracted from "RCPT TO" command, is in the whitelist
// Example usage: server.Use(middleware.RecipientDomainsWhitelist("example.com", "other-domain.com"))
// if any domain in RCPT TO is in whitelist the middleware will continue
// if no domain in RCPT TO is in whitelist the middleware will return stats code 550
// if no domains was provided to RecipientDomainsWhitelist, all domains are allowed
// envelope.Envelope.RcptTo is not filtered and will contain all that was received in RCPT TO
func RecipientDomainsWhitelist(domain ...string) smtpx.Middleware {
	var set = map[string]bool{}
	for _, d := range domain {
		set[strings.ToLower(d)] = true
	}
	return func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(e *envelope.Envelope) smtpx.Response {
			if len(set) == 0 {
				return next(e)
			}
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
// if the whitelist contains no domains, all domains are valid
func SenderDomainsWhitelist(domain ...string) smtpx.Middleware {

	var set = map[string]bool{}
	for _, d := range domain {
		set[strings.ToLower(d)] = true
	}

	return func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(e *envelope.Envelope) smtpx.Response {
			if len(set) == 0 {
				return next(e)
			}
			domain := strings.ToLower(utils.DomainOfEmail(e.MailFrom))
			if !set[domain] {
				return smtpx.NewResponse(550, "Sender domain not allowed")
			}

			return next(e)
		}
	}
}
