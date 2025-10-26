package utils

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"io/ioutil"
	"os"
	"path/filepath"
)

// GetHostKeyCallback returns appropriate SSH host key callback based on security settings
// If insecure is true, returns InsecureIgnoreHostKey (NOT RECOMMENDED)
// Otherwise, attempts to use known_hosts file for verification
func GetHostKeyCallback(insecure bool) (ssh.HostKeyCallback, error) {
	if insecure {
		// WARNING: This disables host key verification - vulnerable to MITM attacks
		return ssh.InsecureIgnoreHostKey(), nil
	}

	// Try to use known_hosts file
	knownHostsPath := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	if _, err := os.Stat(knownHostsPath); err == nil {
		callback, err := knownhosts.New(knownHostsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse known_hosts: %w", err)
		}
		return callback, nil
	}

	// If no known_hosts file exists, return error - user must either:
	// 1. Create known_hosts file, or
	// 2. Use --insecure-ssh flag (with understanding of risks)
	return nil, fmt.Errorf("no known_hosts file found at %s - either create it or use --insecure-ssh flag (NOT RECOMMENDED)", knownHostsPath)
}

// LoadSSHPrivateKey loads SSH private key from file
// Supports RSA, ECDSA, ED25519 keys with or without passphrase
func LoadSSHPrivateKey(keyPath string, passphrase []byte) (ssh.Signer, error) {
	keyData, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key file: %w", err)
	}

	// Try to parse without passphrase first
	signer, err := ssh.ParsePrivateKey(keyData)
	if err == nil {
		return signer, nil
	}

	// If failed and passphrase provided, try with passphrase
	if len(passphrase) > 0 {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH key with passphrase: %w", err)
		}
		return signer, nil
	}

	// Check if key is encrypted but no passphrase provided
	block, _ := pem.Decode(keyData)
	if block != nil && x509.IsEncryptedPEMBlock(block) {
		return nil, fmt.Errorf("SSH key is encrypted but no passphrase provided")
	}

	return nil, fmt.Errorf("failed to parse SSH private key: %w", err)
}

// GetSSHAuthMethods returns SSH authentication methods based on provided credentials
func GetSSHAuthMethods(keyPath string, password string) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	// Prefer key-based authentication
	if keyPath != "" {
		signer, err := LoadSSHPrivateKey(keyPath, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to load SSH key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Add password authentication if provided (as fallback or standalone)
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method provided (need SSH key or password)")
	}

	return authMethods, nil
}
