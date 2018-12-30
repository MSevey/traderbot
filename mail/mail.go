package mail

// the mail package handles sending emails. The code in this package should be
// restricted to that which generates and sends the email based on provided
// information. Any code that gathers the data or information for the email
// should be managed in the package where the data and/or information is coming
// from.
//
// NOTE: It is set up to send emails via gmail and the email addresses are
// currently hardcoded to be from and to mjsevey@gmail.com

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

var (
	gmailPassword = os.Getenv("gmailPassword")

	// SenderEmail is the Hard Coded sender Email
	SenderEmail = "mjsevey@gmail.com"

	// ToEmail is the Hard Coded To email address
	ToEmail = "mjsevey@gmail.com"
)

// Mail contains the information about the email being sent
type Mail struct {
	senderID string
	toIds    []string
	subject  string
	body     string
}

// SMTPServer is the server settings for emailing
type SMTPServer struct {
	host string
	port string
}

// SendEmail sends an email, based on the provided Mail parameters
//
// NOTE: currently hardcoded for Gmail
func SendEmail(mail Mail) error {
	// Build message
	messageBody := mail.BuildMessage()

	// Set mail server parameters
	smtpServer := SMTPServer{host: "smtp.gmail.com", port: "465"}

	// build authorization
	auth := smtp.PlainAuth("", mail.senderID, gmailPassword, smtpServer.host)

	// Gmail will reject connection if it's not secure
	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpServer.host,
	}
	conn, err := tls.Dial("tcp", smtpServer.ServerName(), tlsconfig)
	if err != nil {
		return err
	}

	// Build client
	client, err := smtp.NewClient(conn, smtpServer.host)
	if err != nil {
		return err
	}

	// Use Authorization
	if err = client.Auth(auth); err != nil {
		return err
	}

	// add all from and to addresses
	if err = client.Mail(mail.senderID); err != nil {
		return err
	}
	for _, k := range mail.toIds {
		if err = client.Rcpt(k); err != nil {
			return err
		}
	}

	// Write Data to body
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(messageBody))
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	// Close the Client
	err = client.Quit()
	if err != nil {
		return err
	}

	return nil
}

// ServerName returns the constructed name of the server
func (s *SMTPServer) ServerName() string {
	return s.host + ":" + s.port
}

// BuildMessage builds the email message
func (m *Mail) BuildMessage() string {
	var message string
	message += fmt.Sprintf("From: %s\r\n", m.senderID)
	if len(m.toIds) > 0 {
		message += fmt.Sprintf("To: %s\r\n", strings.Join(m.toIds, ";"))
	}

	message += fmt.Sprintf("Subject: %s\r\n", m.subject)
	message += "\r\n" + m.body

	return message
}
