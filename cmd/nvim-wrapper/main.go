package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"managed-nvim/internal/manifest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

var protectedFiles = []string{
	".claude/settings.json",
	".copilot",
	".vscode/settings.json",
	".vscode/tasks.json",
	".gitconfig",
	".ssh/config",
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "nvim-wrapper: cannot determine home dir: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "--sandbox-check" {
		runSandboxCheck(home)
		return
	}

	nvimPath := findRealNvim()

	exportManagedEnv(home)

	protected := resolveExisting(home, protectedFiles)

	before := hashFiles(protected)

	// Locks all protected files, so that they cannot be modified while Neovim is running
	for _, p := range protected {
		if err := setImmutable(p); err != nil {
			fmt.Fprintf(os.Stderr, "nvim-wrapper: warning: could not lock %s: %v\n", p, err)
		}
	}

	// Run Neovim, fowards all args, stdin, stdout, stderr etc
	exitCode := runNvim(nvimPath, os.Args[1:], home)

	// Always unlock, eve if Neovim creashes
	for _, p := range protected {
		clearImmutable(p)
	}

	// Intergrity check - was anything modified despite the lock?
	after := hashFiles(protected)
	for path, hash := range before {
		if after[path] != hash {
			fmt.Fprintf(os.Stderr, "[managed-nvim] Security violation: %s was modified during session\n", path)
		}
	}
	os.Exit(exitCode)
}

// looks for the real NVIM binary Checks for nvim.real binary or a path to the real nvim in NVIM_REAL env var. Exits with error if not found.
func findRealNvim() string {
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "nvim.real")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	if v := os.Getenv("NVIM_REAL"); v != "" {
		return v
	}

	if path, err := exec.LookPath("nvim"); err == nil {
		return path
	}

	fmt.Fprintf(os.Stderr, "nvim-wrapper: cannot find real nvim binary. Set NVIUM_REAL or place nvim.real next to this binary.\n")
	os.Exit(1)
	return ""
}

// ResolveExisting returns abollute paths for protected files that exist.
func resolveExisting(home string, files []string) []string {
	var result []string
	for _, rel := range files {
		abs := filepath.Join(home, rel)
		if _, err := os.Stat(abs); err == nil {
			result = append(result, abs)
		}
	}
	return result
}

// HashFiles returns a map of path -> sha256 hex for each path. If a file cannot be read, it is skipped.
func hashFiles(paths []string) map[string]string {
	hashes := make(map[string]string)
	for _, p := range paths {
		if h, err := hashPath(p); err == nil {
			hashes[p] = h
		}
	}

	return hashes
}

// hashPoth hases a file's content, or a directories entry list
func hashPath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return "", err
		}

		h := sha256.New()
		for _, e := range entries {
			fmt.Fprintf(h, "%s:%v\n", e.Name(), e.Type())
		}

		return hex.EncodeToString(h.Sum(nil)), nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil

}

// exportManagedEnv sets environment variables that the Lua plugin reads to
// build the :ManagedNeovim status page. Called before launching nvim so the
// values are inherited by the child process.
func exportManagedEnv(home string) {
	exe, _ := os.Executable()
	os.Setenv("MANAGED_NVIM", "1")
	os.Setenv("MANAGED_NVIM_RUNTIME", filepath.Join(filepath.Dir(exe), "nvim"))

	if sandboxActive() {
		os.Setenv("MANAGED_NVIM_SANDBOX", "1")
	} else {
		os.Setenv("MANAGED_NVIM_SANDBOX", "0")
	}

	manifestPath := os.Getenv("MANAGED_NVIM_MANIFEST_PATH")
	if manifestPath == "" {
		// Default: next to the wrapper binary, then fall back to the repo layout.
		exe, _ := os.Executable()
		candidates := []string{
			filepath.Join(filepath.Dir(exe), "manifest", "plugins.json"),
			filepath.Join(home, ".local", "share", "managed-nvim", "plugins.json"),
			"/opt/managed-nvim/manifest/plugins.json",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				manifestPath = c
				break
			}
		}
	}

	os.Setenv("MANAGED_NVIM_MANIFEST_PATH", manifestPath)

	if manifestPath == "" {
		return
	}

	m, err := manifest.Load(manifestPath)
	if err != nil {
		return
	}

	os.Setenv("MANAGED_NVIM_ORG", m.OrgName)
	os.Setenv("MANAGED_NVIM_MANIFEST_DATE", m.LastUpdated)
	os.Setenv("MANAGED_NVIM_PLUGIN_COUNT", strconv.Itoa(len(m.Plugins)))
	os.Setenv("MANAGED_NVIM_SCHEMA_VERSION", strconv.Itoa(m.SchemaVersion))
}

// Runs nvim in a subprocess, wrapped in the OS sandbox.
func runNvim(nvimPath string, args []string, home string) int {
	cmd := exec.Command(nvimPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd, cleanup := applySandbox(cmd, home)
	defer cleanup()

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}
