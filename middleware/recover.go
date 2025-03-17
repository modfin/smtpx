package middleware

import (
	"errors"
	"fmt"
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
)

func Recover(next smtpx.HandlerFunc) smtpx.HandlerFunc {
	return func(envelope *envelope.Envelope) (res smtpx.Response) {
		defer func() {
			if r := recover(); r != nil {
				envelope.SetError(errors.New(fmt.Sprintf("recovered: %v", r)))
				res = smtpx.NewResponse(500, "Internal Server GetError")
			}
		}()
		res = next(envelope)
		return res
	}

}
