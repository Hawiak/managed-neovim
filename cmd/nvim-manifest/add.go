package main

import (
	"encoding/json"
	"fmt"
	"managed-nvim/internal/manifest"
	"net/http"
	"os"
	"strings"
	"time"
)

func runAdd(manifestPath string, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: nvim-manifest add <owner/repo> <approved-by>")
	}

	repo := args[0]
	sha := args[1]
	approvedBy := args[2]

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("repo must be in owner/name format, got: %s", repo)
	}
	name := parts[1]

	if len(sha) != 40 {
		return fmt.Errorf("sha must be a 400character git commit hash, got :%s", sha)
	}

	m, err := manifest.Load(manifestPath)
	if err != nil {
		return err
	}

	if existing := m.FindByName(name); existing != nil {
		return fmt.Errorf("plugin %q already exists in manifest (sha %s)", name, existing.SHA)
	}

	sha, branch, err := fetchLatestCommit(repo)
	if err != nil {
		return fmt.Errorf("fetching commit for %s: %w", repo, err)
	}

	m.Plugins = append(m.Plugins, manifest.Plugin{
		Name:        name,
		Repo:        repo,
		Upstream:    "https://github.com/" + repo,
		Branch:      branch,
		SHA:         sha,
		ApprovedAt:  time.Now().UTC().Format("2006-01-02"),
		ApprovedBy:  approvedBy,
		Permissions: []string{},
	})

	if err := m.Save(manifestPath); err != nil {
		return err
	}

	fmt.Printf("added %s @ %s (branch: %s)\n", repo, sha[:8], branch)
	return nil
}

func fetchLatestCommit(repo string) (sha, branch string, err error) {
	url := "https://api.github.com/repos/" + repo + "/commits/HEAD"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	if token := getEnv("GITHUB_TOKEN", ""); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned %d status %s", resp.StatusCode, repo)
	}

	var result struct {
		SHA string `json:"sha"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	branch, err = fetchDefaultBranch(repo)
	if err != nil {
		branch = "main"
	}

	return result.SHA, branch, nil
}

func fetchDefaultBranch(repo string) (string, error) {
	url := "https://api.github.com/repos/" + repo

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var result struct {
		DefaultBranch string `json:"default_branch"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.DefaultBranch, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
