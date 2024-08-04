package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

//go:embed config/certs/pve.pem
var CertFile embed.FS

func GetCertFile() ([]byte, error) {
	return CertFile.ReadFile("config/certs/pve.pem")
}

func main() {

	pveCA, err := GetCertFile()
	if err != nil {
		log.Panic("Failed to read certificate:", err)
	}

	block, _ := pem.Decode(pveCA)
	if block == nil {
		log.Panic("Failed to parse PEM block containing the certificate")
	}

	pveurl := "https://pve.barton.local:8006/api2/json/nodes/pve/qemu"
	method := "POST"

	// Cloud-init YAML data (properly escaped for URL encoding)
	// 	cloudInitData := `#cloud-config
	// users:
	//   - name: bbarton
	//     sudo: ['ALL=(ALL) NOPASSWD:ALL']
	//     ssh_authorized_keys:
	//       - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCjK7D4W9JnLiPktPatb5P/MHXPUPa9Fn8wy42V0VJPjDdVjy+tDGMpK/rIY1DUqLwnl1d1Xv0XmwvtatBWOuVCEnLlJ7+lm/tZgErWOKF8/YzFJsmdoH76jqSawqOmAD3WN2tY8leKhJygpfkZz62l3VCUHLM30OzHOrny/D4sSt1Xc7K4qO1QlCFMBr64u0nyt7xHDRvWURgMr7sF4UU2o5FozLr+WunlfjgHCF1cM9LfVkD4lchOndSpRfNmbKd9mstTeIY+sXoPBNhg1zNQ/oDQjzos1CoUbLO9/tJYD47ysAacHb1jzBsaJGFauQWz8d4B1S1My4ssKQsSQbVB6xmgq+qkqwsdtHinG/1yBYWAXMwNTgXtDtmN11JlHf++RWARYlogxNvpZxb4Sa60OTVaJXaG7KhMM+93iE/dNMRPHM5phMZXd4nwBG8mhH5a3FJpjR71kMhB+gCYpyCrITbNTY4EAqdrAIbzCibvJ3grlc++JMSX6VEPe7cepYk= bryanbarton@Bryans-MacBook-Pro.local
	//     groups: sudo
	//     shell: /bin/bash
	// package_upgrade: true
	// packages:
	//   - vim
	//   - git
	//   - curl
	// runcmd:
	//   - [apt-get, update]
	//   - [apt-get, install, -y, python3-pip]
	//   - [pip3, install, ansible]
	// `
	// URL encode the YAML
	// encodedCloudInitData := strings.ReplaceAll(cloudInitData, "\n", "\\n")

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
	cert, err := x509.ParseCertificate(block.Bytes) // Replace pemData with the certificate data
	if err != nil {
		// Handle the error
		fmt.Println("Failed to parse certificate:", err)
		return
	}
	certPool.AddCert(cert)

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
