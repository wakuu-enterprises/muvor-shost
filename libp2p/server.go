package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"errors"
	"io"
	"io/ioutil"
	"time"
	"os/exec"

	"github.com/emersion/go-smtp"
	"github.com/libp2p/go-libp2p-core/peer"
)


// The Backend implements SMTP server methods.
type Backend struct{}

// NewSession is called after client greeting (EHLO, HELO).
func (bkd *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}

// A Session is returned after successful login.
type Session struct{}

type Email struct {
	From    string
	To      string
	Subject string
	Body    string
}

type Publisher struct {
	ID       string
	SMTP     *smtp.Server
	Pubsub   *Pubsub
	Emails   chan Email
	MvrpURLs chan string
}

type Subscriber struct {
	ID       string
	Email    string
	Messages chan Email
}

type Pubsub struct {
	Publishers map[string]*Publisher
	Subscribers map[string]*Subscriber
	mutex      sync.Mutex
}

func NewPubsub() *Pubsub {
	return &Pubsub{
		Publishers:  make(map[string]*Publisher),
		Subscribers: make(map[string]*Subscriber),
	}
}

func (ps *Pubsub) addPublisher(p *Publisher) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.Publishers[p.ID] = p
}

func (ps *Pubsub) removePublisher(p *Publisher) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	delete(ps.Publishers, p.ID)
}

func (ps *Pubsub) addSubscriber(s *Subscriber) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.Subscribers[s.Email] = s
}

func (ps *Pubsub) removeSubscriber(s *Subscriber) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	delete(ps.Subscribers, s.Email)
}

func (ps *Pubsub) getPublisher(id string) (*Publisher, bool) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	p, ok := ps.Publishers[id]
	return p, ok
}

func (ps *Pubsub) getSubscriber(email string) (*Subscriber, bool) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	s, ok := ps.Subscribers[email]
	return s, ok
}

// AuthPlain implements authentication using SASL PLAIN.
func (s *Session) AuthPlain(username, password string) error {
	if username != "username" || password != "password" {
		return errors.New("Invalid username or password")
	}
	return nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	log.Println("Mail from:", from)
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	log.Println("Rcpt to:", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	if b, err := ioutil.ReadAll(r); err != nil {
		return err
	} else {
		log.Println("Data:", string(b))
	}
	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

// ExampleServer runs an example SMTP server.
//
// It can be tested manually with e.g. telnet:
//
//	> telnet localhost 1025
//	EHLO localhost
//	AUTH PLAIN
//	AHVzZXJuYW1lAHBhc3N3b3Jk
//	MAIL FROM:<root@nsa.gov>
//	RCPT TO:<root@gchq.gov.uk>
//	DATA
//	Hey <3
//	.
func createSMTPServer() *smtp.Server {
	be := &Backend{}

	s := smtp.NewServer(be)

	s.Addr = "localhost:1025"
	s.Domain = "muvor.xyz"
	s.WriteTimeout = 10 * time.Second
	s.ReadTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

	log.Println("Starting server at", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
	
	return s
}

func createPublisher() *Publisher {
	// Specify the command and arguments
	pubsub := exec.Command("./build/publisher")

	// Capture the combined output (stdout and stderr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	// Print the output
	log.Printf("Command output:\n%s", output)

	publisher := &Publisher{
		ID:       id,
		SMTP:     createSMTPServer(),
		Pubsub:   pubsub,
		Emails:   make(chan Email),
		MvrpURLs: make(chan string),
	}

	// Register the publisher with the pubsub system
	pubsub.addPublisher(publisher)

	// Run LilP2P server concurrently
	go lilp2pServer(publisher)

	// Run SMTP server concurrently
	go smtpServer(publisher)

	// Process emails concurrently
	go func() {
		for {
			select {
			case email := <-publisher.Emails:
				// Process incoming emails, e.g., forward to subscribers
				for _, subscriber := range pubsub.Subscribers {
					subscriber.Messages <- email
				}
			case mvrpURL := <-publisher.MvrpURLs:
				// Process incoming mvrp URLs
				// For simplicity, assuming mvrp URLs contain peer IDs
				peerID, err := peer.Decode(strings.TrimPrefix(mvrpURL, "mvrp:"))
				if err == nil {
					// Connect to the specified peer
					connectToPeer(publisher, peerID)
				}
			}
		}
	}()

	return publisher
}

func createSubscriber(id int) *Subscriber {
	// Specify the command and arguments
	pubsub := exec.Command("./build/subscriber")

	// Capture the combined output (stdout and stderr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	// Print the output
	log.Printf("Command output:\n%s", output)

	email := fmt.Sprintf("abc_%d@x.com", id)
	return &Subscriber{
		ID:       fmt.Sprintf("subscriber%d", id),
		Email:    email,
		Messages: make(chan Email),
	}
}

func lilp2pServer(publisher *Publisher) {
	http.HandleFunc("/lilp2p", func(w http.ResponseWriter, r *http.Request) {
		// Handle LilP2P network messages here

		// Simulate sending an email from LilP2P to SMTP
		email := Email{
			From:    "lilp2p@x.com",
			To:      "abc@x.com",
			Subject: "LilP2P Message",
			Body:    "This is a LilP2P message!",
		}

		publisher.Emails <- email

		// Simulate sending an mvrp URL to process
		publisher.MvrpURLs <- "mvrp:" + r.RemoteAddr

		fmt.Fprintf(w, "Hello from LilP2P server!\n")
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", publisher.ID), nil))
}

func smtpServer(publisher *Publisher) {
	log.Fatal(publisher.SMTP.ListenAndServe())
}

func connectToPeer(publisher *Publisher, peerID peer.ID) {
	ctx := context.Background() // Use a context for the connection

	// Create a new peerinfo using the provided peer ID
	peerinfo := peer.AddrInfo{
		ID: peerID,
	}

	// Connect to the specified peer
	err := publisher.Pubsub.Host.Connect(ctx, peerinfo)
	if err != nil {
		log.Printf("Error connecting to peer %s: %v", peerID.Pretty(), err)
	}
}

