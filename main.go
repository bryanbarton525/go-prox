package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bryanbarton525/go-prox/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	pveurl := cfg.Proxmox.Url + ":" + cfg.Proxmox.Port + "/api2/json/nodes/pve/qemu"
	method := "POST"

	// VM Parameters (using a map for clarity)
	params := map[string]string{
		"vmid":    "123",
		"name":    "my-ubuntu-vm",
		"memory":  "2048",
		"cores":   "2",
		"sockets": "1",
		"cpu":     "host",
		"net0":    "virtio,bridge=vmbr0",
		"ostype":  "l26",
		"storage": "local-lvm",
		"virtio0": "local-lvm:15",
		// "ide0":	    "local-lvm:vm-123-cloudinit,media=cdrom",
		"ide2": "local:iso/ubuntu-22.04.3-live-server-amd64.iso,media=cdrom", // Correct image path
		// "cicustom": encodedCloudInitData,                                         // Cloud-init data
	}

	// Encode parameters into URL-encoded format
	data := url.Values{}
	for key, value := range params {
		data.Set(key, value)
	}

	// Payload Reader
	payload := strings.NewReader(data.Encode())
	// Load the server's certificate from a file or other source
	certPool := x509.NewCertPool()

	// Load the certificate
	pveCA, err := config.LoadCertificate()
	if err != nil {
		// Handle the error
		fmt.Println("Failed to parse certificate:", err)
		return
	}
	certPool.AddCert(pveCA)

	// Create a transport with the custom certificate pool
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: certPool, ServerName: "pve.barton.local"},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest(method, pveurl, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "PVEAPIToken=automation@pam!vm-automation=22659b1a-9e00-4ecf-bd0b-381e52c21e8e")

	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error making request: %v\n", err) // Log error and exit
	}
	defer res.Body.Close()

	// Check response status code
	if res.StatusCode != http.StatusOK {
		log.Fatalf("Proxmox API returned error status code: %d\n", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v\n", err) // Log error and exit
	}

	// Decode JSON response
	var response struct {
		Data   interface{}       `json:"data"`
		Errors map[string]string `json:"errors"`
	}

	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&response); err != nil {
		log.Fatalf("Error decoding JSON response: %v\n", err)
	}

	if response.Errors != nil {
		log.Printf("Error from Proxmox: %v\n", response.Errors) // Log error but don't exit
	}

	fmt.Println(string(body))
}
