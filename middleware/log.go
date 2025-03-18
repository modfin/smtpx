package middleware

import (
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"log/slog"
	"time"
)

type Skipper func(*envelope.Envelope) bool

type LoggerSettings struct {
	Skip       Skipper
	PreFields  []func(*envelope.Envelope) (string, any)
	PostFields []func(*envelope.Envelope, smtpx.Response) (string, any)
}

type Option func(*LoggerSettings)

func WithSkipper(s Skipper) Option {
	return func(settings *LoggerSettings) {
		settings.Skip = s
	}
}

func Logger(logger *slog.Logger, opts ...Option) smtpx.Middleware {
	var settings = &LoggerSettings{}

	settings.PreFields = []func(*envelope.Envelope) (string, any){
		func(e *envelope.Envelope) (string, any) {
			return "HELO", e.Helo
		},

		func(envelope *envelope.Envelope) (string, any) { return "remote-ip", envelope.RemoteAddr },
		func(envelope *envelope.Envelope) (string, any) { return "MAIL", envelope.MailFrom.Address },
		func(envelope *envelope.Envelope) (string, any) {
			var tos []string
			for _, t := range envelope.RcptTo {
				tos = append(tos, t.Address)
			}
			return "RCPT", tos
		},
		func(envelope *envelope.Envelope) (string, any) { return "TLS", envelope.TLS },
		func(envelope *envelope.Envelope) (string, any) { return "ESMTP", envelope.ESMTP },
		func(envelope *envelope.Envelope) (string, any) { return "UTF8", envelope.UTF8 },
		func(envelope *envelope.Envelope) (string, any) { return "size", envelope.Data.Len() },
	}

	settings.PostFields = []func(*envelope.Envelope, smtpx.Response) (string, any){
		func(envelope *envelope.Envelope, res smtpx.Response) (string, any) {
			code := 250
			if res != nil {
				code = res.StatusCode()
			}
			return "code", code
		},
		func(envelope *envelope.Envelope, res smtpx.Response) (string, any) {
			var s string
			if res != nil {
				s = res.String()
			}
			return "response", s
		},
		func(envelope *envelope.Envelope, res smtpx.Response) (string, any) {
			return "size", envelope.Data.Len()
		},
	}

	for _, o := range opts {
		if o == nil {
			continue
		}
		o(settings)
	}

	return func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(envelope *envelope.Envelope) smtpx.Response {
			if logger == nil {
				return next(envelope)
			}

			if settings.Skip != nil && settings.Skip(envelope) {
				return next(envelope)
			}

			start := time.Now()
			lvl := slog.LevelInfo

			l := logger.With("envelope-id", envelope.EnvelopeId())
			l = logger.With("connection-id", envelope.ConnectionId())

			if m, _ := envelope.Mail(); m != nil {
				if h, _ := m.Headers(); h != nil {
					l = l.With("message-id", h.Get("Message-Id"))
				}
			}

			var args []any
			for _, f := range settings.PreFields {
				k, v := f(envelope)
				args = append(args, k, v)
			}
			l.Log(envelope.Context(), lvl, "Mail request", args...)

			res := next(envelope)

			elapsed := time.Since(start)
			args = append([]any{}, "duration", elapsed)
			for _, f := range settings.PostFields {
				k, v := f(envelope, res)
				args = append(args, k, v)
			}

			err := envelope.GetError()
			if err != nil {
				args = append(args, "err", err)
				lvl = slog.LevelError
			}
			l.Log(envelope.Context(), lvl, "Mail response", args...)
			return res
		}
	}
}
