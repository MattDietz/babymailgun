package babymailgun

import (
	"bytes"
	"errors"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"strings"
	"time"
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

type EmailStatus string
type EmailReason string

const (
	StatusComplete   EmailStatus = "complete"
	StatusIncomplete EmailStatus = "incomplete"
	StatusFailed     EmailStatus = "failed"
)

const (
	ReasonInvalidRecipient    EmailReason = "A recipient's address is invalid or does not exist"
	ReasonUnrecognizedCommand EmailReason = "Invalid authentication for the server, or server auth may be down"
	ReasonEOF                 EmailReason = "The server disconnected while trying to transmit the email"
)

type Email struct {
	ID         string `_id`
	Subject    string
	Body       string
	Recipients []EmailRecipients
	MailFrom   string `sender`
	CreatedAt  string // Go to Golang date
	UpdatedAt  string // Go to Golang date
	Status     EmailStatus
	Reason     EmailReason
	Tries      int
	WorkerId   string `worker_id`
}

type EmailUpdate struct {
	Recipients []EmailRecipients
	Status     EmailStatus
	Reason     EmailReason
	Tries      int
}

type MongoClientConfig struct {
	Host              string
	Port              string
	DatabaseName      string
	ConnectionRetries int
	ConnectionTimeout int
	SendRetries       int
	SendRetryInterval int
}

type MongoClient struct {
	Config  *MongoClientConfig
	session *mgo.Session
}

func (e *EmailUpdate) FromEmail(email *Email) {
	e.Tries = email.Tries
	e.Status = email.Status
	e.Reason = email.Reason
	for _, rcpt := range email.Recipients {
		e.Recipients = append(e.Recipients, rcpt)
	}
}

func (e *Email) FormatMessage() ([]byte, error) {
	var bodyBytes bytes.Buffer
	bodyBytes.WriteString(fmt.Sprintf("From: %s\r\n", e.MailFrom))
	var toRecipients, ccRecipients []string

	for _, recipient := range e.Recipients {
		switch strings.ToLower(recipient.Type) {
		case RecipientTo:
			toRecipients = append(toRecipients, recipient.Address)
		case RecipientCC:
			ccRecipients = append(ccRecipients, recipient.Address)
		case RecipientBCC:
		default:
			return nil, errors.New("Malformatted email, recipient type is invalid")
		}
	}
	if len(toRecipients) > 0 {
		bodyBytes.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(toRecipients, ", ")))
	}
	if len(ccRecipients) > 0 {
		bodyBytes.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(ccRecipients, ", ")))
	}
	bodyBytes.WriteString(fmt.Sprintf("Subject: %s\r\n", e.Subject))
	bodyBytes.WriteString("\r\n")
	bodyBytes.WriteString(e.Body)
	return bodyBytes.Bytes(), nil
}

func (m *MongoClient) dial() (*mgo.Session, error) {
	hostPort := fmt.Sprintf("%s:%s", m.Config.Host, m.Config.Port)
	timeout := time.Duration(m.Config.ConnectionTimeout) * time.Second
	session, err := mgo.DialWithTimeout(hostPort, timeout)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (m *MongoClient) CleanUp() {
	m.session.Close()
}

func (m *MongoClient) getClient() (*mgo.Session, error) {
	var connectionErr error

	for i := 0; i < m.Config.ConnectionRetries; i++ {
		if i > 0 {
			fmt.Println(fmt.Sprintf("Failed to dial datastore, %d tries remaining", m.Config.ConnectionRetries-i-1))
		}
		if m.session == nil {
			if sess, err := m.dial(); err != nil {
				connectionErr = err
				continue
			} else {
				m.session = sess
				m.session.SetSocketTimeout(time.Duration(m.Config.ConnectionTimeout) * time.Second)
				m.session.SetSyncTimeout(time.Duration(m.Config.ConnectionTimeout) * time.Second)
			}
		}
		if m.session != nil {
			if err := m.session.Ping(); err != nil {
				connectionErr = err
				m.session = nil
				continue
			} else {
				return m.session.Clone(), nil
			}
		}
	}
	// Couldn't access the datastore, what do we do?
	// In a production system, this would (at least) send alerts/page people, because this
	// may be lost data. We'll settle for a retry with backoff here, and fall back
	// to a panic. Not ideal, but we're basically dead if our datastore is dead.
	panic(fmt.Sprintf("Failed to connect to the datastore. Error: %s", connectionErr.Error()))
}

func (m *MongoClient) FetchReadyEmail(workerId string) *Email {
	session, err := m.getClient()
	if err != nil {
		return nil
	}
	session.SetMode(mgo.Strong, true)
	emailCollection := session.DB(m.Config.DatabaseName).C("emails")
	email := Email{}
	change := mgo.Change{
		Update:    bson.M{"$set": bson.M{"worker_id": workerId}},
		ReturnNew: true}

	// Fetch emails that are incomplete and N seconds old or older
	olderThan := bson.M{"$lt": time.Now().Add(-time.Duration(m.Config.SendRetryInterval) * time.Second)}
	info, err := emailCollection.Find(bson.M{"worker_id": nil, "status": "incomplete", "updated_at": olderThan}).Apply(change, &email)

	if err != nil {
		// Go back to sleep and hope mongo is ok?
		return nil
	}

	if info.Matched == 0 {
		return nil
	}

	return &email
}

func (m *MongoClient) UpdateEmail(email *Email, emailUpdate *EmailUpdate) error {
	session, err := m.getClient()
	if err != nil {
		return err
	}

	// TODO findAndModify only updates one document and single-document
	//			updates are atomic, but is this safe enough? We're doing a
	//			compare and swap of sorts (find with specific keys and replace)
	emailCollection := session.DB(m.Config.DatabaseName).C("emails")
	err = emailCollection.Update(
		bson.M{"_id": email.ID},
		// TODO This status has to be provided by the calling function
		bson.M{"$set": bson.M{
			"worker_id":  nil,
			"tries":      emailUpdate.Tries,
			"status":     emailUpdate.Status,
			"reason":     emailUpdate.Reason,
			"updated_at": time.Now(),
			"recipients": emailUpdate.Recipients}})

	if err != nil {
		return err
	}
	return nil
}
