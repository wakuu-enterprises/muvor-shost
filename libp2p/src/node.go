package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-tcp-transport"
)

func main() {
	ctx := context.Background()

	// Set up a libp2p host
	host, err := libp2p.New(ctx, libp2p.Transport(tcp.NewTCPTransport)))
	if err != nil {
		fmt.Println("Error creating libp2p host:", err)
		return
	}

	// Start QEMU virtual machine on one node
	if host.ID() == "Node1" {
		go startQEMU("myimage.qcow2")
	}

	// Discover peers
	host.SetStreamHandler("/image-share/1.0.0", handleImageRequest)

	// Simulate peer discovery
	peerList := []string{"Node1"}
	for _, peer := range peerList {
		if peer != host.ID() {
			connectToPeer(ctx, host, peer)
		}
	}

	select {}
}

func startQEMU(imagePath string) {
	cmd := exec.Command("qemu-system-x86_64", "-drive", "file="+imagePath, "-m", "512M", "-enable-kvm")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Error starting QEMU:", err)
	}
}

func connectToPeer(ctx context.Context, host host.Host, peerID string) {
	// Connect to a peer
	peerInfo := host.Peerstore().PeerInfo(peerID)
	err := host.Connect(ctx, peerInfo)
	if err != nil {
		fmt.Println("Error connecting to peer:", err)
		return
	}

	fmt.Println("Connected to peer:", peerID)

	// Request the image from the connected peer
	stream, err := host.NewStream(ctx, peerInfo.ID, "/image-share/1.0.0")
	if err != nil {
		fmt.Println("Error opening stream:", err)
		return
	}

	// Send a simple request
	_, err = stream.Write([]byte("RequestImage"))
	if err != nil {
		fmt.Println("Error writing to stream:", err)
		return
	}

	// Handle the response
	// (You would implement more sophisticated handling based on your protocol)
}

func handleImageRequest(stream network.Stream) {
	// Receive and process the request
	buf := make([]byte, 1024)
	n, err := stream.Read(buf)
	if err != nil {
		fmt.Println("Error reading from stream:", err)
		return
	}

	request := string(buf[:n])

	// Respond to the request
	if request == "RequestImage" {
		// Send the image data over the stream
		// (You would implement image data transfer based on your protocol)
	}
}
