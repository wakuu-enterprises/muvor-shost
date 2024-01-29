package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	containerd "github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

func main() {
	ctx := context.Background()

	// Start containerd
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		fmt.Printf("Error creating containerd client: %v\n", err)
		return
	}
	defer client.Close()

	// Pull an example image
	image, err := client.Pull(ctx, "docker.io/library/alpine:latest", containerd.WithPullUnpack)
	if err != nil {
		fmt.Printf("Error pulling image: %v\n", err)
		return
	}

	// Create a libp2p host
	h, err := libp2p.New(ctx)
	if err != nil {
		fmt.Printf("Error creating libp2p host: %v\n", err)
		return
	}

	// Extract peer info from libp2p host
	selfPeerInfo := peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}

	// Share our peer info with the container
	err = sharePeerInfo(ctx, client, selfPeerInfo)
	if err != nil {
		fmt.Printf("Error sharing peer info: %v\n", err)
		return
	}

	// Run container
	container, err := client.NewContainer(
		ctx,
		"example-container",
		containerd.WithImage(image),
		containerd.WithNewSnapshot("example-snapshot", image),
		containerd.WithNewSpec(
			containerd.WithImageConfig(ctx, image),
			containerd.WithImageConfig(ctx, image),
			containerd.WithImageConfig(ctx, image),
		),
	)
	if err != nil {
		fmt.Printf("Error creating container: %v\n", err)
		return
	}
	defer container.Delete(ctx, containerd.WithSnapshotCleanup)

	// Start container
	task, err := container.NewTask(ctx, nil)
	if err != nil {
		fmt.Printf("Error creating task: %v\n", err)
		return
	}
	defer task.Delete(ctx)

	// Connect libp2p to the container network namespace
	err = connectLibp2pToContainerNetworkNamespace(ctx, h, task)
	if err != nil {
		fmt.Printf("Error connecting libp2p to container network namespace: %v\n", err)
		return
	}

	// Wait for container to exit
	statusC, err := task.Wait(ctx)
	if err != nil {
		fmt.Printf("Error waiting for task: %v\n", err)
		return
	}

	// Print exit status
	status := <-statusC
	fmt.Printf("Container exited with status: %d\n", status.ExitCode())
}

func sharePeerInfo(ctx context.Context, client *containerd.Client, self peer.AddrInfo) error {
	// Get container
	container, err := client.LoadContainer(ctx, "example-container")
	if err != nil {
		return err
	}

	// Get task
	task, err := container.Task(ctx, nil)
	if err != nil {
		return err
	}

	// TODO: Share peer info with the container network namespace

	return nil
}

func connectLibp2pToContainerNetworkNamespace(ctx context.Context, h host.Host, task containerd.Task) error {
	// Get container's network namespace path
	nsPath, err := task.NetworkNamespacePath(ctx)
	if err != nil {
		return err
	}

	// Create a veth pair
	cmd := exec.Command("ip", "link", "add", "veth0", "type", "veth", "peer", "name", "veth1")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Set the veth1 interface inside the container's network namespace
	cmd = exec.Command("ip", "link", "set", "veth1", "netns", nsPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Set IP address for veth0 on the host side
	cmd = exec.Command("ip", "addr", "add", "192.168.1.1/24", "dev", "veth0", "up")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Bring up veth0
	cmd = exec.Command("ip", "link", "set", "dev", "veth0", "up")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// TODO: Set the IP address for veth1 inside the container's network namespace

	// TODO: Connect libp2p to the veth0 interface

	return nil
}
