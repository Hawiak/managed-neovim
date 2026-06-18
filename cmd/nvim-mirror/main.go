package main

import (
	"fmt"
	"managed-nvim/internal/manifest"
	"os"
)

type config struct {
	giteaURL     string
	giteaOwner   string
	giteaToken   string
	manifestPath string
}

func main() {
	cfg := config{
		giteaURL:     envOr("GITEA_URL", "http://localhost:2222"),
		giteaOwner:   requireEnv("GITEA_OWNER"),
		giteaToken:   requireEnv("GITEA_TOKEN"),
		manifestPath: envOr("MANIFEST_PATH", "manifest/plugins.json"),
	}

	m, err := manifest.Load(cfg.manifestPath)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading manifest: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Mirroring %d plugins... \n\n", len(m.Plugins))

	failures := 0

	for _, p := range m.Plugins {
		fmt.Printf("-> %s @ %s\n", p.Repo, p.SHA[:8])
		if err := mirrorPlugin(p, cfg); err != nil {
			fmt.Printf("	FAIL: %v\n", err)
			failures++
		} else {
			fmt.Printf("	OK\n")
		}
	}

	fmt.Printf("\ndone: %d ok, %d failed\n", len(m.Plugins)-failures, failures)
	if failures > 0 {
		os.Exit(1)
	}
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "required environment variable not set: %s\n", key)
		os.Exit(1)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
