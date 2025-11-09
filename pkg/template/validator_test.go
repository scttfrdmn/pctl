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
	"strings"
	"testing"
)

func TestValidatorClusterValidation(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    Template
		wantErr []string
	}{
		{
			name: "valid cluster config",
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
			wantErr: nil,
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
			wantErr: []string{"cluster.name is required"},
		},
		{
			name: "invalid cluster name",
			tmpl: Template{
				Cluster: ClusterConfig{
					Name:   "123-invalid",
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
			wantErr: []string{"cluster.name must start with a letter"},
		},
		{
			name: "invalid region",
			tmpl: Template{
				Cluster: ClusterConfig{
					Name:   "test-cluster",
					Region: "invalid-region",
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
			wantErr: []string{"cluster.region 'invalid-region' is not a valid AWS region"},
		},
	}

	validator := NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateTemplate(&tt.tmpl)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateTemplate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateTemplate() expected error containing %v, got nil", tt.wantErr)
					return
				}
				errMsg := err.Error()
				for _, want := range tt.wantErr {
					if !strings.Contains(errMsg, want) {
						t.Errorf("ValidateTemplate() error = %v, want error containing %v", err, want)
					}
				}
			}
		})
	}
}

func TestValidatorComputeValidation(t *testing.T) {
	tests := []struct {
		name    string
		compute ComputeConfig
		wantErr []string
	}{
		{
			name: "valid compute config",
			compute: ComputeConfig{
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
			wantErr: nil,
		},
		{
			name: "invalid instance type",
			compute: ComputeConfig{
				HeadNode: "invalid-type",
				Queues: []Queue{
					{
						Name:          "compute",
						InstanceTypes: []string{"c5.xlarge"},
						MinCount:      0,
						MaxCount:      10,
					},
				},
			},
			wantErr: []string{"compute.head_node 'invalid-type' is not a valid instance type"},
		},
		{
			name: "max less than min",
			compute: ComputeConfig{
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
			wantErr: []string{"max_count (5) must be >= min_count (10)"},
		},
		{
			name: "duplicate queue names",
			compute: ComputeConfig{
				HeadNode: "t3.medium",
				Queues: []Queue{
					{
						Name:          "compute",
						InstanceTypes: []string{"c5.xlarge"},
						MinCount:      0,
						MaxCount:      10,
					},
					{
						Name:          "compute",
						InstanceTypes: []string{"c5.2xlarge"},
						MinCount:      0,
						MaxCount:      5,
					},
				},
			},
			wantErr: []string{"name 'compute' is duplicate"},
		},
	}

	validator := NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := Template{
				Cluster: ClusterConfig{
					Name:   "test-cluster",
					Region: "us-east-1",
				},
				Compute: tt.compute,
			}
			err := validator.ValidateTemplate(&tmpl)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateTemplate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateTemplate() expected error containing %v, got nil", tt.wantErr)
					return
				}
				errMsg := err.Error()
				for _, want := range tt.wantErr {
					if !strings.Contains(errMsg, want) {
						t.Errorf("ValidateTemplate() error = %v, want error containing %v", err, want)
					}
				}
			}
		})
	}
}

