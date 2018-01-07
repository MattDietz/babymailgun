package babymailgun

import (
	"bytes"
	"errors"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"math/rand"
	"strings"
	"time"
)

type EmailStatus string
type EmailReason string
type RecipientType string

const (
	RecipientTo  RecipientType = "to"
	RecipientCC  RecipientType = "cc"
	RecipientBCC RecipientType = "bcc"
)

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

type EmailRecipient struct {
	Address string
	Status  int
	Reason  string
	Type    RecipientType
}

type Email struct {
	ID         string `_id`
	Subject    string
	Body       string
	Recipients []EmailRecipient
	MailFrom   string `sender`
	CreatedAt  string // Go to Golang date
	UpdatedAt  string // Go to Golang date
	Status     EmailStatus
	Reason     EmailReason
	Tries      int
	WorkerId   string `worker_id`
}

type EmailUpdate struct {
	ID         string
	Recipients []EmailRecipient
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

// NOTE In a real production environment we wouldn't store the credentials in mongo. Instead we'd use
//			something like Hashicorp Vault and distribute x509 certs to all the production hosts we want
//			to have access to any sensitive data. Also this would be an interface type to correspond
//			to different auth mechanisms (No reason to assume SMTP Auth is the only auth we need to consider)
//			Lastly, I'm choosing not to model failing servers out here. I think such logic is better served
//			by a load balancer capable of dynamicaly adding and removing hosts from a pool (Such as HAProxy).
type SMTPServer struct {
	ID       string `_id`
	Username string
	Password string
	Hostname string
	Port     int
}

var (
	NoServersFoundError   = errors.New("No SMTP servers are available to send emails")
	InvalidRecipientError = errors.New("Malformatted email, recipient type is invalid")
)

func (e *EmailUpdate) FromEmail(email *Email) {
	e.ID = email.ID
	e.Tries = email.Tries
	e.Status = email.Status
	e.Reason = email.Reason
	for _, rcpt := range email.Recipients {
		e.Recipients = append(e.Recipients, rcpt)
	}
}

func (m *MongoClient) GetSMTPServer() (*SMTPServer, error) {
	// NOTE In a production environment, we may want to restructure this to be a pipeline
	//			of sorts, where emails are logically assigned to SMTP servers based on some
	//			load semantics. Either load could be represented in the database, or distinct
	//			sending-only workers could sit in front of a single SMTP server/relay, monitoring
	//			load, and accepting messages from a queue. A leaky-bucket could simulate this
	//			fairly well. https://en.wikipedia.org/wiki/Leaky_bucket However, this is fairly
	//			complex to model and has lots of race-prone conditions to consider, and probably
	//			warrants a project all it's own
	session, err := m.getClient()
	if err != nil {
		return nil, err
	}
	session.SetMode(mgo.Strong, false)
	smtpCollection := session.DB(m.Config.DatabaseName).C("servers")
	query := bson.M{}
	var smtpServers []SMTPServer

	// Yes, this is naive. If we had a couple hundred thousand servers this would
	// be a disaster. We'd likely want to replace this with round-robin selection
	// at the very least, but optimally instead something like the NOTE above
	err = smtpCollection.Find(query).All(&smtpServers)
	if err != nil {
		return nil, err
	}
	if len(smtpServers) == 0 {
		return nil, NoServersFoundError
	}
	server := &smtpServers[rand.Intn(len(smtpServers))]
	log.Printf("Found %d server(s), using %s:%d", len(smtpServers), server.Hostname, server.Port)
	return server, nil
}

func (e *Email) FormattedMessage() []byte {
	var bodyBytes bytes.Buffer
	bodyBytes.WriteString(fmt.Sprintf("From: %s\r\n", e.MailFrom))
	var toRecipients, ccRecipients []string

	for _, recipient := range e.Recipients {
		switch recipient.Type {
		case RecipientTo:
			toRecipients = append(toRecipients, recipient.Address)
		case RecipientCC:
			ccRecipients = append(ccRecipients, recipient.Address)
		case RecipientBCC:
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
	return bodyBytes.Bytes()
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
	if m.session != nil {
		m.session.Close()
	}
}

func (m *MongoClient) getClient() (*mgo.Session, error) {
	var connectionErr error

	for i := 0; i < m.Config.ConnectionRetries; i++ {
		if i > 0 {
			log.Println(fmt.Sprintf("Failed to dial datastore with %d timeout, %d tries remaining", m.Config.ConnectionTimeout, m.Config.ConnectionRetries-i-1))
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
	// to an error.
	return nil, errors.New(fmt.Sprintf("Failed to connect to the datastore. Error: %s", connectionErr.Error()))
}

func (m *MongoClient) FetchReadyEmail(workerId string) (*Email, error) {
	session, err := m.getClient()
	if err != nil {
		return nil, err
	}
	session.SetMode(mgo.Strong, true)
	emailCollection := session.DB(m.Config.DatabaseName).C("emails")
	email := Email{}
	change := mgo.Change{
		Update:    bson.M{"$set": bson.M{"worker_id": workerId}},
		ReturnNew: true}

	// Fetch emails that are incomplete and N seconds old or older
	olderThan := bson.M{"$lt": time.Now().Add(-time.Duration(m.Config.SendRetryInterval) * time.Second)}
	_, err = emailCollection.Find(bson.M{"worker_id": nil, "status": "incomplete", "updated_at": olderThan}).Apply(change, &email)

	if err != nil {
		// We might have lost contact with Mongo, or there are simply no emails to send right now
		if err.Error() == "not found" {
			log.Println("Nothing to send")
			return nil, nil
		}
		return nil, err
	}

	return &email, nil
}

func (m *MongoClient) UpdateEmail(email *Email, emailUpdate *EmailUpdate) error {
	session, err := m.getClient()
	if err != nil {
		return err
	}

	emailCollection := session.DB(m.Config.DatabaseName).C("emails")
	err = emailCollection.Update(
		bson.M{"_id": email.ID},
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
