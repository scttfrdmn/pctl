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

// Package network provides VPC and networking management for clusters.
package network

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// NetworkResources represents created network resources.
type NetworkResources struct {
	VpcID             string
	PublicSubnetID    string
	PrivateSubnetID   string
	InternetGatewayID string
	RouteTableID      string
	SecurityGroupID   string
	Region            string
	ClusterName       string
	ManagedByPctl     bool
}

// Manager manages VPC and networking resources.
type Manager struct {
	ec2Client *ec2.Client
	region    string
}

// NewManager creates a new network manager.
func NewManager(ctx context.Context, region string) (*Manager, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Manager{
		ec2Client: ec2.NewFromConfig(cfg),
		region:    region,
	}, nil
}

// CreateNetwork creates a complete VPC network for a cluster.
func (m *Manager) CreateNetwork(ctx context.Context, clusterName string) (*NetworkResources, error) {
	resources := &NetworkResources{
		Region:        m.region,
		ClusterName:   clusterName,
		ManagedByPctl: true,
	}

	// Create VPC
	vpcID, err := m.createVPC(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create VPC: %w", err)
	}
	resources.VpcID = vpcID

	// Create Internet Gateway
	igwID, err := m.createInternetGateway(ctx, clusterName, vpcID)
	if err != nil {
		m.cleanup(ctx, resources)
		return nil, fmt.Errorf("failed to create internet gateway: %w", err)
	}
	resources.InternetGatewayID = igwID

	// Create public subnet for head node
	publicSubnetID, err := m.createSubnet(ctx, clusterName, vpcID, "10.0.1.0/24", "public")
	if err != nil {
		m.cleanup(ctx, resources)
		return nil, fmt.Errorf("failed to create public subnet: %w", err)
	}
	resources.PublicSubnetID = publicSubnetID

	// Create private subnet for compute nodes
	privateSubnetID, err := m.createSubnet(ctx, clusterName, vpcID, "10.0.2.0/24", "private")
	if err != nil {
		m.cleanup(ctx, resources)
		return nil, fmt.Errorf("failed to create private subnet: %w", err)
	}
	resources.PrivateSubnetID = privateSubnetID

	// Create and configure route table
	routeTableID, err := m.createRouteTable(ctx, clusterName, vpcID, igwID, publicSubnetID)
	if err != nil {
		m.cleanup(ctx, resources)
		return nil, fmt.Errorf("failed to create route table: %w", err)
	}
	resources.RouteTableID = routeTableID

	// Create security group
	sgID, err := m.createSecurityGroup(ctx, clusterName, vpcID)
	if err != nil {
		m.cleanup(ctx, resources)
		return nil, fmt.Errorf("failed to create security group: %w", err)
	}
	resources.SecurityGroupID = sgID

	return resources, nil
}

func (m *Manager) createVPC(ctx context.Context, clusterName string) (string, error) {
	output, err := m.ec2Client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String("10.0.0.0/16"),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("pctl-%s", clusterName))},
					{Key: aws.String("ManagedBy"), Value: aws.String("pctl")},
					{Key: aws.String("ClusterName"), Value: aws.String(clusterName)},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	// Enable DNS hostnames
	_, err = m.ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId:              output.Vpc.VpcId,
		EnableDnsHostnames: &types.AttributeBooleanValue{Value: aws.Bool(true)},
	})
	if err != nil {
		return "", fmt.Errorf("failed to enable DNS hostnames: %w", err)
	}

	return *output.Vpc.VpcId, nil
}

func (m *Manager) createInternetGateway(ctx context.Context, clusterName, vpcID string) (string, error) {
	output, err := m.ec2Client.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInternetGateway,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("pctl-%s-igw", clusterName))},
					{Key: aws.String("ManagedBy"), Value: aws.String("pctl")},
					{Key: aws.String("ClusterName"), Value: aws.String(clusterName)},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	igwID := *output.InternetGateway.InternetGatewayId

	// Attach to VPC
	_, err = m.ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(igwID),
		VpcId:             aws.String(vpcID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to attach internet gateway: %w", err)
	}

	return igwID, nil
}

func (m *Manager) createSubnet(ctx context.Context, clusterName, vpcID, cidr, subnetType string) (string, error) {
	output, err := m.ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		VpcId:     aws.String(vpcID),
		CidrBlock: aws.String(cidr),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSubnet,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("pctl-%s-%s", clusterName, subnetType))},
					{Key: aws.String("ManagedBy"), Value: aws.String("pctl")},
					{Key: aws.String("ClusterName"), Value: aws.String(clusterName)},
					{Key: aws.String("Type"), Value: aws.String(subnetType)},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	subnetID := *output.Subnet.SubnetId

	// Enable auto-assign public IP for public subnet
	if subnetType == "public" {
		_, err = m.ec2Client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
			SubnetId:            aws.String(subnetID),
			MapPublicIpOnLaunch: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
		if err != nil {
			return "", fmt.Errorf("failed to enable public IP assignment: %w", err)
		}
	}

	return subnetID, nil
}

