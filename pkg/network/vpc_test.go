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

package network

import (
	"testing"
)

func TestNetworkResources(t *testing.T) {
	resources := &NetworkResources{
		VpcID:             "vpc-12345",
		PublicSubnetID:    "subnet-public-12345",
		PrivateSubnetID:   "subnet-private-12345",
		InternetGatewayID: "igw-12345",
		RouteTableID:      "rtb-12345",
		SecurityGroupID:   "sg-12345",
		Region:            "us-east-1",
		ClusterName:       "test-cluster",
		ManagedByPctl:     true,
	}

	if resources.VpcID != "vpc-12345" {
		t.Errorf("Expected VpcID vpc-12345, got %s", resources.VpcID)
	}

	if resources.PublicSubnetID != "subnet-public-12345" {
		t.Errorf("Expected PublicSubnetID subnet-public-12345, got %s", resources.PublicSubnetID)
	}

	if resources.PrivateSubnetID != "subnet-private-12345" {
		t.Errorf("Expected PrivateSubnetID subnet-private-12345, got %s", resources.PrivateSubnetID)
	}

	if resources.InternetGatewayID != "igw-12345" {
		t.Errorf("Expected InternetGatewayID igw-12345, got %s", resources.InternetGatewayID)
	}

	if resources.RouteTableID != "rtb-12345" {
		t.Errorf("Expected RouteTableID rtb-12345, got %s", resources.RouteTableID)
	}

	if resources.SecurityGroupID != "sg-12345" {
		t.Errorf("Expected SecurityGroupID sg-12345, got %s", resources.SecurityGroupID)
	}

	if resources.Region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", resources.Region)
	}

	if resources.ClusterName != "test-cluster" {
		t.Errorf("Expected cluster name test-cluster, got %s", resources.ClusterName)
	}

	if !resources.ManagedByPctl {
		t.Error("Expected ManagedByPctl to be true")
	}
}

func TestNetworkResourcesDefaults(t *testing.T) {
	resources := &NetworkResources{}

	if resources.VpcID != "" {
		t.Error("Expected empty VpcID")
	}

	if resources.ManagedByPctl {
		t.Error("Expected ManagedByPctl to be false by default")
	}
}

func TestNetworkResourcesPartial(t *testing.T) {
	// Test with only some fields populated
	resources := &NetworkResources{
		VpcID:         "vpc-12345",
		Region:        "us-west-2",
		ClusterName:   "partial-cluster",
		ManagedByPctl: false,
	}

	if resources.VpcID != "vpc-12345" {
		t.Errorf("Expected VpcID vpc-12345, got %s", resources.VpcID)
	}

	if resources.PublicSubnetID != "" {
		t.Error("Expected empty PublicSubnetID")
	}

	if resources.PrivateSubnetID != "" {
		t.Error("Expected empty PrivateSubnetID")
	}

	if resources.Region != "us-west-2" {
		t.Errorf("Expected region us-west-2, got %s", resources.Region)
	}

	if resources.ManagedByPctl {
		t.Error("Expected ManagedByPctl to be false")
	}
}

func TestNetworkResourcesUserProvided(t *testing.T) {
	// Test representing user-provided networking
	resources := &NetworkResources{
		VpcID:          "vpc-user-12345",
		PublicSubnetID: "subnet-user-12345",
		Region:         "eu-west-1",
		ClusterName:    "user-cluster",
		ManagedByPctl:  false, // User-provided, not managed by pctl
	}

	if resources.ManagedByPctl {
		t.Error("Expected ManagedByPctl to be false for user-provided networking")
	}

	if resources.VpcID != "vpc-user-12345" {
		t.Errorf("Expected VpcID vpc-user-12345, got %s", resources.VpcID)
	}

	// User might not provide all resources
	if resources.PrivateSubnetID != "" {
		t.Error("Expected empty PrivateSubnetID for user-provided networking")
	}
}

