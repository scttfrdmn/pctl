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
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/scttfrdmn/pctl/pkg/ami"
	"github.com/scttfrdmn/pctl/pkg/template"
	"github.com/spf13/cobra"
)

var (
	amiTemplateFile string
	amiName         string
	amiDescription  string
	amiSubnetID     string
	amiKeyName      string
	amiTimeout      int
)

// amiCmd represents the ami command group
var amiCmd = &cobra.Command{
	Use:   "ami",
	Short: "Manage custom AMIs",
	Long: `Manage custom AMIs with pre-installed software.

Custom AMIs dramatically reduce cluster creation time by pre-installing
all software during the AMI build process (30-90 minutes once), allowing
clusters to boot in 2-3 minutes instead of waiting hours for software installation.`,
}

// buildAMICmd builds a custom AMI from a template
var buildAMICmd = &cobra.Command{
	Use:   "build",
	Short: "Build a custom AMI from a template",
	Long: `Build a custom AMI from a pctl template.

This command:
1. Launches a temporary EC2 instance
2. Installs all software from the template (Spack packages, Lmod, etc.)
3. Creates an AMI from the configured instance
4. Cleans up temporary resources

The process typically takes 30-90 minutes depending on the number of packages.

Example:
  pctl ami build -t bioinformatics.yaml --name bio-cluster-v1 --subnet-id subnet-xxx --key-name my-key`,
	RunE: runBuildAMI,
}

// listAMIsCmd lists all custom AMIs
var listAMIsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all custom AMIs",
	Long:  `List all pctl-managed custom AMIs in the current region.`,
	RunE:  runListAMIs,
}

// deleteAMICmd deletes a custom AMI
var deleteAMICmd = &cobra.Command{
	Use:   "delete [ami-id]",
	Short: "Delete a custom AMI",
	Long: `Delete a custom AMI and its associated snapshots.

Example:
  pctl ami delete ami-1234567890abcdef`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteAMI,
}

func init() {
	rootCmd.AddCommand(amiCmd)
	amiCmd.AddCommand(buildAMICmd)
	amiCmd.AddCommand(listAMIsCmd)
	amiCmd.AddCommand(deleteAMICmd)

	// Build AMI flags
	buildAMICmd.Flags().StringVarP(&amiTemplateFile, "template", "t", "", "template file (required)")
	buildAMICmd.Flags().StringVar(&amiName, "name", "", "AMI name (required)")
	buildAMICmd.Flags().StringVar(&amiDescription, "description", "", "AMI description")
	buildAMICmd.Flags().StringVar(&amiSubnetID, "subnet-id", "", "subnet ID for build instance (required)")
	buildAMICmd.Flags().StringVar(&amiKeyName, "key-name", "", "EC2 key pair name for SSH access (optional)")
	buildAMICmd.Flags().IntVar(&amiTimeout, "timeout", 90, "timeout in minutes for software installation")

	buildAMICmd.MarkFlagRequired("template")
	buildAMICmd.MarkFlagRequired("name")
	buildAMICmd.MarkFlagRequired("subnet-id")
}

