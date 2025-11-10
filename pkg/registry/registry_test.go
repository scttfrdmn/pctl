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

package registry

import (
	"fmt"
	"testing"
	"time"
)

// mockRegistry is a mock implementation of the Registry interface for testing
type mockRegistry struct {
	templates map[string]*TemplateMetadata
	content   map[string]string
	listErr   error
	searchErr error
	getErr    error
	pullErr   error
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{
		templates: make(map[string]*TemplateMetadata),
		content:   make(map[string]string),
	}
}

func (m *mockRegistry) List() ([]*TemplateMetadata, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*TemplateMetadata
	for _, tmpl := range m.templates {
		result = append(result, tmpl)
	}
	return result, nil
}

func (m *mockRegistry) Search(query string) ([]*TemplateMetadata, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	// Simple substring search on name, title, and description
	var results []*TemplateMetadata
	for _, tmpl := range m.templates {
		if contains(tmpl.Name, query) || contains(tmpl.Title, query) || contains(tmpl.Description, query) {
			results = append(results, tmpl)
		}
	}
	return results, nil
}

func (m *mockRegistry) Get(name string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	if content, ok := m.content[name]; ok {
		return content, nil
	}
	return "", fmt.Errorf("template %q not found", name)
}

func (m *mockRegistry) GetMetadata(name string) (*TemplateMetadata, error) {
	if tmpl, ok := m.templates[name]; ok {
		return tmpl, nil
	}
	return nil, fmt.Errorf("template %q not found", name)
}

func (m *mockRegistry) Pull(name, destination string) error {
	if m.pullErr != nil {
		return m.pullErr
	}
	if _, ok := m.templates[name]; !ok {
		return fmt.Errorf("template %q not found", name)
	}
	// In a real implementation, this would write to destination
	return nil
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTemplateMetadata(t *testing.T) {
	now := time.Now()
	metadata := &TemplateMetadata{
		Name:        "bioinformatics",
		Title:       "Bioinformatics Cluster",
		Description: "Template for bioinformatics workloads",
		Author:      "test-author",
		Version:     "1.0.0",
		Tags:        []string{"bio", "genomics"},
		Source:      "https://github.com/example/templates",
		Path:        "/templates/bio.yaml",
		UpdatedAt:   now,
		Stars:       42,
		Downloads:   1000,
	}

	if metadata.Name != "bioinformatics" {
		t.Errorf("Expected name bioinformatics, got %s", metadata.Name)
	}

	if metadata.Title != "Bioinformatics Cluster" {
		t.Errorf("Expected title 'Bioinformatics Cluster', got %s", metadata.Title)
	}

	if len(metadata.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(metadata.Tags))
	}

	if metadata.Stars != 42 {
		t.Errorf("Expected 42 stars, got %d", metadata.Stars)
	}
}

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.registries == nil {
		t.Error("registries slice is nil")
	}

	if len(manager.registries) != 0 {
		t.Errorf("Expected 0 registries, got %d", len(manager.registries))
	}
}

func TestManagerAddRegistry(t *testing.T) {
	manager := NewManager()
	registry := newMockRegistry()

	manager.AddRegistry(registry)

	if len(manager.registries) != 1 {
		t.Errorf("Expected 1 registry, got %d", len(manager.registries))
	}

	// Add another registry
	registry2 := newMockRegistry()
	manager.AddRegistry(registry2)

	if len(manager.registries) != 2 {
		t.Errorf("Expected 2 registries, got %d", len(manager.registries))
	}
}

func TestManagerList(t *testing.T) {
	manager := NewManager()

	// Create first registry with templates
	reg1 := newMockRegistry()
	reg1.templates["template1"] = &TemplateMetadata{
		Name:  "template1",
		Title: "Template 1",
	}
	reg1.templates["template2"] = &TemplateMetadata{
		Name:  "template2",
		Title: "Template 2",
	}

	// Create second registry with templates
	reg2 := newMockRegistry()
	reg2.templates["template3"] = &TemplateMetadata{
		Name:  "template3",
		Title: "Template 3",
	}

	manager.AddRegistry(reg1)
	manager.AddRegistry(reg2)

	templates, err := manager.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(templates) != 3 {
		t.Errorf("Expected 3 templates, got %d", len(templates))
	}
}

