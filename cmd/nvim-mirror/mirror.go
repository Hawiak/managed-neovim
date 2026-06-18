package main

import (
	"fmt"
	"managed-nvim/internal/manifest"
	"os"
	"os/exec"
	"path/filepath"
)


func mirrorPlugin(p manifest.Plugin, cfg config) error {
	// Already in Gitea, skip
	exists, err := artifactExists(cfg, p.Repo, p.SHA)
	if err != nil {
		return fmt.Errorf("checking artifact existence: %w", err)
	}
	if exists {
		fmt.Printf("  already mirrored, skipping\n")
		return nil
	}

	tmpDir, err := os.MkdirTemp("", "nvim-mirror-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneDir := filepath.Join(tmpDir, "plugin")

	if err := gitClone(p.Upstream, p.Branch, cloneDir); err != nil {
		return fmt.Errorf("cloning %s: %w", p.Upstream, err)
	}

	if err := gitCheckout(cloneDir, p.SHA); err != nil {
		return fmt.Errorf("checking out %s: %w", p.SHA, err)
	}

	archivePath := filepath.Join(tmpDir, "plugin.tar.gz")
	if err := createArchive(cloneDir, archivePath); err != nil {
		return fmt.Errorf("creating archive: %w", err)
	}

	if err := uploadArtifact(cfg, p.Repo, p.SHA, archivePath); err != nil {
		return fmt.Errorf("uploading to gitea: %w", err)
	}

	return nil
}

func gitClone(upstream, branch, destDir string) error {
	cmd := exec.Command("git", "clone",
		"-c", "advice.detachedHead=false",
		"--branch", branch,
		"--single-branch",
		upstream, destDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitCheckout(dir, sha string) error {
	cmd := exec.Command("git", "-C", dir,
		"-c", "advice.detachedHead=false",
		"checkout", sha,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
