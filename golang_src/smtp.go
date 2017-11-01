package smtp

import (
	"fmt"
	"net/smtp"
)

type Email struct {
	Subject   string
	Body      string
	MailFrom  []string
	MailTo    []string
	CreatedAt string // Go to Golang date
	UpdatedAt string // Go to Golang date
	Status    string // Should be an emum
	Reason    string
	Tries     int
}

func sendMail(email *Email) {
	// Call protocol based interface method (SMTP, IMAP...) (Don't actually do this, this is fine)
	// Auth
	// Middleware (DKIM signing?), add headers (some standard, some premium features?), adding attachments etc
	// Formatting
	// Send
}

func main() {

}
