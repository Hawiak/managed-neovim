package main

import (
	"fmt"
	"managed-nvim/internal/manifest"
)

func runRemove(manifestPath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: nvim-manifest remove <plugin-name>")
	}
	name := args[0]

	m, err := manifest.Load(manifestPath)
	if err != nil {
		return err
	}

	before := len(m.Plugins)
	filtered := m.Plugins[:0]
	for _, p := range m.Plugins {
		if p.Name != name {
			filtered = append(filtered, p)
		}
	}

	m.Plugins = filtered

	if len(m.Plugins) == before {
		return fmt.Errorf("plugin %q not found in manifest", name)
	}

	if err := m.Save(manifestPath); err != nil {
		return err
	}

	fmt.Printf("removed plugin %q\n", name)
	return nil
}
