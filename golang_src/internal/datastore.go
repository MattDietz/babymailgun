package babymailgun

import (
	"errors"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type EmailRecipients struct {
	Address      string
	Status       int
	StatusReason string
}

type Email struct {
	ID      string `_id`
	Subject string
	Body    string
	EmailRecipients
	MailFrom  []string
	MailTo    []string
	CreatedAt string // Go to Golang date
	UpdatedAt string // Go to Golang date
	Status    string // Should be an emum
	Reason    string
	Tries     int
	WorkerId  string `worker_id`
}

type MongoClient struct {
	Host         string
	Port         string
	DatabaseName string
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

func (m *MongoClient) UpdateEmail(emailId string, email *Email) error {
	return nil
}

func (m *MongoClient) ReleaseEmail(email *Email) error {
	session, err := m.getClient()
	if err != nil {
		return err
	}

	defer session.Close()
	emailCollection := session.DB(m.DatabaseName).C("emails")
	err = emailCollection.Update(
		bson.M{"_id": email.ID},
		bson.M{"$set": bson.M{"worker_id": nil}})
	if err != nil {
		return err
	}
	return nil
}
