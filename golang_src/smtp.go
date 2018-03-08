package babymailgun

import (
	"fmt"
	"log"
	"net/smtp"
)

func SendMail(smtpServer *SMTPServer, email *Email) error {
	log.Println("Sending email...")

	auth := smtp.PlainAuth("", smtpServer.Username, smtpServer.Password, smtpServer.Hostname)

	hostPort := fmt.Sprintf("%s:%d", smtpServer.Hostname, smtpServer.Port)
	var sendTo []string
	for _, recipient := range email.Recipients {
		sendTo = append(sendTo, recipient.Address)
	}

	message := email.FormattedMessage()
	err := smtp.SendMail(hostPort, auth, email.MailFrom, sendTo, message)
	if err != nil {
		return err
	}
	return nil
}
