package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Plugin struct {
	Name        string   `json:"name"`
	Repo        string   `json:"repo"`
	Upstream    string   `json:"upstream"`
	Branch      string   `json:"branch"`
	SHA         string   `json:"sha"`
	ApprovedAt  string   `json:"approved_at"`
	ApprovedBy  string   `json:"approved_by"`
	Permissions []string `json:"permissions"`
}

type Manifest struct {
	SchemaVersion int      `json:"schema_version"`
	OrgName       string   `json:"org_name"`
	LastUpdated   string   `json:"last_updated"`
	Plugins       []Plugin `json:"plugins"`
}

// Load reads and parse plugins.json from the given path
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &m, nil
}

// Save writes the manifest back to disk
func (m *Manifest) Save(path string) error {
	m.LastUpdated = time.Now().Format("2006-01-02")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

// FindByName returns the plugin with the given name, or nil
func (m *Manifest) FindByName(name string) *Plugin {
	for i := range m.Plugins {
		if m.Plugins[i].Name == name {
			return &m.Plugins[i]
		}
	}
	return nil
}
