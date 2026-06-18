package main

import (
	"fmt"
	"managed-nvim/internal/manifest"
	"regexp"
)

var shaPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)

func runVerify(manifestPath string) error {
	m, err := manifest.Load(manifestPath)
	if err != nil {
		return err
	}

	failed := 0
	for _, p := range m.Plugins {
		if !shaPattern.MatchString(p.SHA) {
			fmt.Printf("FAIL %s - invalid SHA: %s\n", p.Name, p.SHA)
			failed++
			continue
		}

		if p.Repo == "" || p.Upstream == "" {
			fmt.Printf("FAIL %s - missing repo or upstream URL\n", p.Name)
			failed++
			continue
		}
		fmt.Printf("OK    %s @ %s\n", p.Name, p.SHA[:8])
	}
	fmt.Printf("\n%d plugins, %d failed\n", len(m.Plugins), failed)
	if failed > 0 {
		return fmt.Errorf("%d plugin(s) failed verification", failed)
	}
	return nil
}
