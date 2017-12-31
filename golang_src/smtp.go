package babymailgun

import (
	"fmt"
	"log"
	"net/smtp"
)

type MailConfig struct {
	MailHost   string
	MailPort   string
	AdminEmail string
}

func SendMail(cfg *MailConfig, email *Email) error {
	log.Println("Sending email...")
	auth := smtp.PlainAuth("", cfg.AdminEmail, "password", cfg.MailHost)

	hostPort := fmt.Sprintf("%s:%s", cfg.MailHost, cfg.MailPort)
	var sendTo []string
	for _, recipient := range email.Recipients {
		sendTo = append(sendTo, recipient.Address)
	}

	message, err := email.FormatMessage()
	if err != nil {
		return err
	}
	// TODO This never seems to time out, making it hard to kill the server
	err = smtp.SendMail(hostPort, auth, email.MailFrom, sendTo, message)
	if err != nil {
		return err
	}
	return nil
}
