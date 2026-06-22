package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func cacheDir(home string) string {
	return filepath.Join(home, ".local", "share", "managed-nvim")
}

func fetchManifest(home string, cfg *Config) error {
	if cfg == nil || cfg.Manifest.URL == "" {
		return fmt.Errorf("no manifest URL configured, please set the 'manifest.url' field in managed-nvim.toml")
	}

	if cfg.Manifest.SigningKey == "" {
		return fmt.Errorf("no manifest signing key configured, please set the 'manifest.signing_key' field in managed-nvim.toml")
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(cfg.Manifest.SigningKey)
	if err != nil {
		return fmt.Errorf("decoding signing key failed: %w", err)
	}

	pubKey := ed25519.PublicKey(pubKeyBytes)

	data, err := httpGet(cfg.Manifest.URL)
	if err != nil {
		return fmt.Errorf("fetching manifest: %w", err)
	}

	sig, err := httpGet(cfg.Manifest.URL + ".sig")
	if err != nil {
		return fmt.Errorf("fetching manifest signature: %w", err)
	}

	if !ed25519.Verify(pubKey, data, sig) {
		return fmt.Errorf("manifest signature verification failed")
	}

	dir := cacheDir(home)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "plugins.json"), data, 0600); err != nil {
		return fmt.Errorf("writing manifest file: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "plugins.json.sig"), sig, 0600); err != nil {
		return fmt.Errorf("writing signature file: %w", err)
	}

	return nil
}

func httpGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
