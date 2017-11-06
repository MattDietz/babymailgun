package babymailgun

import (
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
}

func FetchReadyEmail(host string, port string, dbName string, workerId string) (*Email, error) {
	hostPort := fmt.Sprintf("%s:%s", host, port)
	session, err := mgo.Dial(hostPort)
	if err != nil {
		return nil, err
	}
	defer session.Close()
	session.SetMode(mgo.Strong, true)

	emailCollection := session.DB(dbName).C("emails")
	email := Email{}
	change := mgo.Change{
		Update:    bson.M{"$set": bson.M{"worker_id": workerId}},
		ReturnNew: true}

	info, err := emailCollection.Find(bson.M{"workerId": nil}).Apply(change, &email)
	if err != nil {
		// Go back to sleep and hope mongo is ok?
		return nil, err
	}
	fmt.Println(info)

	return &email, nil
}
