package main

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
)

// DiscoveryInterval is how often we re-publish our mDNS records.
const DiscoveryInterval = time.Hour

// DiscoveryServiceTag is used in our mDNS advertisements to discover other peers.
const DiscoveryServiceTag = "mvrp"

func main() {
	ctx := context.Background()

	// create a new libp2p Host that listens on a random TCP port
	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/7654"))
	if err != nil {
		panic(err)
	}

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

	// discovery addresses
	multiAddr, err := multiaddr.NewMultiaddr("/ip4/20.228.145.221/tcp/7654/p2p/QmS1bjwjMnx98KmEF6faSoeXxTSQu2aS5Eg5J2AQD36Vfq")
	if err != nil {
		panic(err)
	}
	println(multiAddr)

	// setup DHT
	//discoveryPeers := []multiaddr.Multiaddr{multiAddr}
	discoveryPeers := []multiaddr.Multiaddr{}
	dht, err := NewDHT(ctx, host, discoveryPeers)
	if err != nil {
		panic(err)
	}

	// setup peer discovery
	go Discover(ctx, host, dht, "muv")

	// join the pubsub topic called muv
	room := "muv"
	topic, err := gossipSub.Join(room)
	if err != nil {
		panic(err)
	}

	// subscribe to topic
	subscriber, err := topic.Subscribe()
	if err != nil {
		panic(err)
	}
	subscribe(subscriber, ctx, host.ID())
}

// start subsriber to topic
func subscribe(subscriber *pubsub.Subscription, ctx context.Context, hostID peer.ID) {
	for {
		msg, err := subscriber.Next(ctx)
		if err != nil {
			panic(err)
		}

		// only consider messages delivered by other peers
		if msg.ReceivedFrom == hostID {
			continue
		}

		fmt.Printf("got message: %s, from: %s\n", string(msg.Data), msg.ReceivedFrom.Pretty())
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
