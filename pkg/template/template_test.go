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

package template

import (
	"testing"
)

func TestTemplateValidate(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    Template
		wantErr bool
	}{
		{
			name: "valid template",
			tmpl: Template{
				Cluster: ClusterConfig{
					Name:   "test-cluster",
					Region: "us-east-1",
				},
				Compute: ComputeConfig{
					HeadNode: "t3.medium",
					Queues: []Queue{
						{
							Name:          "compute",
							InstanceTypes: []string{"c5.xlarge"},
							MinCount:      0,
							MaxCount:      10,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing cluster name",
			tmpl: Template{
				Cluster: ClusterConfig{
					Region: "us-east-1",
				},
				Compute: ComputeConfig{
					HeadNode: "t3.medium",
					Queues: []Queue{
						{
							Name:          "compute",
							InstanceTypes: []string{"c5.xlarge"},
							MinCount:      0,
							MaxCount:      10,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid queue config",
			tmpl: Template{
				Cluster: ClusterConfig{
					Name:   "test-cluster",
					Region: "us-east-1",
				},
				Compute: ComputeConfig{
					HeadNode: "t3.medium",
					Queues: []Queue{
						{
							Name:          "compute",
							InstanceTypes: []string{"c5.xlarge"},
							MinCount:      10,
							MaxCount:      5,
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tmpl.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Template.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
