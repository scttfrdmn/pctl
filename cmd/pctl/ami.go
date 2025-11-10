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

	"github.com/schollz/progressbar/v3"
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
	amiSkipCleanup  bool
	amiDetach       bool
	amiWatch        bool
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

// statusBuildCmd checks the status of an AMI build
var statusBuildCmd = &cobra.Command{
	Use:   "status [build-id]",
	Short: "Check the status of an AMI build",
	Long: `Check the status of an AMI build by its build ID.

This command shows:
- Build progress and current status
- Elapsed time
- AMI ID (if complete)
- Error message (if failed)

Example:
  pctl ami status 550e8400-e29b-41d4-a716-446655440000`,
	Args: cobra.ExactArgs(1),
	RunE: runStatusBuild,
}

// listBuildsCmd lists all AMI builds
var listBuildsCmd = &cobra.Command{
	Use:   "list-builds",
	Short: "List all AMI builds",
	Long: `List all AMI builds (in-progress, completed, and failed).

Example:
  pctl ami list-builds`,
	RunE: runListBuilds,
}

func init() {
	rootCmd.AddCommand(amiCmd)
	amiCmd.AddCommand(buildAMICmd)
	amiCmd.AddCommand(listAMIsCmd)
	amiCmd.AddCommand(deleteAMICmd)
	amiCmd.AddCommand(statusBuildCmd)
	amiCmd.AddCommand(listBuildsCmd)

	// Build AMI flags
	buildAMICmd.Flags().StringVarP(&amiTemplateFile, "template", "t", "", "template file (required)")
	buildAMICmd.Flags().StringVar(&amiName, "name", "", "AMI name (required)")
	buildAMICmd.Flags().StringVar(&amiDescription, "description", "", "AMI description")
	buildAMICmd.Flags().StringVar(&amiSubnetID, "subnet-id", "", "subnet ID for build instance (required)")
	buildAMICmd.Flags().StringVar(&amiKeyName, "key-name", "", "EC2 key pair name for SSH access (optional)")
	buildAMICmd.Flags().IntVar(&amiTimeout, "timeout", 90, "timeout in minutes for software installation")
	buildAMICmd.Flags().BoolVar(&amiSkipCleanup, "no-cleanup", false, "skip automatic cleanup before AMI creation (not recommended)")
	buildAMICmd.Flags().BoolVar(&amiDetach, "detach", false, "start build and exit immediately (build continues in AWS)")

	buildAMICmd.MarkFlagRequired("template")
	buildAMICmd.MarkFlagRequired("name")
	buildAMICmd.MarkFlagRequired("subnet-id")

	// Status command flags
	statusBuildCmd.Flags().BoolVarP(&amiWatch, "watch", "w", false, "continuously watch build progress until complete")
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
	opts.SkipCleanup = amiSkipCleanup
	opts.Detach = amiDetach

	// Show cleanup status
	if amiSkipCleanup {
		fmt.Printf("⚠️  Cleanup disabled - AMI will be larger and may contain sensitive data\n\n")
	} else {
		fmt.Printf("✅ Cleanup enabled - AMI will be optimized for size and security\n\n")
	}

	// Show detach status
	if amiDetach {
		fmt.Printf("🚀 Detach mode enabled - build will start and CLI will exit\n\n")
	}

	// Build AMI
	metadata, err := builder.BuildAMI(ctx, tmpl, opts)
	if err != nil {
		return fmt.Errorf("AMI build failed: %w", err)
	}

	// If detached, the build details were already printed by BuildAMI
	if amiDetach {
		return nil
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

func runStatusBuild(cmd *cobra.Command, args []string) error {
	buildID := args[0]

	stateManager, err := ami.NewStateManager()
	if err != nil {
		return fmt.Errorf("failed to create state manager: %w", err)
	}

	state, err := stateManager.LoadState(buildID)
	if err != nil {
		return fmt.Errorf("failed to load build state: %w", err)
	}

	// Display build status
	fmt.Printf("Build Status\n")
	fmt.Printf("============\n\n")
	fmt.Printf("Build ID:     %s\n", state.BuildID)
	fmt.Printf("Status:       %s\n", formatStatus(state.Status))
	fmt.Printf("Progress:     %d%%\n", state.Progress)
	if state.ProgressMessage != "" {
		fmt.Printf("Message:      %s\n", state.ProgressMessage)
	}
	fmt.Printf("AMI Name:     %s\n", state.AMIName)
	fmt.Printf("Template:     %s\n", state.TemplateName)
	fmt.Printf("Region:       %s\n", state.Region)
	fmt.Printf("Packages:     %d\n", state.PackageCount)
	fmt.Printf("Started:      %s\n", state.StartTime.Format(time.RFC3339))

	// Calculate elapsed time
	var elapsed time.Duration
	if state.EndTime != nil {
		elapsed = state.EndTime.Sub(state.StartTime)
		fmt.Printf("Completed:    %s\n", state.EndTime.Format(time.RFC3339))
		fmt.Printf("Duration:     %s\n", formatDuration(elapsed))
	} else {
		elapsed = time.Since(state.StartTime)
		fmt.Printf("Elapsed:      %s\n", formatDuration(elapsed))
	}

	// Show AMI ID if complete
	if state.Status == ami.BuildStatusComplete && state.AMIID != "" {
		fmt.Printf("\n✅ AMI ID:   %s\n", state.AMIID)
		fmt.Printf("\nYou can now use this AMI with:\n")
		fmt.Printf("  pctl create -t template.yaml --key-name <key> --custom-ami %s\n", state.AMIID)
	}

	// Show error if failed
	if state.Status == ami.BuildStatusFailed && state.ErrorMessage != "" {
		fmt.Printf("\n❌ Error:    %s\n", state.ErrorMessage)
	}

	// Watch mode
	if amiWatch && state.Status != ami.BuildStatusComplete && state.Status != ami.BuildStatusFailed {
		fmt.Printf("\n⏳ Watching build progress (press Ctrl+C to exit)...\n\n")
		return watchBuild(stateManager, buildID)
	}

	return nil
}

func runListBuilds(cmd *cobra.Command, args []string) error {
	stateManager, err := ami.NewStateManager()
	if err != nil {
		return fmt.Errorf("failed to create state manager: %w", err)
	}

	states, err := stateManager.ListStates()
	if err != nil {
		return fmt.Errorf("failed to list builds: %w", err)
	}

	if len(states) == 0 {
		fmt.Println("No builds found.")
		fmt.Println("\nStart a build with:")
		fmt.Println("  pctl ami build -t template.yaml --name my-ami --subnet-id subnet-xxx --key-name my-key")
		return nil
	}

	// Print builds in a table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "BUILD ID\tSTATUS\tPROGRESS\tAMI NAME\tSTARTED\tDURATION\n")
	fmt.Fprintf(w, "────────\t──────\t────────\t────────\t───────\t────────\n")

	for _, state := range states {
		var duration string
		if state.EndTime != nil {
			duration = formatDuration(state.EndTime.Sub(state.StartTime))
		} else {
			duration = formatDuration(time.Since(state.StartTime))
		}

		// Truncate build ID for display
		shortID := state.BuildID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}

		fmt.Fprintf(w, "%s\t%s\t%d%%\t%s\t%s\t%s\n",
			shortID,
			formatStatus(state.Status),
			state.Progress,
			state.AMIName,
			formatRelativeTime(state.StartTime),
			duration)
	}

	w.Flush()

	fmt.Printf("\nTotal: %d build(s)\n\n", len(states))
	fmt.Printf("Use 'pctl ami status <build-id>' to check build details.\n")

	return nil
}

func formatStatus(status ami.BuildStatus) string {
	switch status {
	case ami.BuildStatusLaunching:
		return "🚀 launching"
	case ami.BuildStatusInstalling:
		return "📦 installing"
	case ami.BuildStatusCreating:
		return "🏗️  creating"
	case ami.BuildStatusComplete:
		return "✅ complete"
	case ami.BuildStatusFailed:
		return "❌ failed"
	default:
		return string(status)
	}
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes())
	days := hours / 24

	if days > 0 {
		return fmt.Sprintf("%dd ago", days)
	} else if hours > 0 {
		return fmt.Sprintf("%dh ago", hours)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm ago", minutes)
	}
	return "just now"
}