func (m *Manager) createRouteTable(ctx context.Context, clusterName, vpcID, igwID, publicSubnetID string) (string, error) {
	output, err := m.ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
		VpcId: aws.String(vpcID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeRouteTable,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("pctl-%s-public", clusterName))},
					{Key: aws.String("ManagedBy"), Value: aws.String("pctl")},
					{Key: aws.String("ClusterName"), Value: aws.String(clusterName)},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	routeTableID := *output.RouteTable.RouteTableId

	// Create route to internet gateway
	_, err = m.ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         aws.String(routeTableID),
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(igwID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create route: %w", err)
	}

	// Associate with public subnet
	_, err = m.ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(routeTableID),
		SubnetId:     aws.String(publicSubnetID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to associate route table: %w", err)
	}

	return routeTableID, nil
}

func (m *Manager) createSecurityGroup(ctx context.Context, clusterName, vpcID string) (string, error) {
	output, err := m.ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(fmt.Sprintf("pctl-%s", clusterName)),
		Description: aws.String(fmt.Sprintf("Security group for pctl cluster %s", clusterName)),
		VpcId:       aws.String(vpcID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroup,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("pctl-%s", clusterName))},
					{Key: aws.String("ManagedBy"), Value: aws.String("pctl")},
					{Key: aws.String("ClusterName"), Value: aws.String(clusterName)},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	sgID := *output.GroupId

	// Allow SSH from anywhere (you may want to restrict this)
	_, err = m.ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(sgID),
		IpPermissions: []types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(22),
				ToPort:     aws.Int32(22),
				IpRanges:   []types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
			},
			{
				IpProtocol: aws.String("-1"),
				UserIdGroupPairs: []types.UserIdGroupPair{
					{GroupId: aws.String(sgID)},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to authorize ingress rules: %w", err)
	}

	return sgID, nil
}

// DeleteNetwork deletes all network resources for a cluster.
func (m *Manager) DeleteNetwork(ctx context.Context, resources *NetworkResources) error {
	if !resources.ManagedByPctl {
		return nil // Don't delete user-provided networking
	}

	return m.cleanup(ctx, resources)
}

func (m *Manager) cleanup(ctx context.Context, resources *NetworkResources) error {
	var lastErr error

	// Delete security group
	if resources.SecurityGroupID != "" {
		_, err := m.ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(resources.SecurityGroupID),
		})
		if err != nil {
			lastErr = fmt.Errorf("failed to delete security group: %w", err)
		}
	}

	// Delete route table (association is deleted automatically)
	if resources.RouteTableID != "" {
		_, err := m.ec2Client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
			RouteTableId: aws.String(resources.RouteTableID),
		})
		if err != nil {
			lastErr = fmt.Errorf("failed to delete route table: %w", err)
		}
	}

	// Delete subnets
	if resources.PublicSubnetID != "" {
		_, err := m.ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
			SubnetId: aws.String(resources.PublicSubnetID),
		})
		if err != nil {
			lastErr = fmt.Errorf("failed to delete public subnet: %w", err)
		}
	}

	if resources.PrivateSubnetID != "" {
		_, err := m.ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
			SubnetId: aws.String(resources.PrivateSubnetID),
		})
		if err != nil {
			lastErr = fmt.Errorf("failed to delete private subnet: %w", err)
		}
	}

	// Detach and delete internet gateway
	if resources.InternetGatewayID != "" {
		if resources.VpcID != "" {
			_, err := m.ec2Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
				InternetGatewayId: aws.String(resources.InternetGatewayID),
				VpcId:             aws.String(resources.VpcID),
			})
			if err != nil {
				lastErr = fmt.Errorf("failed to detach internet gateway: %w", err)
			}
		}

		_, err := m.ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: aws.String(resources.InternetGatewayID),
		})
		if err != nil {
			lastErr = fmt.Errorf("failed to delete internet gateway: %w", err)
		}
	}

	// Delete VPC
	if resources.VpcID != "" {
		_, err := m.ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
			VpcId: aws.String(resources.VpcID),
		})
		if err != nil {
			lastErr = fmt.Errorf("failed to delete VPC: %w", err)
		}
	}

	return lastErr
}
