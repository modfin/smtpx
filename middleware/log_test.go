package middleware

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net"
	"net/mail"
	"testing"
)

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	middleware := Logger(testLogger)

	mockNext := func(e *envelope.Envelope) smtpx.Response {
		return smtpx.NewResponse(250, "OK")
	}

	testEnvelope := envelope.NewEnvelope(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}, 2)
	testEnvelope.Helo = "test.com"
	testEnvelope.RemoteAddr = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
	testEnvelope.MailFrom = &mail.Address{Address: "sender@example.com"}
	testEnvelope.RcptTo = []*mail.Address{{Address: "recipient@example.com"}}
	testEnvelope.TLS = true
	testEnvelope.ESMTP = true
	testEnvelope.UTF8 = true
	testEnvelope.Data = &envelope.Data{}

	testEnvelope.Data.WriteString("Message-Id: <12345>\r\nTo: recipient@example.com\r\nFrom: sender@example.com\r\n\r\nHello World")

	testEnvelope.SetError(errors.New("a error"))

	handler := middleware(mockNext)
	response := handler(testEnvelope)

	assert.Equal(t, 250, response.StatusCode())
	assert.Equal(t, "250 OK: OK", response.String())

	logOutput := buf.String()

	fmt.Println(logOutput)

	assert.Contains(t, logOutput, "Mail request")
	assert.Contains(t, logOutput, "HELO=test.com")
	assert.Contains(t, logOutput, "remote-ip=127.0.0.1:12345")
	assert.Contains(t, logOutput, "MAIL=sender@example.com")
	assert.Contains(t, logOutput, "RCPT=[recipient@example.com]")
	assert.Contains(t, logOutput, "TLS=true")
	assert.Contains(t, logOutput, "ESMTP=true")
	assert.Contains(t, logOutput, "UTF8=true")
	assert.Contains(t, logOutput, "Mail response")
	assert.Contains(t, logOutput, "code=250")
	assert.Contains(t, logOutput, `response="250 OK: OK"`)
	assert.Contains(t, logOutput, "duration=")
	assert.Contains(t, logOutput, "level=ERROR")
	assert.Contains(t, logOutput, `err="a error"`)

}

func TestLoggerNil(t *testing.T) {
	var buf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	middleware := Logger(testLogger)

	mockNext := func(e *envelope.Envelope) smtpx.Response {
		return nil
	}

	testEnvelope := envelope.NewEnvelope(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}, 2)
	testEnvelope.Helo = "test.com"
	testEnvelope.RemoteAddr = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
	testEnvelope.MailFrom = &mail.Address{Address: "sender@example.com"}
	testEnvelope.RcptTo = []*mail.Address{{Address: "recipient@example.com"}}
	testEnvelope.TLS = true
	testEnvelope.ESMTP = true
	testEnvelope.UTF8 = true
	testEnvelope.Data = &envelope.Data{}

	testEnvelope.Data.WriteString("Message-Id: <12345>\r\nTo: recipient@example.com\r\nFrom: sender@example.com\r\n\r\nHello World")

	testEnvelope.SetError(errors.New("a error"))

	handler := middleware(mockNext)
	res := handler(testEnvelope)

	assert.Equal(t, nil, res)

	logOutput := buf.String()

	fmt.Println(logOutput)

	assert.Contains(t, logOutput, "Mail request")
	assert.Contains(t, logOutput, "HELO=test.com")
	assert.Contains(t, logOutput, "remote-ip=127.0.0.1:12345")
	assert.Contains(t, logOutput, "MAIL=sender@example.com")
	assert.Contains(t, logOutput, "RCPT=[recipient@example.com]")
	assert.Contains(t, logOutput, "TLS=true")
	assert.Contains(t, logOutput, "ESMTP=true")
	assert.Contains(t, logOutput, "UTF8=true")
	assert.Contains(t, logOutput, "Mail response")
	assert.Contains(t, logOutput, "code=250")
	assert.Contains(t, logOutput, "duration=")
	assert.Contains(t, logOutput, "level=ERROR")
	assert.Contains(t, logOutput, `err="a error"`)

}
