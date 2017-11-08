package babymailgun

import (
	"bytes"
	"errors"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"strings"
)

const (
	RecipientTo  = "to"
	RecipientCC  = "cc"
	RecipientBCC = "bcc"
)

type EmailRecipients struct {
	Address      string
	Status       int
	StatusReason string
	Type         string
}

type Email struct {
	ID         string `_id`
	Subject    string
	Body       string
	Recipients []EmailRecipients
	MailFrom   []string
	CreatedAt  string // Go to Golang date
	UpdatedAt  string // Go to Golang date
	Status     string // Should be an emum
	Reason     string
	Tries      int
	WorkerId   string `worker_id`
}

type MongoClient struct {
	Host         string
	Port         string
	DatabaseName string
}

func (e *Email) FormatMessage() ([]byte, error) {
	var bodyBytes bytes.Buffer
	bodyBytes.WriteString("From: %s\r\n", e.MailFrom)
	for _, recipient := range e.Recipients {
		var rcptHeader string
		switch strings.ToLower(recipient.Type) {
		case RecipientTo:
			rcptHeader = fmt.Sprintf("To: %s\r\n", recipient.Address)
		case RecipientCC:
			rcptHeader = fmt.Sprintf("Cc: %s\r\n", recipient.Address)
		case RecipientBCC:
		default:
			return nil, errors.New("Malformatted email, recipient type is invalid")
		}
		if len(rcptAddr) > 0 {
			bodyBytes.WriteString(rcptHeader)
		}
	}
	bodyBytes.WriteString(fmt.Sprintf("Subject: %s\r\n", e.Subject))
	bodyBytes.WriteString(e.Body)
	return bodyBytes.Bytes(), nil
}

func (m *MongoClient) getClient() (*mgo.Session, error) {
	hostPort := fmt.Sprintf("%s:%s", m.Host, m.Port)
	session, err := mgo.Dial(hostPort)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (m *MongoClient) FetchReadyEmail(workerId string) (*Email, error) {
	session, err := m.getClient()
	if err != nil {
		return nil, err
	}

	defer session.Close()
	session.SetMode(mgo.Strong, true)

	emailCollection := session.DB(m.DatabaseName).C("emails")
	email := Email{}
	change := mgo.Change{
		Update:    bson.M{"$set": bson.M{"worker_id": workerId}},
		ReturnNew: true}

	info, err := emailCollection.Find(bson.M{"worker_id": nil}).Apply(change, &email)
	if err != nil {
		// Go back to sleep and hope mongo is ok?
		return nil, err
	}

	if info.Matched == 0 {
		return nil, errors.New("No emails available to be sent")
	}

	return &email, nil
}

func (m *MongoClient) UpdateEmail(email *Email) error {
	session, err := m.getClient()
	if err != nil {
		return err
	}

	defer session.Close()
	// TODO findAndModify only updates one document and single-document
	//			updates are atomic, but is this safe enough? We're doing a
	//			compare and swap of sorts (find with specific keys and replace)
	emailCollection := session.DB(m.DatabaseName).C("emails")
	err = emailCollection.Update(
		bson.M{"_id": email.ID},
		bson.M{"$set": bson.M{"worker_id": nil}})
	if err != nil {
		return err
	}
	return nil
}