func TestValidatorUsersValidation(t *testing.T) {
	tests := []struct {
		name    string
		users   []User
		wantErr []string
	}{
		{
			name: "valid users",
			users: []User{
				{Name: "user1", UID: 5001, GID: 5001},
				{Name: "user2", UID: 5002, GID: 5002},
			},
			wantErr: nil,
		},
		{
			name: "duplicate user names",
			users: []User{
				{Name: "user1", UID: 5001, GID: 5001},
				{Name: "user1", UID: 5002, GID: 5002},
			},
			wantErr: []string{"name 'user1' is duplicate"},
		},
		{
			name: "duplicate UIDs",
			users: []User{
				{Name: "user1", UID: 5001, GID: 5001},
				{Name: "user2", UID: 5001, GID: 5002},
			},
			wantErr: []string{"uid 5001 is duplicate"},
		},
		{
			name: "invalid user name",
			users: []User{
				{Name: "User1", UID: 5001, GID: 5001},
			},
			wantErr: []string{"name 'User1' must start with lowercase letter"},
		},
		{
			name: "system range UID",
			users: []User{
				{Name: "user1", UID: 500, GID: 5001},
			},
			wantErr: []string{"uid 500 is in system range"},
		},
	}

	validator := NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := Template{
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
				Users: tt.users,
			}
			err := validator.ValidateTemplate(&tmpl)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateTemplate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateTemplate() expected error containing %v, got nil", tt.wantErr)
					return
				}
				errMsg := err.Error()
				for _, want := range tt.wantErr {
					if !strings.Contains(errMsg, want) {
						t.Errorf("ValidateTemplate() error = %v, want error containing %v", err, want)
					}
				}
			}
		})
	}
}

func TestValidatorS3Validation(t *testing.T) {
	tests := []struct {
		name      string
		s3Mounts  []S3Mount
		wantErr   []string
		wantNoErr bool
	}{
		{
			name: "valid S3 mounts",
			s3Mounts: []S3Mount{
				{Bucket: "my-bucket", MountPoint: "/mnt/data"},
				{Bucket: "another-bucket", MountPoint: "/shared/files"},
			},
			wantNoErr: true,
		},
		{
			name: "invalid bucket name",
			s3Mounts: []S3Mount{
				{Bucket: "INVALID_BUCKET", MountPoint: "/mnt/data"},
			},
			wantErr: []string{"bucket 'INVALID_BUCKET' is not a valid S3 bucket name"},
		},
		{
			name: "relative mount point",
			s3Mounts: []S3Mount{
				{Bucket: "my-bucket", MountPoint: "relative/path"},
			},
			wantErr: []string{"mount_point 'relative/path' must be an absolute path"},
		},
		{
			name: "duplicate mount points",
			s3Mounts: []S3Mount{
				{Bucket: "bucket1", MountPoint: "/mnt/data"},
				{Bucket: "bucket2", MountPoint: "/mnt/data"},
			},
			wantErr: []string{"mount_point '/mnt/data' is duplicate"},
		},
	}

	validator := NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := Template{
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
				Data: DataConfig{
					S3Mounts: tt.s3Mounts,
				},
			}
			err := validator.ValidateTemplate(&tmpl)
			if tt.wantNoErr {
				if err != nil {
					t.Errorf("ValidateTemplate() unexpected error = %v", err)
				}
			} else if len(tt.wantErr) > 0 {
				if err == nil {
					t.Errorf("ValidateTemplate() expected error containing %v, got nil", tt.wantErr)
					return
				}
				errMsg := err.Error()
				for _, want := range tt.wantErr {
					if !strings.Contains(errMsg, want) {
						t.Errorf("ValidateTemplate() error = %v, want error containing %v", err, want)
					}
				}
			}
		})
	}
}

func TestValidationErrorMultiple(t *testing.T) {
	tmpl := Template{
		Cluster: ClusterConfig{
			Name:   "", // missing
			Region: "invalid-region",
		},
		Compute: ComputeConfig{
			HeadNode: "",        // missing
			Queues:   []Queue{}, // empty
		},
	}

	validator := NewValidator()
	err := validator.ValidateTemplate(&tmpl)
	if err == nil {
		t.Fatal("ValidateTemplate() expected error, got nil")
	}

	// Should have multiple errors
	errMsg := err.Error()
	expectedErrors := []string{
		"cluster.name is required",
		"cluster.region",
		"compute.head_node is required",
		"compute.queues must have at least one queue",
	}

	for _, want := range expectedErrors {
		if !strings.Contains(errMsg, want) {
			t.Errorf("ValidateTemplate() error = %v, want error containing %v", err, want)
		}
	}
}