func TestManagerListError(t *testing.T) {
	manager := NewManager()

	// Create registry that returns error
	reg := newMockRegistry()
	reg.listErr = fmt.Errorf("list failed")

	manager.AddRegistry(reg)

	_, err := manager.List()
	if err == nil {
		t.Error("Expected error from List(), got nil")
	}
}

func TestManagerSearch(t *testing.T) {
	manager := NewManager()

	// Create registries with templates
	reg1 := newMockRegistry()
	reg1.templates["bio"] = &TemplateMetadata{
		Name:        "bio",
		Title:       "Bioinformatics",
		Description: "For genomics",
	}
	reg1.templates["ml"] = &TemplateMetadata{
		Name:        "ml",
		Title:       "Machine Learning",
		Description: "For AI workloads",
	}

	reg2 := newMockRegistry()
	reg2.templates["quantum"] = &TemplateMetadata{
		Name:        "quantum",
		Title:       "Quantum Computing",
		Description: "For quantum simulations",
	}

	manager.AddRegistry(reg1)
	manager.AddRegistry(reg2)

	// Search for "bio"
	results, err := manager.Search("bio")
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'bio', got %d", len(results))
	}

	if len(results) > 0 && results[0].Name != "bio" {
		t.Errorf("Expected result 'bio', got %s", results[0].Name)
	}
}