func TestNetworkResourcesCompleteness(t *testing.T) {
	tests := []struct {
		name      string
		resources *NetworkResources
		complete  bool
	}{
		{
			name: "complete_resources",
			resources: &NetworkResources{
				VpcID:             "vpc-12345",
				PublicSubnetID:    "subnet-pub-12345",
				PrivateSubnetID:   "subnet-priv-12345",
				InternetGatewayID: "igw-12345",
				RouteTableID:      "rtb-12345",
				SecurityGroupID:   "sg-12345",
			},
			complete: true,
		},
		{
			name: "missing_vpc",
			resources: &NetworkResources{
				PublicSubnetID:  "subnet-pub-12345",
				PrivateSubnetID: "subnet-priv-12345",
			},
			complete: false,
		},
		{
			name: "missing_subnets",
			resources: &NetworkResources{
				VpcID: "vpc-12345",
			},
			complete: false,
		},
		{
			name:      "empty_resources",
			resources: &NetworkResources{},
			complete:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if resources have required fields
			hasRequired := tt.resources.VpcID != "" &&
				tt.resources.PublicSubnetID != "" &&
				tt.resources.PrivateSubnetID != "" &&
				tt.resources.InternetGatewayID != "" &&
				tt.resources.RouteTableID != "" &&
				tt.resources.SecurityGroupID != ""

			if hasRequired != tt.complete {
				t.Errorf("Expected completeness %v, got %v", tt.complete, hasRequired)
			}
		})
	}
}

func TestNetworkResourcesValidation(t *testing.T) {
	tests := []struct {
		name      string
		resources *NetworkResources
		valid     bool
	}{
		{
			name: "valid_managed_resources",
			resources: &NetworkResources{
				VpcID:             "vpc-12345",
				PublicSubnetID:    "subnet-pub-12345",
				PrivateSubnetID:   "subnet-priv-12345",
				InternetGatewayID: "igw-12345",
				RouteTableID:      "rtb-12345",
				SecurityGroupID:   "sg-12345",
				Region:            "us-east-1",
				ClusterName:       "test-cluster",
				ManagedByPctl:     true,
			},
			valid: true,
		},
		{
			name: "valid_user_provided",
			resources: &NetworkResources{
				VpcID:          "vpc-12345",
				PublicSubnetID: "subnet-12345",
				Region:         "us-east-1",
				ClusterName:    "test-cluster",
				ManagedByPctl:  false,
			},
			valid: true,
		},
		{
			name: "missing_region",
			resources: &NetworkResources{
				VpcID:       "vpc-12345",
				ClusterName: "test-cluster",
			},
			valid: false,
		},
		{
			name: "missing_cluster_name",
			resources: &NetworkResources{
				VpcID:  "vpc-12345",
				Region: "us-east-1",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple validation: require region and cluster name
			isValid := tt.resources.Region != "" && tt.resources.ClusterName != ""

			if isValid != tt.valid {
				t.Errorf("Expected validity %v, got %v", tt.valid, isValid)
			}
		})
	}
}

func TestNetworkResourcesClone(t *testing.T) {
	original := &NetworkResources{
		VpcID:             "vpc-12345",
		PublicSubnetID:    "subnet-pub-12345",
		PrivateSubnetID:   "subnet-priv-12345",
		InternetGatewayID: "igw-12345",
		RouteTableID:      "rtb-12345",
		SecurityGroupID:   "sg-12345",
		Region:            "us-east-1",
		ClusterName:       "test-cluster",
		ManagedByPctl:     true,
	}

	// Create a copy
	clone := &NetworkResources{
		VpcID:             original.VpcID,
		PublicSubnetID:    original.PublicSubnetID,
		PrivateSubnetID:   original.PrivateSubnetID,
		InternetGatewayID: original.InternetGatewayID,
		RouteTableID:      original.RouteTableID,
		SecurityGroupID:   original.SecurityGroupID,
		Region:            original.Region,
		ClusterName:       original.ClusterName,
		ManagedByPctl:     original.ManagedByPctl,
	}

	// Verify all fields match
	if clone.VpcID != original.VpcID {
		t.Error("VpcID mismatch in clone")
	}

	if clone.PublicSubnetID != original.PublicSubnetID {
		t.Error("PublicSubnetID mismatch in clone")
	}

	if clone.PrivateSubnetID != original.PrivateSubnetID {
		t.Error("PrivateSubnetID mismatch in clone")
	}

	if clone.InternetGatewayID != original.InternetGatewayID {
		t.Error("InternetGatewayID mismatch in clone")
	}

	if clone.RouteTableID != original.RouteTableID {
		t.Error("RouteTableID mismatch in clone")
	}

	if clone.SecurityGroupID != original.SecurityGroupID {
		t.Error("SecurityGroupID mismatch in clone")
	}

	if clone.Region != original.Region {
		t.Error("Region mismatch in clone")
	}

	if clone.ClusterName != original.ClusterName {
		t.Error("ClusterName mismatch in clone")
	}

	if clone.ManagedByPctl != original.ManagedByPctl {
		t.Error("ManagedByPctl mismatch in clone")
	}

	// Modify clone and verify original is unaffected
	clone.VpcID = "vpc-different"
	if original.VpcID == "vpc-different" {
		t.Error("Modifying clone affected original")
	}
}
