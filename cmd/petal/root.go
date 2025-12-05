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

	"github.com/scttfrdmn/petal/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "petal",
	Short: "🌸 Grow HPC clusters from seeds - Simplified AWS ParallelCluster deployment",
	Long: `petal is a CLI tool that simplifies AWS ParallelCluster deployment using
intuitive seed files (YAML). Plant a seed, watch your cluster bloom! 🌱

Bridges the gap between ParallelCluster's power and what you actually need:
a simple, repeatable way to deploy HPC clusters with software, users, and
data pre-configured.

For more information, visit: https://github.com/scttfrdmn/petal`,
	SilenceUsage:  true,
	SilenceErrors: false,
}

func init() {
	// Migrate from old pctl directory to petal on first run
	if err := config.MigrateFromPctl(); err != nil {
		fmt.Printf("⚠️  Warning: Config migration failed: %v\n", err)
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.petal/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
