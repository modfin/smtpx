package utils

import (
	"net/mail"
	"strings"
)

func DomainOfEmail(email *mail.Address) string {
	if email == nil {
		return ""
	}
	parts := strings.Split(email.Address, "@")
	return strings.ToLower(parts[len(parts)-1])
}
