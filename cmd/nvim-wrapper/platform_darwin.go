//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func sandboxActive() bool { return true }

func setImmutable(path string) error {
	return exec.Command("chflags", "uchg", path).Run()
}

func clearImmutable(path string) error {
	return exec.Command("chflags", "nouchg", path).Run()
}

// applySandbox wraps cmd in a sandbox-exec profile that:
//   - Blocks reading raw private key files (SSH keys, tokens)
//   - Blocks writing to AI tool config directories (hook injection prevention)
//   - Allows everything else, including unix socket connections to ssh-agent
func applySandbox(cmd *exec.Cmd, home string) (*exec.Cmd, func()) {
	profile := generateProfile(home)

	tmp, err := os.CreateTemp("", "nvim-sandbox-*.sb")
	if err != nil {
		fmt.Fprintf(os.Stderr, "nvim-wrapper: cannot create sandbox profile: %v\n", err)
		os.Exit(1)
	}
	if _, err := tmp.WriteString(profile); err != nil {
		fmt.Fprintf(os.Stderr, "nvim-wrapper: cannot write sandbox profile: %v\n", err)
		os.Exit(1)
	}
	tmp.Close()

	cleanup := func() { os.Remove(tmp.Name()) }

	// sandbox-exec -f <profile> <nvim-binary> <args...>
	newArgs := append([]string{"-f", tmp.Name(), cmd.Path}, cmd.Args[1:]...)
	wrapped := exec.Command("sandbox-exec", newArgs...)
	wrapped.Stdin = cmd.Stdin
	wrapped.Stdout = cmd.Stdout
	wrapped.Stderr = cmd.Stderr
	wrapped.Env = cmd.Env

	return wrapped, cleanup
}

func generateProfile(home string) string {
	// Collect additional socket paths to annotate in the profile for clarity.
	// SSH_AUTH_SOCK is inherited by all child processes (git subprocesses from
	// neogit/diffview) so they can authenticate without reading the private key.
	sshSockNote := ""
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		sshSockNote = "  ; ssh-agent socket: " + sock
	}

	// GPG agent socket for commit signing.
	gpgSockNote := ""
	gpgSock := home + "/.gnupg/S.gpg-agent"
	if _, err := os.Stat(gpgSock); err == nil {
		gpgSockNote = "  ; gpg-agent socket: " + gpgSock
	}

	lines := []string{
		`(version 1)`,
		`(allow default)`,
		``,
		`; ── Read protection ─────────────────────────────────────────────────────────`,
		`; Raw private key files must never be readable from within Neovim.`,
		`; ~/.ssh/config IS allowed (git needs it). Only key material is blocked.`,
		`; Agent sockets (SSH_AUTH_SOCK, gpg-agent) are unix sockets — not file reads.`,
		`; They remain accessible via the default allow above.`,
		sshSockNote,
		gpgSockNote,
		`(deny file-read*`,
		`  (literal "` + home + `/.ssh/id_rsa")`,
		`  (literal "` + home + `/.ssh/id_ed25519")`,
		`  (literal "` + home + `/.ssh/id_ecdsa")`,
		`  (literal "` + home + `/.ssh/id_dsa")`,
		`  (literal "` + home + `/.claude/credentials")`,
		`  (literal "` + home + `/.aws/credentials")`,
		`)`,
		``,
		`; ── Write protection ────────────────────────────────────────────────────────`,
		`; Belt-and-suspenders on top of chflags uchg immutability.`,
		`; Prevents hook injection into AI tool configs even if chflags is bypassed.`,
		`(deny file-write*`,
		`  (subpath "` + home + `/.claude")`,
		`  (subpath "` + home + `/.copilot")`,
		`  (subpath "` + home + `/.vscode")`,
		`  (literal "` + home + `/.gitconfig")`,
		`  (subpath "` + home + `/.ssh")`,
		`)`,
	}

	// Filter out empty note lines to keep the profile clean.
	var out []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" || l == "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n") + "\n"
}
