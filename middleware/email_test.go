package middleware

import (
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"github.com/stretchr/testify/assert"
	"net/mail"
	"testing"
)

func TestFilterRecipient(t *testing.T) {
	tests := []struct {
		name           string
		recipients     []*mail.Address
		includeFunc    func(*mail.Address) bool
		expectedRcpt   []*mail.Address
		expectedStatus int
	}{
		{
			name: "Allow all recipients",
			recipients: []*mail.Address{
				{Address: "user1@example.com"},
				{Address: "user2@example.com"},
			},
			includeFunc: func(*mail.Address) bool { return true },
			expectedRcpt: []*mail.Address{
				{Address: "user1@example.com"},
				{Address: "user2@example.com"},
			},
			expectedStatus: 250,
		},
		{
			name: "Filter out all recipients",
			recipients: []*mail.Address{
				{Address: "user1@example.com"},
				{Address: "user2@example.com"},
			},
			includeFunc:    func(*mail.Address) bool { return false },
			expectedRcpt:   nil,
			expectedStatus: 550,
		},
		{
			name: "Filter some recipients",
			recipients: []*mail.Address{
				{Address: "user1@example.com"},
				{Address: "user2@otherdomain.com"},
			},
			includeFunc: func(addr *mail.Address) bool {
				return addr.Address == "user1@example.com"
			},
			expectedRcpt: []*mail.Address{
				{Address: "user1@example.com"},
			},
			expectedStatus: 250,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := FilterRecipient(tt.includeFunc)
			handler := middleware(func(e *envelope.Envelope) smtpx.Response {
				return smtpx.NewResponse(250, "OK")
			})

			env := &envelope.Envelope{RcptTo: tt.recipients}
			response := handler(env)

			assert.Equal(t, tt.expectedRcpt, env.RcptTo)
			assert.Equal(t, tt.expectedStatus, response.StatusCode())
		})
	}
}
