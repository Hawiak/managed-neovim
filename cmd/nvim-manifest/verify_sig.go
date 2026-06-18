package main

import (
	"crypto/ed25519"
	"encoding/pem"
	"fmt"
	"os"
)

func runVerifySig(manifestPath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: nvim-manifest verify-sig <public-key-file>")
	}

	keyPath := args[0]

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("reading public key: %w", err)
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return fmt.Errorf("public key file is not valid PEM")
	}
	publicKey := ed25519.PublicKey(block.Bytes)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	sig, err := os.ReadFile(manifestPath + ".sig")
	if err != nil {
		return fmt.Errorf("reading signature: %w", err)
	}

	if !ed25519.Verify(publicKey, data, sig) {
		return fmt.Errorf("signature verification failed, manifest might be tampered with")
	}

	fmt.Println("signature OK")
	return nil
}
