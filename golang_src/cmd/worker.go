package main

import (
	"crypto/rand"
	"fmt"
	"github.com/cerberus98/babymailgun/internal"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Create a UUID4. Source implementation here: https://groups.google.com/forum/#!msg/golang-nuts/d0nF_k4dSx4/rPGgfXv6QCoJ
func uuid() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func processingLoop(host string, port string, dbName string, wg *sync.WaitGroup, workerId string, quit chan bool) {
	// Fetch emails ready to be sent
	// Try to send them
	// Update the datastore
	// Go back to sleep
	fmt.Println("Starting processing loop...")
loop:
	for {
		select {
		case <-quit:
			break loop
		default:
			fmt.Println("Looking for emails to send")
		}
		email, err := babymailgun.FetchReadyEmail(host, port, dbName, workerId)
		if err == nil {
			fmt.Printf("%T %s\n", email, email)
		} else {
			fmt.Printf("Error while fetching emails: %s\n", err)
		}
		time.Sleep(1 * time.Second)
	}
	fmt.Println("Finishing up...")
	wg.Done()
}

func main() {
	fmt.Println("Running worker")
	host, ok := os.LookupEnv("DB_HOST")
	if !ok {
		log.Fatal("Can't find necessary environment variable DB_HOST")
	}
	port, ok := os.LookupEnv("DB_PORT")
	if !ok {
		log.Fatal("Can't find necessary environment variable DB_PORT")
	}
	dbName, ok := os.LookupEnv("DB_NAME")
	if !ok {
		log.Fatal("Can't find necessary environment variable DB_NAME")
	}
	sigs := make(chan os.Signal, 1)
	quit := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	workerId := uuid()
	go processingLoop(host, port, dbName, &wg, workerId, quit)
	wg.Add(1)
	select {
	case <-sigs:
		quit <- true
	}
	wg.Wait()
	fmt.Println("Done")
}
