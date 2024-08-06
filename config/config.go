package config

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Proxmox Proxmox `mapstructure:"proxmox"`
}

type Proxmox struct {
	CaPath string `mapstructure:"capath"` // Case-sensitive mapping
	Url    string `mapstructure:"url"`    // Case-sensitive mapping
	Port   string `mapstructure:"port"`   // Case-sensitive mapping
}

func LoadConfig() (Config, error) {
	var config Config

	viper.SetConfigName("config")  // config file name without extension
	viper.AddConfigPath("config/") // look for config in the working directory
	viper.AutomaticEnv()
	viper.SetEnvPrefix("go_prox") // Set the environment variable prefix to GOPROX

	// Read the configuration file
	if err := viper.ReadInConfig(); err != nil {
		return config, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// 6. Get Configuration Values
	pveHost := viper.GetString("url")
	pvePort := viper.GetString("port")
	pveCA := viper.GetString("caPath")

	// 7. Use Configuration Values
	fmt.Println("Proxmox Host:", pveHost)
	fmt.Println("Proxmox Port:", pvePort)
	fmt.Println("Proxmox CA:", pveCA)

	// 8. Unmarshal Configuration into Struct
	if err := viper.Unmarshal(&config); err != nil {
		return config, fmt.Errorf("unable to decode into struct, %v", err)
	}

	fmt.Printf("Config after unmarshal: %+v\n", config) // Print the entire config

	return config, nil
}

// LoadCertificate loads a certificate from a file path specified in an environment variable
func LoadCertificate() (*x509.Certificate, error) {
	// Read the certificate file path from an environment variable
	certFilePath := viper.GetString("proxmox.caPath")
	if certFilePath == "" {
		return nil, fmt.Errorf("CERT_FILE_PATH environment variable not set")
	}

	// Read the certificate file
	certPEM, err := os.ReadFile(certFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		log.Panic("Failed to parse PEM block containing the certificate")
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}
