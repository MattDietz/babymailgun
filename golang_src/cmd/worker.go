package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/cerberus98/babymailgun/golang_src"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type WorkerConfig struct {
	Host              string
	Port              string
	DatabaseName      string
	WorkerSleep       int
	WorkerPool        int
	ConnectionTimeout int
	ConnectionRetries int
	SendRetries       int
	SendRetryInterval int // in seconds
}

const (
	InvalidRecipientError    string = "550 Invalid Recipient"
	UnrecognizedCommandError string = "500 Unrecognised command"
	EOFError                 string = "EOF"
)

const (
	InvalidRecipientStatus    = 550
	UnrecognizedCommandStatus = 500
)

// Create a UUID4. Source implementation here: https://groups.google.com/forum/#!msg/golang-nuts/d0nF_k4dSx4/rPGgfXv6QCoJ
func uuid() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func ErrorToUpdateStatus(err error, cfg *WorkerConfig, emailUpdate *babymailgun.EmailUpdate) {
	if err != nil {
		errStr := err.Error()
		fmt.Println("Email sending failed ", errStr)
		errStr = strings.ToLower(errStr)

		if strings.HasPrefix(errStr, strings.ToLower(InvalidRecipientError)) {
			// This is a catastrophic failure. The server says our recipient doesn't exist
			var failedEmail string
			if len(errStr) == len(InvalidRecipientError) {
				// Edge case: the recipient is an empty string
				failedEmail = ""
			} else {
				failedEmail = errStr[len(InvalidRecipientError)+1:]
			}
			for _, rcpt := range emailUpdate.Recipients {
				if rcpt.Address == failedEmail {
					rcpt.Status = InvalidRecipientStatus
					rcpt.Reason = InvalidRecipientError
				}
			}
			emailUpdate.Status = babymailgun.StatusFailed
			emailUpdate.Reason = babymailgun.ReasonInvalidRecipient
		}

		if strings.HasPrefix(errStr, strings.ToLower(UnrecognizedCommandError)) {
			// This is an auth failure. This may be catastrophic, but we'll retry since it could
			// be a function of load. In other words, we don't know how it's actually handling auth
			// on the server side, and the referring service could be down temporarily
			emailUpdate.Status = babymailgun.StatusIncomplete
			emailUpdate.Reason = babymailgun.ReasonUnrecognizedCommand
		}

		if strings.HasPrefix(errStr, strings.ToLower(EOFError)) {
			// this is potentially a temporary failure, and the message should be retried
			emailUpdate.Status = babymailgun.StatusIncomplete
			emailUpdate.Reason = babymailgun.ReasonEOF
		}

		if emailUpdate.Status == babymailgun.StatusIncomplete {
			emailUpdate.Tries += 1
			log.Printf("Email '%s' failed to send and has %d tries remaining. Reason: %s", emailUpdate.ID, cfg.SendRetries-emailUpdate.Tries, emailUpdate.Reason)
			if emailUpdate.Tries >= cfg.SendRetries {
				log.Printf("Tries exhausted. Marking email '%s' as failed", emailUpdate.ID)
				emailUpdate.Status = babymailgun.StatusFailed
			}
		}
	} else {
		for _, rcpt := range emailUpdate.Recipients {
			rcpt.Reason = "ok"
		}
		emailUpdate.Status = babymailgun.StatusComplete
	}
}

