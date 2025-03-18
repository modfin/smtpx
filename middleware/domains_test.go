package middleware

import (
	"github.com/modfin/smtpx/utils"
	"net/mail"
	"testing"

	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"github.com/stretchr/testify/assert"
)

func TestSenderDomainsWhitelist(t *testing.T) {
	tests := []struct {
		name           string
		whitelist      []string
		senderDomain   string
		expectedStatus int
	}{
		{"Empty whitelist", []string{}, "example.com", 250},
		{"Sender in whitelist", []string{"example.com"}, "example.com", 250},
		{"Sender not in whitelist", []string{"example.com"}, "other.com", 550},
		{"Multiple domains in whitelist", []string{"example.com", "other.com"}, "other.com", 250},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := SenderDomainsWhitelist(tt.whitelist...)
			handler := middleware(func(e *envelope.Envelope) smtpx.Response {
				return smtpx.NewResponse(250, "OK")
			})

			env := &envelope.Envelope{MailFrom: &mail.Address{Address: "someone@" + tt.senderDomain}}
			response := handler(env)

			assert.Equal(t, tt.expectedStatus, response.StatusCode())
		})
	}
}

func TestFilterRecipientDomains(t *testing.T) {
	tests := []struct {
		name           string
		whitelist      []string
		recipients     []string
		expectedStatus int
	}{
		{"Empty whitelist", []string{}, []string{"user@example.com"}, 250},
		{"Recipient in whitelist", []string{"example.com"}, []string{"user@example.com"}, 250},
		{"Recipient not in whitelist", []string{"example.com"}, []string{"user@other.com"}, 550},
		{"Multiple recipients, one in whitelist", []string{"example.com"}, []string{"user1@other.com", "user2@example.com"}, 250},
		{"Multiple recipients, none in whitelist", []string{"example.com"}, []string{"user1@other.com", "user2@another.com"}, 550},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := FilterRecipientDomains(tt.whitelist...)
			handler := middleware(func(e *envelope.Envelope) smtpx.Response {
				return smtpx.NewResponse(250, "OK")
			})

			env := &envelope.Envelope{RcptTo: make([]*mail.Address, len(tt.recipients))}
			for i, recipient := range tt.recipients {
				env.RcptTo[i] = &mail.Address{Address: recipient}
			}
			response := handler(env)

			assert.Equal(t, tt.expectedStatus, response.StatusCode())
		})
	}
}

func TestFilterRecipientDomains2(t *testing.T) {
	tests := []struct {
		name           string
		whitelist      []string
		recipients     []string
		expectedStatus int
		expectedRcptTo int
	}{
		{"Empty whitelist", []string{}, []string{"user@example.com"}, 250, 1},
		{"Recipient in whitelist", []string{"example.com"}, []string{"user@example.com"}, 250, 1},
		{"Recipient not in whitelist", []string{"example.com"}, []string{"user@other.com"}, 550, 0},
		{"Multiple recipients, one in whitelist", []string{"example.com"}, []string{"user1@other.com", "user2@example.com"}, 250, 1},
		{"Multiple recipients, none in whitelist", []string{"example.com"}, []string{"user1@other.com", "user2@another.com"}, 550, 0},
		{"Multiple recipients, all in whitelist", []string{"example.com", "other.com"}, []string{"user1@example.com", "user2@other.com"}, 250, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := FilterRecipientDomains(tt.whitelist...)
			handler := middleware(func(e *envelope.Envelope) smtpx.Response {
				return smtpx.NewResponse(250, "OK")
			})

			env := &envelope.Envelope{RcptTo: make([]*mail.Address, len(tt.recipients))}
			for i, recipient := range tt.recipients {
				env.RcptTo[i] = &mail.Address{Address: recipient}
			}
			response := handler(env)

			assert.Equal(t, tt.expectedStatus, response.StatusCode())
			assert.Equal(t, tt.expectedRcptTo, len(env.RcptTo), "Number of recipients after filtering should match expected")

			if tt.expectedStatus == 250 && len(tt.whitelist) > 0 {
				for _, rcpt := range env.RcptTo {
					domain := utils.DomainOfEmail(rcpt)
					assert.Contains(t, tt.whitelist, domain, "Remaining recipients should be in the whitelist")
				}
			}
		})
	}
}

func TestRecipientDomainsWhitelist(t *testing.T) {
	tests := []struct {
		name           string
		whitelist      []string
		recipients     []string
		expectedStatus int
	}{
		{"Empty whitelist", []string{}, []string{"user@example.com"}, 250},
		{"Recipient in whitelist", []string{"example.com"}, []string{"user@example.com"}, 250},
		{"Recipient not in whitelist", []string{"example.com"}, []string{"user@other.com"}, 550},
		{"Multiple recipients, one in whitelist", []string{"example.com"}, []string{"user1@other.com", "user2@example.com"}, 250},
		{"Multiple recipients, none in whitelist", []string{"example.com"}, []string{"user1@other.com", "user2@another.com"}, 550},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := RecipientDomainsWhitelist(tt.whitelist...)
			handler := middleware(func(e *envelope.Envelope) smtpx.Response {
				return smtpx.NewResponse(250, "OK")
			})

			env := &envelope.Envelope{RcptTo: make([]*mail.Address, len(tt.recipients))}
			for i, recipient := range tt.recipients {
				env.RcptTo[i] = &mail.Address{Address: recipient}
			}
			response := handler(env)

			assert.Equal(t, tt.expectedStatus, response.StatusCode())
		})
	}
}
