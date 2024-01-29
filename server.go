package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

type SystemInfo struct {
	CPUSpeed string `json:"cpuSpeed"`
	RAM      string `json:"ram"`
	// Add more fields as needed
}

func getSystemInfo() (SystemInfo, error) {
	// Run shell commands to get system information
	cpuSpeedCmd := exec.Command("sysctl", "-n", "machdep.cpu.brand_string")
	ramCmd := exec.Command("sysctl", "-n", "hw.memsize")

	cpuSpeed, err := cpuSpeedCmd.Output()
	if err != nil {
		return SystemInfo{}, err
	}

	ram, err := ramCmd.Output()
	if err != nil {
		return SystemInfo{}, err
	}

	// Trim newline characters and convert to string
	cpuSpeedStr := string(cpuSpeed)[:len(cpuSpeed)-1]
	ramStr := string(ram)[:len(ram)-1]

	return SystemInfo{
		CPUSpeed: cpuSpeedStr,
		RAM:      ramStr,
		// Add more fields as needed
	}, nil
}

func systemInfoHandler(w http.ResponseWriter, r *http.Request) {
	info, err := getSystemInfo()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert struct to JSON
	response, err := json.Marshal(info)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Write JSON response
	w.Write(response)
}

func main() {
	http.HandleFunc("/system-info", systemInfoHandler)
	fmt.Println("Server is running on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
