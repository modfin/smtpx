package middleware

import (
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"net/mail"
)

// FilterRecipient Check if the recipient email address should be included.
func FilterRecipient(include func(address *mail.Address) bool) smtpx.Middleware {
	return func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(e *envelope.Envelope) smtpx.Response {

			var res []*mail.Address
			for _, r := range e.RcptTo {
				if include(r) {
					res = append(res, r)
				}
			}
			e.RcptTo = res
			if len(e.RcptTo) > 0 {
				return next(e)
			}

			return smtpx.NewResponse(550, "filtered recipient")
		}
	}
}
