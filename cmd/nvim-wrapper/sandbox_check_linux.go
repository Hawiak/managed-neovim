//go:build linux

package main

import "fmt"

func runSandboxCheck(_ string) {
	fmt.Println("Linux sandbox (Landlock) is not yet implemented.")
}
