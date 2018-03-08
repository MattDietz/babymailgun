package babymailgun

import (
	"fmt"
	"testing"
)

func TestEmailUpdateFromEmail(t *testing.T) {
	recipients := []EmailRecipient{EmailRecipient{Address: "test@unittests.com", Status: 0, Reason: "", Type: RecipientTo}}
	e := Email{ID: "1", Tries: 0, Status: "", Reason: "", Recipients: recipients}
	update := EmailUpdate{}
	update.FromEmail(&e)

	if update.Tries != 0 {
		t.Errorf("Expected update.Tries == 0, instead got %d", update.Tries)
	}

	if update.Status != "" {
		t.Errorf("Expected update.Status == '', instead got '%d;", update.Status)
	}

	if update.Reason != "" {
		t.Errorf("Expected update.Reason == '', instead got '%d;", update.Reason)
	}

	if len(update.Recipients) != 1 {
		t.Errorf("Expected len(update.Recipients) == 1, instead it's len %d", len(update.Recipients))
	}

	if update.Recipients[0].Address != recipients[0].Address {
		t.Errorf("Expected update.Recipients[0].Address == '%s', instead got '%s'", recipients[0].Address, update.Recipients[0].Address)
	}
}

func TestFormattedMessage(t *testing.T) {
	body := "This is an email body"
	recipients := []EmailRecipient{
		EmailRecipient{Address: "test@unittests.com", Type: RecipientTo},
		EmailRecipient{Address: "cc@unittests.com", Type: RecipientCC},
		EmailRecipient{Address: "bcc@unittests.com", Type: RecipientBCC}}

	e := Email{ID: "1", Tries: 0, Status: "", Reason: "", Recipients: recipients, Body: body, Subject: "A Subject", MailFrom: "from@unitests.com"}

	expected := fmt.Sprintf("From: %s\r\nTo: %s\r\nCc: %s\r\nSubject: %s\r\n\r\n%s", e.MailFrom, e.Recipients[0].Address, e.Recipients[1].Address, e.Subject, body)

	message_bytes := e.FormattedMessage()

	message := string(message_bytes[:])
	if message != expected {
		t.Errorf("Expected email.Body == '%s', instead got '%s'", message)
	}
}
