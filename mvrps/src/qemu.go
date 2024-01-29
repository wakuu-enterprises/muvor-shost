package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/digitalocean/go-qemu"
)

func newPub() {
	// create qemu img
	// cmd := exec.Command("./qemu-img.sh")
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	// if err := cmd.Run(); err != nil {
	// 	fmt.Println("Error converting QEMU Image:", err)
	// }
	// Create QEMU VM configuration
	vmConfig := qemu.Config{
		Bios: "./openwrt-x86-64-generic-kernel.bin",
		Devices: []qemu.Device{
			qemu.IDE{
				Driver: qemu.Driver{
					Name: "qemu",
					Type: "raw",
				},
				File: "./openwrt-x86-64-generic-ext4-combined-efi.img",
			},
		},
		Memory:    512, // in MB
		CPUs:      2,
		Kernel:    "~/System/Library/Kernels/kernel",
		Append:    "root=/dev/sda",
		BootOrder: []string{"n"}, []string{"cdn"},
	}

	// Create QEMU VM instance
	vm, err := qemu.NewVM(vmConfig)
	if err != nil {
		fmt.Println("Error creating QEMU VM:", err)
		return
	}
	defer vm.Close()

	// Start QEMU VM
	if err := vm.Run(); err != nil {
		fmt.Println("Error running QEMU VM:", err)
		return
	}

	// Wait for the VM to exit
	vm.Wait()

	fmt.Println("QEMU VM has exited.")
}
