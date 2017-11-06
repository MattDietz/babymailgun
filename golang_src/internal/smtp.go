package babymailgun

import (
	"fmt"
	//	"net/smtp"
)

func SendMail(email *Email) {
	// Call protocol based interface method (SMTP, IMAP...) (Don't actually do this, this is fine)
	// Auth
	// Middleware (DKIM signing?), add headers (some standard, some premium features?), adding attachments etc
	// Formatting
	// Send
	fmt.Println(email)
}
