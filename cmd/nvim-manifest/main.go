package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	manifestPath := os.Getenv("MANIFEST_PATH")
	if manifestPath == "" {
		manifestPath = "manifest/plugins.json"
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "add":
		err = runAdd(manifestPath, args)
	case "remove":
		err = runRemove(manifestPath, args)
	case "verify":
		err = runVerify(manifestPath)
	case "sign":
		err = runVerify(manifestPath)
	case "verify-sig":
		err = runVerifySig(manifestPath, args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %s\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`nvim-manifest - managed-nvim plugin manifest tool

	Usage:
	nvim-manifest add <github-repo> <approved-by>
	nvim-manifest remove <plugin-name>
	nvim-manifest verify
	nvim-manifest sign <private-key-file>
	nvim-manifest verify-sig <public-key-file>`)
}