func runBuildAMI(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load and validate template
	fmt.Printf("📄 Loading template: %s\n", amiTemplateFile)
	tmpl, err := template.Load(amiTemplateFile)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	if err := tmpl.Validate(); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	if len(tmpl.Software.SpackPackages) == 0 {
		return fmt.Errorf("template has no software packages - AMI building only makes sense for templates with software")
	}

	fmt.Printf("✅ Template validated\n\n")

	// Create AMI builder
	builder, err := ami.NewBuilder(ctx, tmpl.Cluster.Region)
	if err != nil {
		return fmt.Errorf("failed to create AMI builder: %w", err)
	}

	// Prepare build options
	opts := ami.DefaultBuildOptions()
	opts.Name = amiName
	opts.Description = amiDescription
	if opts.Description == "" {
		opts.Description = fmt.Sprintf("pctl AMI for %s template with %d packages",
			tmpl.Cluster.Name, len(tmpl.Software.SpackPackages))
	}
	opts.SubnetID = amiSubnetID
	opts.KeyName = amiKeyName
	opts.WaitTimeout = time.Duration(amiTimeout) * time.Minute

	// Build AMI
	metadata, err := builder.BuildAMI(ctx, tmpl, opts)
	if err != nil {
		return fmt.Errorf("AMI build failed: %w", err)
	}

	fmt.Printf("✅ AMI build successful!\n\n")
	fmt.Printf("AMI Details:\n")
	fmt.Printf("  ID:          %s\n", metadata.AMIID)
	fmt.Printf("  Name:        %s\n", metadata.Name)
	fmt.Printf("  Region:      %s\n", metadata.Region)
	fmt.Printf("  Template:    %s\n", metadata.TemplateName)
	fmt.Printf("  Packages:    %d\n\n", len(metadata.SpackPackages))

	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. Test the AMI:\n")
	fmt.Printf("     pctl create -t %s --key-name <key> --custom-ami %s\n\n", amiTemplateFile, metadata.AMIID)
	fmt.Printf("  2. Share the AMI with other AWS accounts if needed:\n")
	fmt.Printf("     aws ec2 modify-image-attribute --image-id %s --launch-permission \"Add=[{UserId=123456789012}]\"\n\n", metadata.AMIID)

	return nil
}

func runListAMIs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Determine region (use from config or default)
	region := "us-east-1" // TODO: Get from config

	manager, err := ami.NewManager(ctx, region)
	if err != nil {
		return fmt.Errorf("failed to create AMI manager: %w", err)
	}

	fmt.Printf("Fetching AMIs from region: %s\n\n", region)

	amis, err := manager.ListAMIs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list AMIs: %w", err)
	}

	if len(amis) == 0 {
		fmt.Println("No custom AMIs found.")
		fmt.Println("\nBuild your first AMI with:")
		fmt.Println("  pctl ami build -t template.yaml --name my-ami --subnet-id subnet-xxx --key-name my-key")
		return nil
	}

	// Print AMIs in a table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "AMI ID\tNAME\tTEMPLATE\tREGION\n")
	fmt.Fprintf(w, "──────\t────\t────────\t──────\n")

	for _, amiMeta := range amis {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			amiMeta.AMIID, amiMeta.Name, amiMeta.TemplateName, amiMeta.Region)
	}

	w.Flush()

	fmt.Printf("\nTotal: %d AMI(s)\n\n", len(amis))
	fmt.Printf("Use 'pctl create -t template.yaml --custom-ami <ami-id>' to create a cluster with a custom AMI.\n")

	return nil
}

func runDeleteAMI(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	amiID := args[0]

	// Determine region
	region := "us-east-1" // TODO: Get from config

	manager, err := ami.NewManager(ctx, region)
	if err != nil {
		return fmt.Errorf("failed to create AMI manager: %w", err)
	}

	// Get AMI details before deletion
	metadata, err := manager.GetAMI(ctx, amiID)
	if err != nil {
		return fmt.Errorf("failed to get AMI details: %w", err)
	}

	// Confirm deletion
	fmt.Printf("⚠️  About to delete AMI:\n")
	fmt.Printf("  ID:       %s\n", metadata.AMIID)
	fmt.Printf("  Name:     %s\n", metadata.Name)
	fmt.Printf("  Template: %s\n\n", metadata.TemplateName)
	fmt.Printf("This will also delete associated snapshots.\n")
	fmt.Printf("Type 'yes' to confirm deletion: ")

	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "yes" {
		fmt.Println("\n❌ Deletion cancelled.")
		return nil
	}

	fmt.Printf("\n🗑️  Deleting AMI %s...\n", amiID)

	err = manager.DeleteAMI(ctx, amiID)
	if err != nil {
		return fmt.Errorf("failed to delete AMI: %w", err)
	}

	fmt.Printf("✅ AMI deleted successfully\n")

	return nil
}
