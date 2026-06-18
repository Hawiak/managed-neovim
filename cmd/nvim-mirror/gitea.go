package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 120 * time.Second}

// packageName converts a repo path like "folke/lazy.nvim" into a Gitea
// generic package name "folke-lazy.nvim" (slashes not allowed in package names).
func packageName(repo string) string {
	return strings.ReplaceAll(repo, "/", "-")
}

// packageURL builds the Gitea generic package API URL for a plugin archive.
// Structure: /api/packages/{owner}/generic/{package-name}/{sha}/plugin.tar.gz
func packageURL(cfg config, repo, sha string) string {
	return fmt.Sprintf("%s/api/packages/%s/generic/%s/%s/plugin.tar.gz",
		cfg.giteaURL, cfg.giteaOwner, packageName(repo), sha)
}

// artifactExists checks whether a plugin archive is already uploaded.
func artifactExists(cfg config, repo, sha string) (bool, error) {
	req, err := http.NewRequest("HEAD", packageURL(cfg, repo, sha), nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "token "+cfg.giteaToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// uploadArtifact streams a local .tar.gz file to Gitea's generic package registry.
func uploadArtifact(cfg config, repo, sha, localFile string) error {
	f, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	req, err := http.NewRequest("PUT", packageURL(cfg, repo, sha), f)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+cfg.giteaToken)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gitea returned %d for %s @ %s", resp.StatusCode, repo, sha[:8])
	}

	return nil
}