func processingLoop(cfg *WorkerConfig, wg *sync.WaitGroup, quit <-chan bool) {
	// Fetch emails ready to be sent
	// Try to send them
	// Update the datastore
	// Go back to sleep
	log.Println("Starting processing loop...")
	workerId := uuid()
	defer wg.Done()

	clientConfig := babymailgun.MongoClientConfig{
		Host: cfg.Host, Port: cfg.Port, DatabaseName: cfg.DatabaseName,
		ConnectionTimeout: cfg.ConnectionTimeout, ConnectionRetries: cfg.ConnectionRetries,
		SendRetries: cfg.SendRetries, SendRetryInterval: cfg.SendRetryInterval}
	mongoClient := babymailgun.MongoClient{Config: &clientConfig}

loop:
	for {
		select {
		case <-quit:
			log.Println("Worker goroutine received quit")
			mongoClient.CleanUp()
			break loop
		default:
			log.Println("Looking for emails to send")
		}

		var err error
		var email *babymailgun.Email
		var smtpServer *babymailgun.SMTPServer

		// Get a server first. Don't lock up an email if we can't send it
		smtpServer, err = mongoClient.GetSMTPServer()
		if err == nil {
			email, err = mongoClient.FetchReadyEmail(workerId)
			if email != nil {
				log.Printf("Got email %s Worker ID: %s\n", email.ID, workerId)

				emailUpdate := babymailgun.EmailUpdate{}
				emailUpdate.FromEmail(email)

				err = babymailgun.SendMail(smtpServer, email)
				ErrorToUpdateStatus(err, cfg, &emailUpdate)

				// Process the email response to see if any recipients failed and create
				// and update document. Additionally, "release" the email by setting the
				// worker_id to nil
				log.Printf("Updating and releasing email %s\n", email.ID)
				mongoClient.UpdateEmail(email, &emailUpdate)
			}
		}

		if err != nil || smtpServer == nil || email == nil {
			if err != nil {
				// NOTE If the datastore goes down long enough, We'd want to trigger an alert at this point,
				//			and probably have monitoring restart the entire process via signal, as it's the safest
				//			(non-racey) way to recover
				log.Printf("Caught Error: %s", err.Error())
			}
			log.Printf("Going back to sleep for %d seconds\n", cfg.WorkerSleep)
			time.Sleep(time.Duration(cfg.WorkerSleep) * time.Second)
		}
	}
	log.Println("Finishing up...")
}

func loadConfig() (*WorkerConfig, error) {
	viper.BindEnv("DB_HOST")
	viper.BindEnv("DB_PORT")
	viper.BindEnv("DB_NAME")
	viper.BindEnv("WORKER_SLEEP")
	viper.BindEnv("WORKER_POOL")
	viper.BindEnv("CONNECTION_RETRIES")
	viper.BindEnv("CONNECTION_TIMEOUT")
	viper.BindEnv("SEND_RETRIES")
	viper.BindEnv("SEND_RETRY_INTERVAL")

	viper.SetDefault("WORKER_SLEEP", 10)
	viper.SetDefault("WORKER_POOL", 5)
	viper.SetDefault("CONNECTION_RETRIES", 3)
	viper.SetDefault("CONNECTION_TIMEOUT", 30)
	viper.SetDefault("SEND_RETRIES", 3)
	viper.SetDefault("SEND_RETRY_INTERVAL", 600) // in seconds

	missing_keys := make([]string, 0)
	for _, key := range []string{"DB_HOST", "DB_PORT", "DB_NAME"} {
		if !viper.IsSet(key) {
			missing_keys = append(missing_keys, key)
		}
	}
	if len(missing_keys) > 0 {
		return nil, errors.New(fmt.Sprintf("Can't find necessary environment variable(s): %s", strings.Join(missing_keys, ", ")))
	}

	for _, key := range []string{"WORKER_SLEEP", "WORKER_POOL", "CONNECTION_RETRIES", "CONNECTION_TIMEOUT", "SEND_RETRIES", "SEND_RETRY_INTERVAL"} {
		if viper.GetInt(key) <= 0 {
			return nil, errors.New(fmt.Sprintf("Config %s must have a value >= 0", key))
		}
	}
	return &WorkerConfig{
		Host:              viper.GetString("db_host"),
		Port:              viper.GetString("db_port"),
		DatabaseName:      viper.GetString("db_name"),
		WorkerSleep:       viper.GetInt("worker_sleep"),
		WorkerPool:        viper.GetInt("worker_pool"),
		ConnectionRetries: viper.GetInt("connection_retries"),
		ConnectionTimeout: viper.GetInt("connection_timeout"),
		SendRetries:       viper.GetInt("send_retries"),
		SendRetryInterval: viper.GetInt("send_retry_interval"),
	}, nil
}

func main() {
	log.Println("Running worker")
	workerConfig, err := loadConfig()
	if err != nil {
		log.Fatal(err.Error())
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// The parent closes this channel which lets the workers know to exit
	quitChannel := make(chan bool)
	for i := 0; i < workerConfig.WorkerPool; i++ {
		go processingLoop(workerConfig, &wg, quitChannel)
		wg.Add(1)
	}

	select {
	case <-sigs:
		log.Printf("Received quit")
		close(quitChannel)
	}
	log.Println("Waiting for workers to exit...")
	wg.Wait()
	log.Println("Done")
}
