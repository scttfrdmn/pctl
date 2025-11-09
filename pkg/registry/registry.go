// Copyright 2025 Scott Friedman
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package registry provides template registry and discovery features.
package registry

import (
	"fmt"
	"time"
)

// TemplateMetadata contains information about a registry template.
type TemplateMetadata struct {
	// Name is the template name (e.g., "bioinformatics")
	Name string `json:"name"`
	// Title is the human-readable title
	Title string `json:"title"`
	// Description is a brief description
	Description string `json:"description"`
	// Author is the template author
	Author string `json:"author"`
	// Version is the template version
	Version string `json:"version"`
	// Tags are searchable tags
	Tags []string `json:"tags"`
	// Source is the source URL (e.g., GitHub repo)
	Source string `json:"source"`
	// Path is the path to the template file in the source
	Path string `json:"path"`
	// UpdatedAt is when the template was last updated
	UpdatedAt time.Time `json:"updated_at"`
	// Stars is the number of stars/likes
	Stars int `json:"stars,omitempty"`
	// Downloads is the download count
	Downloads int `json:"downloads,omitempty"`
}

// Registry defines the interface for template registries.
type Registry interface {
	// List returns all available templates
	List() ([]*TemplateMetadata, error)

	// Search searches for templates by keyword
	Search(query string) ([]*TemplateMetadata, error)

	// Get retrieves template content by name
	Get(name string) (string, error)

	// GetMetadata retrieves metadata for a template
	GetMetadata(name string) (*TemplateMetadata, error)

	// Pull downloads a template to local filesystem
	Pull(name, destination string) error
}

// DefaultRegistry is the default template registry URL.
const DefaultRegistry = "https://github.com/scttfrdmn/pctl-registry"

// Manager manages template registries.
type Manager struct {
	registries []Registry
}

// NewManager creates a new registry manager.
func NewManager() *Manager {
	return &Manager{
		registries: []Registry{},
	}
}

// AddRegistry adds a registry to the manager.
func (m *Manager) AddRegistry(r Registry) {
	m.registries = append(m.registries, r)
}

// List lists all templates from all registries.
func (m *Manager) List() ([]*TemplateMetadata, error) {
	var all []*TemplateMetadata
	for _, reg := range m.registries {
		templates, err := reg.List()
		if err != nil {
			return nil, fmt.Errorf("failed to list from registry: %w", err)
		}
		all = append(all, templates...)
	}
	return all, nil
}

// Search searches all registries for templates.
func (m *Manager) Search(query string) ([]*TemplateMetadata, error) {
	var results []*TemplateMetadata
	for _, reg := range m.registries {
		templates, err := reg.Search(query)
		if err != nil {
			// Log error but continue with other registries
			continue
		}
		results = append(results, templates...)
	}
	return results, nil
}

// Get retrieves a template by name from the first registry that has it.
func (m *Manager) Get(name string) (string, error) {
	for _, reg := range m.registries {
		content, err := reg.Get(name)
		if err == nil {
			return content, nil
		}
	}
	return "", fmt.Errorf("template %q not found in any registry", name)
}

// Pull downloads a template to the local filesystem.
func (m *Manager) Pull(name, destination string) error {
	for _, reg := range m.registries {
		err := reg.Pull(name, destination)
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("template %q not found in any registry", name)
}
