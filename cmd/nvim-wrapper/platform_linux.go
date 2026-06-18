//go:build linux

package main

import "os/exec"

func sandboxActive() bool { return false } // true once Landlock is implemented

func setImmutable(path string) error {
	return exec.Command("chattr", "+i", path).Run()
}

func clearImmutable(path string) error {
	return exec.Command("chattr", "-i", path).Run()
}

// applySandbox on Linux is a no-op until Landlock is implemented (Part 7).
// Landlock controls filesystem path access only — it does not restrict unix
// socket connections, so SSH_AUTH_SOCK-based git authentication already works
// without any special configuration.
func applySandbox(cmd *exec.Cmd, home string) (*exec.Cmd, func()) {
	return cmd, func() {}
}
