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

package ami

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/scttfrdmn/petal/pkg/template"
)

// CacheEntry represents a cached AMI entry.
type CacheEntry struct {
	AMIID        string    `json:"ami_id"`
	Region       string    `json:"region"`
	Fingerprint  string    `json:"fingerprint"`
	TemplateName string    `json:"template_name,omitempty"`
	Created      time.Time `json:"created"`
	LastUsed     time.Time `json:"last_used"`
}

// Cache manages local AMI cache storage.
type Cache struct {
	cacheFile string
	entries   map[string]CacheEntry // key: region:fingerprint
}

// NewCache creates a new AMI cache.
func NewCache() (*Cache, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	pctlDir := filepath.Join(homeDir, ".pctl")
	if err := os.MkdirAll(pctlDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .pctl directory: %w", err)
	}

	cacheFile := filepath.Join(pctlDir, "ami-cache.json")
	cache := &Cache{
		cacheFile: cacheFile,
		entries:   make(map[string]CacheEntry),
	}

	// Load existing cache
	if err := cache.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load cache: %w", err)
	}

	return cache, nil
}

// Get retrieves a cached AMI for a fingerprint in a specific region.
func (c *Cache) Get(region string, fingerprint *template.AMIFingerprint) (*CacheEntry, bool) {
	key := c.makeKey(region, fingerprint.Hash)
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	// Update last used time
	entry.LastUsed = time.Now()
	c.entries[key] = entry

	return &entry, true
}

// Add adds an AMI to the cache.
func (c *Cache) Add(region, amiID string, fingerprint *template.AMIFingerprint, templateName string) error {
	key := c.makeKey(region, fingerprint.Hash)
	entry := CacheEntry{
		AMIID:        amiID,
		Region:       region,
		Fingerprint:  fingerprint.Hash,
		TemplateName: templateName,
		Created:      time.Now(),
		LastUsed:     time.Now(),
	}

	c.entries[key] = entry

	return c.save()
}

// Remove removes an AMI from the cache.
func (c *Cache) Remove(region, amiID string) error {
	// Find and remove by AMI ID
	for key, entry := range c.entries {
		if entry.Region == region && entry.AMIID == amiID {
			delete(c.entries, key)
			break
		}
	}

	return c.save()
}

// List returns all cached AMI entries.
func (c *Cache) List() []CacheEntry {
	entries := make([]CacheEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		entries = append(entries, entry)
	}
	return entries
}

// Clean removes expired or invalid entries from the cache.
func (c *Cache) Clean(maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	for key, entry := range c.entries {
		if entry.LastUsed.Before(cutoff) {
			delete(c.entries, key)
		}
	}

	return c.save()
}

// load loads the cache from disk.
func (c *Cache) load() error {
	data, err := os.ReadFile(c.cacheFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &c.entries)
}

// save saves the cache to disk.
func (c *Cache) save() error {
	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(c.cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// makeKey creates a cache key from region and fingerprint hash.
func (c *Cache) makeKey(region, fingerprintHash string) string {
	return fmt.Sprintf("%s:%s", region, fingerprintHash)
}

// FindAMIByFingerprint searches for an existing AMI with the given fingerprint.
// It checks the local cache first, then queries AWS.
func (m *Manager) FindAMIByFingerprint(ctx context.Context, fingerprint *template.AMIFingerprint) (string, error) {
	// Initialize cache
	cache, err := NewCache()
	if err != nil {
		return "", fmt.Errorf("failed to initialize cache: %w", err)
	}

	region := m.builder.region

	// Check local cache first
	if entry, ok := cache.Get(region, fingerprint); ok {
		// Verify AMI still exists in AWS
		if m.amiExists(ctx, entry.AMIID) {
			return entry.AMIID, nil
		}
		// AMI no longer exists, remove from cache
		cache.Remove(region, entry.AMIID)
	}

	// Query AWS for AMI with matching fingerprint tag
	result, err := m.builder.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners: []string{"self"},
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:pctl:fingerprint"),
				Values: []string{fingerprint.Hash},
			},
			{
				Name:   aws.String("tag:ManagedBy"),
				Values: []string{"pctl"},
			},
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to query AMIs: %w", err)
	}

	// No matching AMI found
	if len(result.Images) == 0 {
		return "", nil
	}

	// Use the most recent AMI if multiple exist
	mostRecent := result.Images[0]
	for _, img := range result.Images {
		if img.CreationDate != nil && mostRecent.CreationDate != nil {
			if *img.CreationDate > *mostRecent.CreationDate {
				mostRecent = img
			}
		}
	}

	amiID := *mostRecent.ImageId

	// Add to cache
	templateName := ""
	for _, tag := range mostRecent.Tags {
		if tag.Key != nil && *tag.Key == "TemplateName" && tag.Value != nil {
			templateName = *tag.Value
			break
		}
	}
	cache.Add(region, amiID, fingerprint, templateName)

	return amiID, nil
}

// amiExists checks if an AMI exists in AWS.
func (m *Manager) amiExists(ctx context.Context, amiID string) bool {
	result, err := m.builder.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	})

	if err != nil {
		return false
	}

	return len(result.Images) > 0
}
