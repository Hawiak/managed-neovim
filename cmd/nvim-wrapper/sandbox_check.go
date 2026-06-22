//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type checkResult struct {
	label   string
	passed  bool
	details string
}

func runSandboxCheck(home string) {
	fmt.Println("managed-nvim sandbox check")
	fmt.Println("==========================")

	var results []checkResult

	// Build a small shell script that runs inside sandbox-exec and tries
	// each operation, printing PASS or FAIL for each.
	script := buildCheckScript(home)

	tmp, err := os.CreateTemp("", "nvim-check-*.sh")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot create temp script: %v\n", err)
		os.Exit(1)
	}
	tmp.WriteString(script)
	tmp.Close()
	defer os.Remove(tmp.Name())
	os.Chmod(tmp.Name(), 0700)

	// Run the script inside the same sandbox profile nvim would use.
	profile := generateProfile(home)
	profTmp, err := os.CreateTemp("", "nvim-sandbox-*.sb")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot create sandbox profile: %v\n", err)
		os.Exit(1)
	}
	profTmp.WriteString(profile)
	profTmp.Close()
	defer os.Remove(profTmp.Name())

	out, _ := exec.Command("sandbox-exec", "-f", profTmp.Name(), "/bin/sh", tmp.Name()).Output()

	// Parse the output lines: "LABEL|PASS" or "LABEL|FAIL"
	lines := splitLines(string(out))
	for _, line := range lines {
		parts := splitOnce(line, "|")
		if len(parts) != 2 {
			continue
		}
		results = append(results, checkResult{
			label:  parts[0],
			passed: parts[1] == "PASS",
		})
	}

	// Also check: is ssh-agent available?
	sshSock := os.Getenv("SSH_AUTH_SOCK")
	if sshSock != "" {
		if _, err := os.Stat(sshSock); err == nil {
			results = append(results, checkResult{label: "ssh-agent socket reachable", passed: true})
		} else {
			results = append(results, checkResult{label: "ssh-agent socket reachable", passed: false, details: "socket not found at " + sshSock})
		}
	} else {
		results = append(results, checkResult{label: "ssh-agent socket reachable", passed: false, details: "SSH_AUTH_SOCK not set — git push over SSH will not work from plugins"})
	}

	// Print results
	allPassed := true
	for _, r := range results {
		if r.passed {
			fmt.Printf("  PASS  %s\n", r.label)
		} else {
			fmt.Printf("  FAIL  %s", r.label)
			if r.details != "" {
				fmt.Printf(" (%s)", r.details)
			}
			fmt.Println()
			allPassed = false
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("All checks passed. Sandbox is configured correctly.")
	} else {
		fmt.Println("Some checks failed. Review the output above.")
		os.Exit(1)
	}
}

func buildCheckScript(home string) string {
	sshKey := filepath.Join(home, ".ssh", "id_ed25519")
	sshConfig := filepath.Join(home, ".ssh", "config")
	claudeCreds := filepath.Join(home, ".claude", "credentials")
	claudeDir := filepath.Join(home, ".claude")
	sshDir := filepath.Join(home, ".ssh")

	check := func(label, cmd, wantFail string) string {
		if wantFail == "fail" {
			// Operation should be denied — success means the sandbox is broken
			return fmt.Sprintf(
				`if %s 2>/dev/null; then echo "%s|FAIL"; else echo "%s|PASS"; fi`+"\n",
				cmd, label, label,
			)
		}
		// Operation should succeed
		return fmt.Sprintf(
			`if %s 2>/dev/null; then echo "%s|PASS"; else echo "%s|FAIL"; fi`+"\n",
			cmd, label, label,
		)
	}

	script := "#!/bin/sh\n"

	// Things that must be BLOCKED
	script += check("read SSH private key (must be blocked)", "cat "+sshKey, "fail")
	script += check("read Claude credentials (must be blocked)", "cat "+claudeCreds, "fail")
	script += check("write to .claude dir (must be blocked)", "touch "+claudeDir+"/injected-hook", "fail")
	script += check("write to .ssh dir (must be blocked)", "touch "+sshDir+"/injected", "fail")

	// Things that must WORK
	script += check("read .ssh/config (git needs this)", "cat "+sshConfig, "pass")
	script += check("write to /tmp (nvim needs temp files)", "touch /tmp/nvim-sandbox-write-test && rm /tmp/nvim-sandbox-write-test", "pass")
	script += check("run git binary", "git --version", "pass")
	script += check("read current directory", "ls .", "pass")

	return script
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			if i > start {
				lines = append(lines, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitOnce(s, sep string) []string {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return []string{s[:i], s[i+len(sep):]}
		}
	}
	return []string{s}
}
