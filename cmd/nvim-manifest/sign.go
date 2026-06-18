package main

import (
	"crypto/ed25519"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"managed-nvim/internal/manifest"
	"os"
)

func runSign(manifestPath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: nvim-manifest sign <private-key-file>")
	}

	keyPath := args[0]

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("reading private key: %w", err)
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil || block.Type != "PRIVATE KEY" {
		return fmt.Errorf("invalid PEM block in private key file")
	}

	if len(block.Bytes) != ed25519.PrivateKeySize {
		return fmt.Errorf("unexpected key size: %d bytes", len(block.Bytes))
	}

	privateKey := ed25519.PrivateKey(block.Bytes)

	data, err := os.ReadFile(manifestPath)

	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	var m manifest.Manifest
	if err := validateJSON(data, &m); err != nil {
		return fmt.Errorf("manifest is not valid json: %w", err)
	}

	sig := ed25519.Sign(privateKey, data)

	sigPath := manifestPath + ".sig"
	if err := os.WriteFile(sigPath, sig, 0644); err != nil {
		return fmt.Errorf("writing signature: %w", err)
	}

	fmt.Printf("manifest signed and saved to %s\n", sigPath)
	fmt.Printf("key: %x\n", privateKey.Public())
	fmt.Printf("signature: %x\n", sig)
	return nil
}

func validateJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
