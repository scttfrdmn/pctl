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

package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/scttfrdmn/pctl/pkg/registry"
	"github.com/spf13/cobra"
)

var (
	registryURL string
)

// registryCmd represents the registry command
var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Manage template registry",
	Long: `Manage the template registry for discovering and sharing cluster templates.

The registry provides a curated collection of templates for common HPC workloads
including bioinformatics, machine learning, computational chemistry, and more.`,
}

// registryListCmd lists templates in the registry
var registryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available templates",
	Long: `List all available templates in the registry.

Templates are organized by category and include metadata like author, version,
and popularity metrics.`,
	RunE: runRegistryList,
}

// registrySearchCmd searches for templates
var registrySearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for templates",
	Long: `Search for templates by keyword.

Searches template names, titles, descriptions, and tags.`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrySearch,
}

// registryPullCmd downloads a template
var registryPullCmd = &cobra.Command{
	Use:   "pull [template-name] [destination]",
	Short: "Download a template",
	Long: `Download a template from the registry to your local filesystem.

Example:
  pctl registry pull bioinformatics ./my-cluster.yaml`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runRegistryPull,
}

func init() {
	rootCmd.AddCommand(registryCmd)
	registryCmd.AddCommand(registryListCmd)
	registryCmd.AddCommand(registrySearchCmd)
	registryCmd.AddCommand(registryPullCmd)

	// Add registry URL flag
	registryCmd.PersistentFlags().StringVarP(&registryURL, "registry", "r", registry.DefaultRegistry,
		"registry URL (GitHub repository)")
}

func createRegistryManager() (*registry.Manager, error) {
	manager := registry.NewManager()

	// Parse registry URL and create GitHub registry
	owner, repo, err := registry.ParseGitHubURL(registryURL)
	if err != nil {
		return nil, fmt.Errorf("invalid registry URL: %w", err)
	}

	githubReg := registry.NewGitHubRegistry(owner, repo)
	manager.AddRegistry(githubReg)

	return manager, nil
}

func runRegistryList(cmd *cobra.Command, args []string) error {
	manager, err := createRegistryManager()
	if err != nil {
		return err
	}

	fmt.Printf("Fetching templates from registry...\n\n")

	templates, err := manager.List()
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	if len(templates) == 0 {
		fmt.Println("No templates found in registry.")
		return nil
	}

	// Print templates in a table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "NAME\tTITLE\tAUTHOR\tVERSION\tUPDATED\n")
	fmt.Fprintf(w, "────\t─────\t──────\t───────\t───────\n")

	for _, tmpl := range templates {
		updated := formatTimeAgo(tmpl.UpdatedAt)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			tmpl.Name, tmpl.Title, tmpl.Author, tmpl.Version, updated)
	}

	w.Flush()

	fmt.Printf("\nTotal: %d templates\n", len(templates))
	fmt.Printf("\nUse 'pctl registry pull <name> <destination>' to download a template.\n")

	return nil
}

func runRegistrySearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	manager, err := createRegistryManager()
	if err != nil {
		return err
	}

	fmt.Printf("Searching for '%s'...\n\n", query)

	templates, err := manager.Search(query)
	if err != nil {
		return fmt.Errorf("failed to search templates: %w", err)
	}

	if len(templates) == 0 {
		fmt.Printf("No templates found matching '%s'.\n", query)
		return nil
	}

	// Print results
	for _, tmpl := range templates {
		fmt.Printf("📄 %s - %s\n", tmpl.Name, tmpl.Title)
		fmt.Printf("   %s\n", tmpl.Description)
		if len(tmpl.Tags) > 0 {
			fmt.Printf("   Tags: %v\n", tmpl.Tags)
		}
		fmt.Printf("   Author: %s | Version: %s | Updated: %s\n",
			tmpl.Author, tmpl.Version, formatTimeAgo(tmpl.UpdatedAt))
		fmt.Println()
	}

	fmt.Printf("Found %d template(s) matching '%s'\n", len(templates), query)

	return nil
}

func runRegistryPull(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	destination := templateName + ".yaml"
	if len(args) > 1 {
		destination = args[1]
	}

	manager, err := createRegistryManager()
	if err != nil {
		return err
	}

	fmt.Printf("Downloading template '%s'...\n", templateName)

	err = manager.Pull(templateName, destination)
	if err != nil {
		return fmt.Errorf("failed to pull template: %w", err)
	}

	fmt.Printf("✅ Template saved to: %s\n", destination)
	fmt.Printf("\nYou can now use this template with:\n")
	fmt.Printf("  pctl create -t %s --key-name <your-key>\n", destination)

	return nil
}
