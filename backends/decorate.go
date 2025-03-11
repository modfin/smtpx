package backends

import (
	"github.com/phires/go-guerrilla/mail"
	"strings"
	"time"
)

// We define what a decorator to our processor will look like
type Decorator func(Processor) Processor

// Decorate will decorate a processor with a slice of passed decorators
func Decorate(c Processor, ds ...Decorator) Processor {
	decorated := c
	for _, decorate := range ds {
		decorated = decorate(decorated)
	}
	return decorated
}

func DecorateDeliveryHeader(primaryHost string) Decorator {
	return func(p Processor) Processor {
		return ProcessWith(func(e *mail.Envelope, task SelectTask) (Result, error) {
			if task == TaskSaveMail {
				to := strings.TrimSpace(e.RcptTo[0].User) + "@" + primaryHost
				hash := "unknown"
				if len(e.Hashes) > 0 {
					hash = e.Hashes[0]
				}
				protocol := "SMTP"
				if e.ESMTP {
					protocol = "E" + protocol
				}
				if e.TLS {
					protocol = protocol + "S"
				}
				var addHead string
				addHead += "Delivered-To: " + to + "\n"
				addHead += "Received: from " + e.RemoteIP + " ([" + e.RemoteIP + "])\n"
				if len(e.RcptTo) > 0 {
					addHead += "	by " + e.RcptTo[0].Host + " with " + protocol + " id " + hash + "@" + e.RcptTo[0].Host + ";\n"
				}
				addHead += "	" + time.Now().Format(time.RFC1123Z) + "\n"
				// save the result
				e.DeliveryHeader = addHead
				// next processor
				return p.Process(e, task)

			} else {
				return p.Process(e, task)
			}
		})
	}
}

func DecorateHeadersParser() Decorator {
	return func(p Processor) Processor {
		return ProcessWith(func(e *mail.Envelope, task SelectTask) (Result, error) {
			if task == TaskSaveMail {
				if err := e.ParseHeaders(); err != nil {
					Log().WithError(err).Error("parse headers error")
				}
				// next processor
				return p.Process(e, task)
			} else {
				// next processor
				return p.Process(e, task)
			}
		})
	}
}
