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

	"github.com/scttfrdmn/petal/pkg/template"
	"github.com/spf13/cobra"
)

var (
	validateTemplate string
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a cluster template",
	Long: `Validate a pctl cluster template file.

This command checks the template for:
- Syntax errors (YAML parsing)
- Schema validation (required fields, types)
- Semantic validation (valid regions, instance types, naming conventions)
- Best practices (UID/GID ranges, resource limits)

The command returns exit code 0 if the template is valid, non-zero otherwise.`,
	Example: `  # Validate a template
  pctl validate -t my-cluster.yaml

  # Validate with verbose output
  pctl validate -t my-cluster.yaml --verbose`,
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().StringVarP(&validateTemplate, "template", "t", "", "path to template file (required)")
	validateCmd.MarkFlagRequired("template")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	if verbose {
		fmt.Printf("Validating template: %s\n", validateTemplate)
	}

	// Load template
	tmpl, err := template.Load(validateTemplate)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	if verbose {
		fmt.Printf("Template loaded successfully\n")
		fmt.Printf("  Cluster: %s\n", tmpl.Cluster.Name)
		fmt.Printf("  Region: %s\n", tmpl.Cluster.Region)
		fmt.Printf("  Head Node: %s\n", tmpl.Compute.HeadNode)
		fmt.Printf("  Queues: %d\n", len(tmpl.Compute.Queues))
		if len(tmpl.Software.SpackPackages) > 0 {
			fmt.Printf("  Software Packages: %d\n", len(tmpl.Software.SpackPackages))
		}
		if len(tmpl.Users) > 0 {
			fmt.Printf("  Users: %d\n", len(tmpl.Users))
		}
		if len(tmpl.Data.S3Mounts) > 0 {
			fmt.Printf("  S3 Mounts: %d\n", len(tmpl.Data.S3Mounts))
		}
		fmt.Println()
	}

	// Validate template
	if err := tmpl.Validate(); err != nil {
		fmt.Printf("❌ Template validation failed:\n\n%v\n", err)
		return fmt.Errorf("validation failed")
	}

	fmt.Printf("✅ Template is valid!\n")
	return nil
}