func TestManagerSearchMultipleResults(t *testing.T) {
	manager := NewManager()

	reg := newMockRegistry()
	reg.templates["bio1"] = &TemplateMetadata{
		Name:        "bio1",
		Title:       "Bioinformatics 1",
		Description: "Template 1",
	}
	reg.templates["bio2"] = &TemplateMetadata{
		Name:        "bio2",
		Title:       "Bioinformatics 2",
		Description: "Template 2",
	}

	manager.AddRegistry(reg)

	results, err := manager.Search("bio")
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestManagerSearchWithErrors(t *testing.T) {
	manager := NewManager()

	// Registry that fails
	reg1 := newMockRegistry()
	reg1.searchErr = fmt.Errorf("search failed")

	// Registry that works
	reg2 := newMockRegistry()
	reg2.templates["template1"] = &TemplateMetadata{
		Name:  "template1",
		Title: "Template 1",
	}

	manager.AddRegistry(reg1)
	manager.AddRegistry(reg2)

	// Search should continue with working registry
	results, err := manager.Search("template")
	if err != nil {
		t.Fatalf("Search() should not fail: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestManagerGet(t *testing.T) {
	manager := NewManager()

	reg1 := newMockRegistry()
	reg1.templates["template1"] = &TemplateMetadata{Name: "template1"}
	reg1.content["template1"] = "template content 1"

	reg2 := newMockRegistry()
	reg2.templates["template2"] = &TemplateMetadata{Name: "template2"}
	reg2.content["template2"] = "template content 2"

	manager.AddRegistry(reg1)
	manager.AddRegistry(reg2)

	// Get from first registry
	content, err := manager.Get("template1")
	if err != nil {
		t.Fatalf("Get(template1) failed: %v", err)
	}

	if content != "template content 1" {
		t.Errorf("Expected 'template content 1', got %s", content)
	}

	// Get from second registry
	content, err = manager.Get("template2")
	if err != nil {
		t.Fatalf("Get(template2) failed: %v", err)
	}

	if content != "template content 2" {
		t.Errorf("Expected 'template content 2', got %s", content)
	}
}

func TestManagerGetNotFound(t *testing.T) {
	manager := NewManager()

	reg := newMockRegistry()
	reg.templates["template1"] = &TemplateMetadata{Name: "template1"}

	manager.AddRegistry(reg)

	_, err := manager.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent template, got nil")
	}
}

func TestManagerPull(t *testing.T) {
	manager := NewManager()

	reg := newMockRegistry()
	reg.templates["template1"] = &TemplateMetadata{Name: "template1"}

	manager.AddRegistry(reg)

	err := manager.Pull("template1", "/tmp/dest")
	if err != nil {
		t.Fatalf("Pull() failed: %v", err)
	}
}

func TestManagerPullNotFound(t *testing.T) {
	manager := NewManager()

	reg := newMockRegistry()
	manager.AddRegistry(reg)

	err := manager.Pull("nonexistent", "/tmp/dest")
	if err == nil {
		t.Error("Expected error for nonexistent template, got nil")
	}
}

func TestManagerPullMultipleRegistries(t *testing.T) {
	manager := NewManager()

	// First registry doesn't have the template
	reg1 := newMockRegistry()
	reg1.templates["other"] = &TemplateMetadata{Name: "other"}

	// Second registry has the template
	reg2 := newMockRegistry()
	reg2.templates["template1"] = &TemplateMetadata{Name: "template1"}

	manager.AddRegistry(reg1)
	manager.AddRegistry(reg2)

	// Should find in second registry
	err := manager.Pull("template1", "/tmp/dest")
	if err != nil {
		t.Fatalf("Pull() should succeed: %v", err)
	}
}

func TestDefaultRegistry(t *testing.T) {
	if DefaultRegistry == "" {
		t.Error("DefaultRegistry should not be empty")
	}

	if DefaultRegistry != "https://github.com/scttfrdmn/pctl-registry" {
		t.Errorf("Expected default registry URL, got %s", DefaultRegistry)
	}
}

func TestManagerEmptySearch(t *testing.T) {
	manager := NewManager()

	reg := newMockRegistry()
	reg.templates["template1"] = &TemplateMetadata{
		Name:        "template1",
		Title:       "Template 1",
		Description: "Description 1",
	}

	manager.AddRegistry(reg)

	// Empty search should return no results
	results, err := manager.Search("")
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty search, got %d", len(results))
	}
}

func TestManagerNoRegistries(t *testing.T) {
	manager := NewManager()

	// List with no registries
	templates, err := manager.List()
	if err != nil {
		t.Fatalf("List() should not fail with no registries: %v", err)
	}

	if len(templates) != 0 {
		t.Errorf("Expected 0 templates, got %d", len(templates))
	}

	// Search with no registries
	results, err := manager.Search("test")
	if err != nil {
		t.Fatalf("Search() should not fail with no registries: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}

	// Get with no registries
	_, err = manager.Get("test")
	if err == nil {
		t.Error("Get() should fail with no registries")
	}
}

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "https URL",
			url:       "https://github.com/scttfrdmn/pctl",
			wantOwner: "scttfrdmn",
			wantRepo:  "pctl",
			wantErr:   false,
		},
		{
			name:      "http URL",
			url:       "http://github.com/scttfrdmn/pctl",
			wantOwner: "scttfrdmn",
			wantRepo:  "pctl",
			wantErr:   false,
		},
		{
			name:      "github.com URL",
			url:       "github.com/scttfrdmn/pctl",
			wantOwner: "scttfrdmn",
			wantRepo:  "pctl",
			wantErr:   false,
		},
		{
			name:      "short form",
			url:       "scttfrdmn/pctl",
			wantOwner: "scttfrdmn",
			wantRepo:  "pctl",
			wantErr:   false,
		},
		{
			name:      "with .git suffix",
			url:       "https://github.com/scttfrdmn/pctl.git",
			wantOwner: "scttfrdmn",
			wantRepo:  "pctl",
			wantErr:   false,
		},
		{
			name:      "with trailing slash and .git",
			url:       "github.com/scttfrdmn/pctl.git",
			wantOwner: "scttfrdmn",
			wantRepo:  "pctl",
			wantErr:   false,
		},
		{
			name:      "invalid URL - no repo",
			url:       "scttfrdmn",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
		{
			name:      "invalid URL - empty",
			url:       "",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseGitHubURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseGitHubURL() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseGitHubURL() unexpected error = %v", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("ParseGitHubURL() owner = %v, want %v", owner, tt.wantOwner)
			}

			if repo != tt.wantRepo {
				t.Errorf("ParseGitHubURL() repo = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}

func TestNewGitHubRegistry(t *testing.T) {
	reg := NewGitHubRegistry("scttfrdmn", "pctl")

	if reg == nil {
		t.Fatal("NewGitHubRegistry() returned nil")
	}

	if reg.Owner != "scttfrdmn" {
		t.Errorf("Owner = %s, want scttfrdmn", reg.Owner)
	}

	if reg.Repo != "pctl" {
		t.Errorf("Repo = %s, want pctl", reg.Repo)
	}

	if reg.Branch != "main" {
		t.Errorf("Branch = %s, want main", reg.Branch)
	}

	if reg.BasePath != "templates" {
		t.Errorf("BasePath = %s, want templates", reg.BasePath)
	}

	if reg.client == nil {
		t.Error("client is nil")
	}
}
