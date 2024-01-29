package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// DiscoveryInterval is how often we re-publish our mDNS records.
const DiscoveryInterval = time.Hour

// DiscoveryServiceTag is used in our mDNS advertisements to discover other peers.
const DiscoveryServiceTag = "librum-pubsub"

func main() {
	ctx := context.Background()

	// create a new libp2p Host that listens on a random TCP port
	// we can specify port like /ip4/0.0.0.0/tcp/3326
	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/7676/ws"))
	if err != nil {
		panic(err)
	}

	// Set up a handler for incoming connections
	host.SetStreamHandler("/example", func(stream network.Stream) {
		// Handle incoming stream
		fmt.Println("Received connection from:", stream.Conn().RemotePeer())
	})

	// Set up WebSocket server
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Fatal(err)
			return
		}

		// Create a LibP2P stream over the WebSocket connection
		stream, err := host.NewStream(ctx, host.ID(), "/example")
		if err != nil {
			log.Fatal(err)
			return
		}

		// Handle the WebSocket connection
		go handleWebSocket(conn, stream)
	})

	// Start WebSocket server
	go func() {
		log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
	}()

	select {}

	// view host details and addresses
	fmt.Printf("host ID %s\n", host.ID().Pretty())
	fmt.Printf("following are the assigned addresses\n")
	for _, addr := range host.Addrs() {
		fmt.Printf("%s\n", addr.String())
	}
	fmt.Printf("\n")

	// create a new PubSub service using the GossipSub router
	gossipSub, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	// setup local mDNS discovery
	if err := setupDiscovery(host); err != nil {
		panic(err)
	}

	// join the pubsub topic called librum
	room := "librum"
	topic, err := gossipSub.Join(room)
	if err != nil {
		panic(err)
	}

	// create publisher
	publish(ctx, topic)
	newPub()
}

// start publisher to topic
func publish(ctx context.Context, topic *pubsub.Topic) {
	for {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			fmt.Printf("enter message to publish: \n")

			msg := scanner.Text()
			if len(msg) != 0 {
				// publish message to topic
				bytes := []byte(msg)
				topic.Publish(ctx, bytes)
			}
		}
	}
}

func handleWebSocket(conn *websocket.Conn, stream network.Stream) {
	defer stream.Close()

	// Create a channel to receive messages from WebSocket
	wsMessages := make(chan []byte)

	// Goroutine to read messages from WebSocket and send them to LibP2P stream
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("WebSocket read error:", err)
				close(wsMessages)
				return
			}
			wsMessages <- message
		}
	}()

	// Goroutine to read messages from LibP2P stream and send them to WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stream.Read(buf)
			if err != nil {
				log.Println("LibP2P stream read error:", err)
				return
			}
			err = conn.WriteMessage(websocket.TextMessage, buf[:n])
			if err != nil {
				log.Println("WebSocket write error:", err)
				return
			}
		}
	}()

	// Main loop to handle messages from both WebSocket and LibP2P stream
	for {
		select {
		case message, ok := <-wsMessages:
			if !ok {
				return
			}
			// Handle messages received from WebSocket (e.g., log or process)
			log.Printf("Received message from WebSocket: %s\n", message)

			// Send the WebSocket message to the LibP2P stream
			_, err := stream.Write(message)
			if err != nil {
				log.Println("LibP2P stream write error:", err)
				return
			}

		case <-stream.Context().Done():
			return
		}
	}
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("discovered new peer %s\n", pi.ID.Pretty())
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID.Pretty(), err)
	}
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func setupDiscovery(h host.Host) error {
	// setup mDNS discovery to find local peers
	s := mdns.NewMdnsService(h, DiscoveryServiceTag, &discoveryNotifee{h: h})
	return s.Start()
}
