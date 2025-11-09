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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GitHubRegistry implements Registry using a GitHub repository.
type GitHubRegistry struct {
	// Owner is the GitHub repository owner
	Owner string
	// Repo is the GitHub repository name
	Repo string
	// Branch is the branch to use (default: main)
	Branch string
	// BasePath is the base path in the repo for templates (default: templates)
	BasePath string
	// client is the HTTP client
	client *http.Client
}

// NewGitHubRegistry creates a new GitHub-based registry.
func NewGitHubRegistry(owner, repo string) *GitHubRegistry {
	return &GitHubRegistry{
		Owner:    owner,
		Repo:     repo,
		Branch:   "main",
		BasePath: "templates",
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// List returns all available templates from the GitHub registry.
func (g *GitHubRegistry) List() ([]*TemplateMetadata, error) {
	// Fetch the registry index file
	indexURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s/index.json",
		g.Owner, g.Repo, g.Branch, g.BasePath)

	resp, err := g.client.Get(indexURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry index not found (status %d)", resp.StatusCode)
	}

	var templates []*TemplateMetadata
	if err := json.NewDecoder(resp.Body).Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to parse registry index: %w", err)
	}

	return templates, nil
}

// Search searches for templates by keyword.
func (g *GitHubRegistry) Search(query string) ([]*TemplateMetadata, error) {
	all, err := g.List()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var results []*TemplateMetadata

	for _, tmpl := range all {
		// Search in name, title, description, and tags
		if strings.Contains(strings.ToLower(tmpl.Name), query) ||
			strings.Contains(strings.ToLower(tmpl.Title), query) ||
			strings.Contains(strings.ToLower(tmpl.Description), query) {
			results = append(results, tmpl)
			continue
		}

		// Search in tags
		for _, tag := range tmpl.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, tmpl)
				break
			}
		}
	}

	return results, nil
}

// Get retrieves template content by name.
func (g *GitHubRegistry) Get(name string) (string, error) {
	metadata, err := g.GetMetadata(name)
	if err != nil {
		return "", err
	}

	// Construct raw GitHub URL
	templateURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s/%s",
		g.Owner, g.Repo, g.Branch, g.BasePath, metadata.Path)

	resp, err := g.client.Get(templateURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch template: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("template not found (status %d)", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}

	return string(content), nil
}

// GetMetadata retrieves metadata for a template.
func (g *GitHubRegistry) GetMetadata(name string) (*TemplateMetadata, error) {
	all, err := g.List()
	if err != nil {
		return nil, err
	}

	for _, tmpl := range all {
		if tmpl.Name == name {
			return tmpl, nil
		}
	}

	return nil, fmt.Errorf("template %q not found", name)
}

// Pull downloads a template to local filesystem.
func (g *GitHubRegistry) Pull(name, destination string) error {
	content, err := g.Get(name)
	if err != nil {
		return err
	}

	// Ensure destination directory exists
	dir := filepath.Dir(destination)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Write template to file
	if err := os.WriteFile(destination, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	return nil
}

// ParseGitHubURL parses a GitHub repository URL and returns owner and repo.
// Supports formats:
// - https://github.com/owner/repo
// - github.com/owner/repo
// - owner/repo
func ParseGitHubURL(url string) (owner, repo string, err error) {
	// Remove https:// or http:// prefix
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Remove github.com/ prefix
	url = strings.TrimPrefix(url, "github.com/")

	// Remove trailing .git
	url = strings.TrimSuffix(url, ".git")

	// Split by /
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub URL format: %s", url)
	}

	return parts[0], parts[1], nil
}
