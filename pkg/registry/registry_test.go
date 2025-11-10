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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestGitHubRegistryList(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/templates/index.json") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		templates := []*TemplateMetadata{
			{
				Name:        "bioinformatics",
				Title:       "Bioinformatics Cluster",
				Description: "Template for genomics workloads",
				Tags:        []string{"bio", "genomics"},
			},
			{
				Name:        "ml",
				Title:       "Machine Learning",
				Description: "Template for ML workloads",
				Tags:        []string{"ml", "ai"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(templates)
	}))
	defer server.Close()

	// Create registry pointing to test server
	reg := NewGitHubRegistry("test", "repo")
	// Override client to use test server URL
	reg.client = &http.Client{
		Transport: &testTransport{
			baseURL: server.URL,
			owner:   "test",
			repo:    "repo",
			branch:  "main",
		},
	}

	templates, err := reg.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(templates) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(templates))
	}

	if templates[0].Name != "bioinformatics" {
		t.Errorf("Expected first template 'bioinformatics', got %s", templates[0].Name)
	}
}

func TestGitHubRegistryListNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	reg := NewGitHubRegistry("test", "repo")
	reg.client = &http.Client{
		Transport: &testTransport{
			baseURL: server.URL,
			owner:   "test",
			repo:    "repo",
			branch:  "main",
		},
	}

	_, err := reg.List()
	if err == nil {
		t.Error("Expected error for 404, got nil")
	}

	if !strings.Contains(err.Error(), "registry index not found") {
		t.Errorf("Expected 'registry index not found' error, got: %v", err)
	}
}

func TestGitHubRegistrySearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		templates := []*TemplateMetadata{
			{
				Name:        "bioinformatics",
				Title:       "Bioinformatics Cluster",
				Description: "Genomics workloads",
				Tags:        []string{"bio", "genomics"},
			},
			{
				Name:        "ml",
				Title:       "Machine Learning",
				Description: "AI workloads",
				Tags:        []string{"ml", "ai"},
			},
			{
				Name:        "chemistry",
				Title:       "Chemistry Cluster",
				Description: "Computational chemistry",
				Tags:        []string{"chem", "molecular"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(templates)
	}))
	defer server.Close()

	reg := NewGitHubRegistry("test", "repo")
	reg.client = &http.Client{
		Transport: &testTransport{
			baseURL: server.URL,
			owner:   "test",
			repo:    "repo",
			branch:  "main",
		},
	}

	// Search by name
	results, err := reg.Search("bio")
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'bio', got %d", len(results))
	}

	if len(results) > 0 && results[0].Name != "bioinformatics" {
		t.Errorf("Expected 'bioinformatics', got %s", results[0].Name)
	}

	// Search by tag
	results, err = reg.Search("molecular")
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'molecular', got %d", len(results))
	}

	// Search by description
	results, err = reg.Search("AI")
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'AI', got %d", len(results))
	}

	// Search with no results
	results, err = reg.Search("quantum")
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for 'quantum', got %d", len(results))
	}
}

func TestGitHubRegistryGetMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		templates := []*TemplateMetadata{
			{
				Name:  "template1",
				Title: "Template 1",
				Path:  "template1.yaml",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(templates)
	}))
	defer server.Close()

	reg := NewGitHubRegistry("test", "repo")
	reg.client = &http.Client{
		Transport: &testTransport{
			baseURL: server.URL,
			owner:   "test",
			repo:    "repo",
			branch:  "main",
		},
	}

	// Test found
	metadata, err := reg.GetMetadata("template1")
	if err != nil {
		t.Fatalf("GetMetadata() failed: %v", err)
	}

	if metadata.Name != "template1" {
		t.Errorf("Expected name 'template1', got %s", metadata.Name)
	}

	if metadata.Path != "template1.yaml" {
		t.Errorf("Expected path 'template1.yaml', got %s", metadata.Path)
	}

	// Test not found
	_, err = reg.GetMetadata("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent template, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestGitHubRegistryGet(t *testing.T) {
	templateContent := "cluster:\n  name: test\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/index.json") {
			templates := []*TemplateMetadata{
				{
					Name: "template1",
					Path: "template1.yaml",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(templates)
		} else if strings.HasSuffix(r.URL.Path, "/template1.yaml") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(templateContent))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	reg := NewGitHubRegistry("test", "repo")
	reg.client = &http.Client{
		Transport: &testTransport{
			baseURL: server.URL,
			owner:   "test",
			repo:    "repo",
			branch:  "main",
		},
	}

	content, err := reg.Get("template1")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if content != templateContent {
		t.Errorf("Expected content %q, got %q", templateContent, content)
	}
}

func TestGitHubRegistryGetNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/index.json") {
			templates := []*TemplateMetadata{}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(templates)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	reg := NewGitHubRegistry("test", "repo")
	reg.client = &http.Client{
		Transport: &testTransport{
			baseURL: server.URL,
			owner:   "test",
			repo:    "repo",
			branch:  "main",
		},
	}

	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent template, got nil")
	}
}

func TestGitHubRegistryPull(t *testing.T) {
	templateContent := "cluster:\n  name: test\n"
	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "subdir", "template.yaml")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/index.json") {
			templates := []*TemplateMetadata{
				{
					Name: "template1",
					Path: "template1.yaml",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(templates)
		} else if strings.HasSuffix(r.URL.Path, "/template1.yaml") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(templateContent))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	reg := NewGitHubRegistry("test", "repo")
	reg.client = &http.Client{
		Transport: &testTransport{
			baseURL: server.URL,
			owner:   "test",
			repo:    "repo",
			branch:  "main",
		},
	}

	err := reg.Pull("template1", destination)
	if err != nil {
		t.Fatalf("Pull() failed: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("Failed to read pulled file: %v", err)
	}

	if string(content) != templateContent {
		t.Errorf("Expected content %q, got %q", templateContent, string(content))
	}
}

func TestGitHubRegistryPullNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "template.yaml")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/index.json") {
			templates := []*TemplateMetadata{}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(templates)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	reg := NewGitHubRegistry("test", "repo")
	reg.client = &http.Client{
		Transport: &testTransport{
			baseURL: server.URL,
			owner:   "test",
			repo:    "repo",
			branch:  "main",
		},
	}

	err := reg.Pull("nonexistent", destination)
	if err == nil {
		t.Error("Expected error for nonexistent template, got nil")
	}
}

// testTransport is a custom HTTP transport for testing that rewrites GitHub URLs to test server URLs
type testTransport struct {
	baseURL string
	owner   string
	repo    string
	branch  string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite GitHub raw URL to test server URL
	expectedPrefix := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/", t.owner, t.repo, t.branch)
	if strings.HasPrefix(req.URL.String(), expectedPrefix) {
		suffix := strings.TrimPrefix(req.URL.String(), expectedPrefix)
		newURL := t.baseURL + "/" + suffix
		newReq, err := http.NewRequest(req.Method, newURL, req.Body)
		if err != nil {
			return nil, err
		}
		return http.DefaultTransport.RoundTrip(newReq)
	}

	return http.DefaultTransport.RoundTrip(req)
}
