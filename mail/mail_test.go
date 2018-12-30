package mail

import (
	"fmt"
	"strings"
	"testing"
)

// TestBuildMessage tests the BuildMessage method
func TestBuildMessage(t *testing.T) {
	// Create mail struct
	email1 := "test1@test.com"
	email2 := "test2@test.com"
	subject := "A random subject"
	emailBody := "The body of this email"
	mail := Mail{
		senderID: SenderEmail,
		toIds:    []string{email1, email2},
		subject:  subject,
		body:     emailBody,
	}

	messageBody := mail.BuildMessage()

	var message string
	message += fmt.Sprintf("From: %s\r\n", SenderEmail)
	message += fmt.Sprintf("To: %s\r\n", strings.Join([]string{email1, email2}, ";"))
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "\r\n" + emailBody

	if message != messageBody {
		t.Log("Message:", message)
		t.Log("MessageBody:", messageBody)
		t.Fatal("message and messageBody not equal")
	}
}
