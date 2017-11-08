package main

import (
	"crypto/rand"
	"fmt"
	"github.com/cerberus98/babymailgun"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type WorkerConfig struct {
	Host         string
	Port         string
	DatabaseName string
	WorkerSleep  int
	WorkerPool   int
	MailHost     string
	MailPort     string
	AdminEmail   string
}

// Create a UUID4. Source implementation here: https://groups.google.com/forum/#!msg/golang-nuts/d0nF_k4dSx4/rPGgfXv6QCoJ
func uuid() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func processingLoop(cfg *WorkerConfig, wg *sync.WaitGroup, quit <-chan bool) {
	// Fetch emails ready to be sent
	// Try to send them
	// Update the datastore
	// Go back to sleep
	log.Println("Starting processing loop...")
	workerId := uuid()
	defer wg.Done()
loop:
	for {
		mongoClient := babymailgun.MongoClient{Host: cfg.Host, Port: cfg.Port, DatabaseName: cfg.DatabaseName}
		select {
		case <-quit:
			log.Printf("Worker goroutine received quit")
			break loop
		default:
			log.Println("Waking up and looking for emails to send")
		}
		email, err := mongoClient.FetchReadyEmail(workerId)
		if err == nil {
			log.Printf("Got email %s Worker ID: %s\n", email.ID, workerId)
			// Try to send the email
			mc := babymailgun.MailConfig{MailHost: cfg.MailHost, MailPort: cfg.MailPort, AdminEmail: cfg.AdminEmail}
			if err = babymailgun.SendMail(&mc, email); err != nil {
				fmt.Println("Email sending failed ", err)
				// TODO Set the reason on the update document
				// TODO Some errors are definitely catastrophic
			}
			// Process the email response to see if any recipients failed and create
			// and update document. Additionally, "release" the email by setting the
			// worker_id to nil

			log.Printf("Updating and releasing email %s\n", email.ID)
			if err := mongoClient.UpdateEmail(email); err != nil {
				// TODO Couldn't update the email, what do we do?
				log.Println(err)
			}
		} else {
			log.Printf("Error while fetching emails: %s, %t\n", err, err)
		}

		log.Printf("Going back to sleep for %d seconds\n", cfg.WorkerSleep)
		time.Sleep(time.Duration(cfg.WorkerSleep) * time.Second)
	}
	log.Println("Finishing up...")
}

func loadConfig() *WorkerConfig {
	viper.BindEnv("DB_HOST")
	viper.BindEnv("DB_PORT")
	viper.BindEnv("DB_NAME")
	viper.SetDefault("WORKER_SLEEP", 2)
	viper.BindEnv("WORKER_SLEEP")
	viper.SetDefault("WORKER_POOL", 5)
	viper.SetDefault("MAIL_PORT", 25)
	viper.BindEnv("WORKER_POOL")
	viper.BindEnv("MAIL_HOST")
	viper.BindEnv("MAIL_PORT")
	viper.BindEnv("ADMIN_EMAIL")

	for _, key := range []string{"DB_HOST", "DB_PORT", "DB_NAME", "MAIL_HOST", "MAIL_PORT", "ADMIN_EMAIL"} {
		if !viper.IsSet(key) {
			log.Fatal(fmt.Sprintf("Can't find necessary environment variable %s", key))
		}
	}

	return &WorkerConfig{
		Host:         viper.GetString("db_host"),
		Port:         viper.GetString("db_port"),
		DatabaseName: viper.GetString("db_name"),
		WorkerSleep:  viper.GetInt("worker_sleep"),
		WorkerPool:   viper.GetInt("worker_pool"),
		MailHost:     viper.GetString("mail_host"),
		MailPort:     viper.GetString("mail_port"),
		AdminEmail:   viper.GetString("admin_email"),
	}
}

func main() {
	log.Println("Running worker")
	workerConfig := loadConfig()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	var quitChannels []chan bool
	for i := 0; i < workerConfig.WorkerPool; i++ {
		quit := make(chan bool, 1)
		quitChannels = append(quitChannels, quit)
		go processingLoop(workerConfig, &wg, quitChannels[i])
		wg.Add(1)
	}
	select {
	case <-sigs:
		log.Printf("Received quit")
		for i := 0; i < workerConfig.WorkerPool; i++ {
			quitChannels[i] <- true
		}
	}
	wg.Wait()
	log.Println("Done")
}
