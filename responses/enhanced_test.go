package responses

import (
	"github.com/modfin/smtpx"
	"testing"
)

func TestGetBasicStatusCode(t *testing.T) {
	// Known status code
	a := getBasicStatusCode(EnhancedStatusCode{2, OtherOrUndefinedProtocolStatus})
	if a != 250 {
		t.Errorf("getBasicStatusCode. Int \"%d\" not expected.", a)
	}

	// Unknown status code
	b := getBasicStatusCode(EnhancedStatusCode{2, OtherStatus})
	if b != 200 {
		t.Errorf("getBasicStatusCode. Int \"%d\" not expected.", b)
	}
}

// TestString for the String function
func TestCustomString(t *testing.T) {
	// Basic testing
	resp := &smtpx.Response{
		EnhancedCode: OtherStatus,
		BasicCode:    200,
		Class:        ClassSuccess,
		Comment:      "Test",
	}

	if resp.String() != "200 2.0.0 Test" {
		t.Errorf("CustomString failed. String \"%s\" not expected.", resp)
	}

	// Default String
	resp2 := &smtpx.Response{
		EnhancedCode: OtherStatus,
		Class:        ClassSuccess,
	}
	if resp2.String() != "200 2.0.0 OK" {
		t.Errorf("String failed. String \"%s\" not expected.", resp2)
	}
}

func TestBuildEnhancedResponseFromDefaultStatus(t *testing.T) {
	//a := buildEnhancedResponseFromDefaultStatus(ClassPermanentFailure, InvalidCommand)
	a := EnhancedStatusCode{ClassPermanentFailure, InvalidCommand}.String()
	if a != "5.5.1" {
		t.Errorf("buildEnhancedResponseFromDefaultStatus failed. String \"%s\" not expected.", a)
	}
}
