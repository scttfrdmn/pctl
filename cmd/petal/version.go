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
	"encoding/json"
	"fmt"

	"github.com/scttfrdmn/petal/internal/version"
	"github.com/spf13/cobra"
)

var (
	versionOutput string
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print detailed version information including build time and git commit.",
	RunE:  runVersion,
}

func init() {
	versionCmd.Flags().StringVarP(&versionOutput, "output", "o", "text", "output format (text|json)")
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	info := version.Get()

	switch versionOutput {
	case "json":
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal version info: %w", err)
		}
		fmt.Println(string(data))
	case "text":
		fmt.Println(info.String())
	default:
		return fmt.Errorf("invalid output format: %s (must be text or json)", versionOutput)
	}

	return nil
}