func watchBuild(stateManager *ami.StateManager, buildID string) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	lastProgress := -1

	// Create progress bar
	bar := progressbar.NewOptions(100,
		progressbar.OptionSetDescription("📦 Building AMI"),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
	)

	//nolint:staticcheck // Intentional infinite loop for polling, exits on build completion
	for {
		select {
		case <-ticker.C:
			state, err := stateManager.LoadState(buildID)
			if err != nil {
				return fmt.Errorf("failed to load build state: %w", err)
			}

			// Update progress bar if changed
			if state.Progress != lastProgress {
				delta := state.Progress - lastProgress
				if delta > 0 {
					bar.Add(delta)
				}

				// Calculate time estimate
				elapsed := time.Since(state.StartTime)
				if state.Progress > 0 && state.Progress < 100 {
					totalEstimate := time.Duration(float64(elapsed) / float64(state.Progress) * 100)
					remaining := totalEstimate - elapsed

					// Update bar description with estimate
					if remaining > 0 {
						bar.Describe(fmt.Sprintf("📦 Building AMI (~%dm remaining)", int(remaining.Minutes())))
					}
				}

				lastProgress = state.Progress
			}

			// Check if complete
			if state.Status == ami.BuildStatusComplete {
				bar.Add(100 - lastProgress) // Ensure bar is complete
				fmt.Printf("\n✅ Build complete! AMI ID: %s\n", state.AMIID)
				return nil
			}

			// Check if failed
			if state.Status == ami.BuildStatusFailed {
				fmt.Printf("\n❌ Build failed: %s\n", state.ErrorMessage)
				return fmt.Errorf("build failed")
			}
		}
	}
}
