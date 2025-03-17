package middleware

import (
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"testing"
)

func TestRecover(t *testing.T) {
	t.Run("No panic", func(t *testing.T) {
		handler := func(e *envelope.Envelope) smtpx.Response {
			return smtpx.NewResponse(200, "OK")
		}

		recoveredHandler := Recover(handler)
		env := &envelope.Envelope{}

		res := recoveredHandler(env)

		if res.StatusCode() != 200 {
			t.Errorf("Expected status code 200, got %d", res.StatusCode())
		}
	})

	t.Run("With panic", func(t *testing.T) {
		handler := func(e *envelope.Envelope) smtpx.Response {
			panic("test panic")
		}

		recoveredHandler := Recover(handler)
		env := &envelope.Envelope{}

		res := recoveredHandler(env)

		if res.StatusCode() != 500 {
			t.Errorf("Expected status code 500, got %d", res.StatusCode())
		}

		if res.String() != "500 Permanent failure: Internal Server GetError" {
			t.Errorf("Expected body 'Internal Server GetError', got '%s'", res.String())
		}

		errorValue := env.GetError()
		if errorValue == nil {
			t.Error("Expected error value in context, got nil")
		}
		if errorValue.Error() != "recovered: test panic" {
			t.Errorf("Expected error value 'test panic', got '%v'", errorValue)
		}
	})
}
