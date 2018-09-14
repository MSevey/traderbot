package mail

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
)

var (
	gmailPassword = os.Getenv("gmailPassword")
)

// Mail is a helper struct for email updates
type Mail struct {
	senderID string
	toIds    []string
	subject  string
	body     string
}

// SMTPServer is the server for emailing
type SMTPServer struct {
	host string
	port string
}

// SendEmail sends an email, currently set up for Gmail
func SendEmail() {
	mail := Mail{}
	mail.senderID = "mjsevey@gmail.com"
	mail.toIds = []string{"mjsevey@gmail.com"}
	mail.subject = "This is the email subject"
	mail.body = "Harry Potter and threat to Hogwarts\n\nGood editing!!"

	messageBody := mail.BuildMessage()

	smtpServer := SMTPServer{host: "smtp.gmail.com", port: "465"}

	log.Println(smtpServer.host)
	// build an auth
	auth := smtp.PlainAuth("", mail.senderID, gmailPassword, smtpServer.host)

	// Gmail will reject connection if it's not secure
	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpServer.host,
	}

	conn, err := tls.Dial("tcp", smtpServer.ServerName(), tlsconfig)
	if err != nil {
		log.Panic(err)
	}

	client, err := smtp.NewClient(conn, smtpServer.host)
	if err != nil {
		log.Panic(err)
	}

	// step 1: Use Auth
	if err = client.Auth(auth); err != nil {
		log.Panic(err)
	}

	// step 2: add all from and to
	if err = client.Mail(mail.senderID); err != nil {
		log.Panic(err)
	}
	for _, k := range mail.toIds {
		if err = client.Rcpt(k); err != nil {
			log.Panic(err)
		}
	}

	// Data
	w, err := client.Data()
	if err != nil {
		log.Panic(err)
	}

	_, err = w.Write([]byte(messageBody))
	if err != nil {
		log.Panic(err)
	}

	err = w.Close()
	if err != nil {
		log.Panic(err)
	}

	client.Quit()

	log.Println("Mail sent successfully")
}

// ServerName returns the constructed name of the server
func (s *SMTPServer) ServerName() string {
	return s.host + ":" + s.port
}

// BuildMessage builds the email message
func (mail *Mail) BuildMessage() string {
	message := ""
	message += fmt.Sprintf("From: %s\r\n", mail.senderID)
	if len(mail.toIds) > 0 {
		message += fmt.Sprintf("To: %s\r\n", strings.Join(mail.toIds, ";"))
	}

	message += fmt.Sprintf("Subject: %s\r\n", mail.subject)
	message += "\r\n" + mail.body

	return message
}
