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

package ami

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Manager provides AMI lifecycle management operations.
type Manager struct {
	builder *Builder
}

// NewManager creates a new AMI manager.
func NewManager(ctx context.Context, region string) (*Manager, error) {
	builder, err := NewBuilder(ctx, region)
	if err != nil {
		return nil, err
	}

	return &Manager{
		builder: builder,
	}, nil
}

// ListAMIs lists all pctl-managed AMIs in the region.
func (m *Manager) ListAMIs(ctx context.Context) ([]*AMIMetadata, error) {
	result, err := m.builder.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners: []string{"self"},
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:ManagedBy"),
				Values: []string{"pctl"},
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list AMIs: %w", err)
	}

	var amis []*AMIMetadata
	for _, img := range result.Images {
		metadata := &AMIMetadata{
			AMIID:       *img.ImageId,
			Name:        *img.Name,
			Description: aws.ToString(img.Description),
			Region:      m.builder.region,
			Tags:        make(map[string]string),
		}

		// Extract tags
		for _, tag := range img.Tags {
			if tag.Key != nil && tag.Value != nil {
				metadata.Tags[*tag.Key] = *tag.Value

				// Extract special tags
				switch *tag.Key {
				case "TemplateName":
					metadata.TemplateName = *tag.Value
				}
			}
		}

		amis = append(amis, metadata)
	}

	return amis, nil
}

// GetAMI retrieves metadata for a specific AMI.
func (m *Manager) GetAMI(ctx context.Context, amiID string) (*AMIMetadata, error) {
	result, err := m.builder.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get AMI: %w", err)
	}

	if len(result.Images) == 0 {
		return nil, fmt.Errorf("AMI %s not found", amiID)
	}

	img := result.Images[0]
	metadata := &AMIMetadata{
		AMIID:       *img.ImageId,
		Name:        *img.Name,
		Description: aws.ToString(img.Description),
		Region:      m.builder.region,
		Tags:        make(map[string]string),
	}

	// Extract tags
	for _, tag := range img.Tags {
		if tag.Key != nil && tag.Value != nil {
			metadata.Tags[*tag.Key] = *tag.Value

			switch *tag.Key {
			case "TemplateName":
				metadata.TemplateName = *tag.Value
			}
		}
	}

	return metadata, nil
}

// DeleteAMI deletes an AMI and its associated snapshots.
func (m *Manager) DeleteAMI(ctx context.Context, amiID string) error {
	// Get AMI details to find snapshots
	result, err := m.builder.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	})

	if err != nil {
		return fmt.Errorf("failed to describe AMI: %w", err)
	}

	if len(result.Images) == 0 {
		return fmt.Errorf("AMI %s not found", amiID)
	}

	// Collect snapshot IDs
	var snapshotIDs []string
	for _, bdm := range result.Images[0].BlockDeviceMappings {
		if bdm.Ebs != nil && bdm.Ebs.SnapshotId != nil {
			snapshotIDs = append(snapshotIDs, *bdm.Ebs.SnapshotId)
		}
	}

	// Deregister AMI
	_, err = m.builder.ec2Client.DeregisterImage(ctx, &ec2.DeregisterImageInput{
		ImageId: aws.String(amiID),
	})

	if err != nil {
		return fmt.Errorf("failed to deregister AMI: %w", err)
	}

	// Delete snapshots
	for _, snapshotID := range snapshotIDs {
		_, err := m.builder.ec2Client.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{
			SnapshotId: aws.String(snapshotID),
		})

		if err != nil {
			// Log error but continue with other snapshots
			fmt.Printf("Warning: failed to delete snapshot %s: %v\n", snapshotID, err)
		}
	}

	return nil
}

// FindAMIByTemplate finds an AMI built from a specific template.
func (m *Manager) FindAMIByTemplate(ctx context.Context, templateName string) (*AMIMetadata, error) {
	result, err := m.builder.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners: []string{"self"},
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:ManagedBy"),
				Values: []string{"pctl"},
			},
			{
				Name:   aws.String("tag:TemplateName"),
				Values: []string{templateName},
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find AMI: %w", err)
	}

	if len(result.Images) == 0 {
		return nil, fmt.Errorf("no AMI found for template %s", templateName)
	}

	// Return the most recent AMI if multiple exist
	latest := result.Images[0]
	for _, img := range result.Images[1:] {
		if img.CreationDate != nil && latest.CreationDate != nil {
			if *img.CreationDate > *latest.CreationDate {
				latest = img
			}
		}
	}

	metadata := &AMIMetadata{
		AMIID:        *latest.ImageId,
		Name:         *latest.Name,
		Description:  aws.ToString(latest.Description),
		Region:       m.builder.region,
		TemplateName: templateName,
		Tags:         make(map[string]string),
	}

	// Extract tags
	for _, tag := range latest.Tags {
		if tag.Key != nil && tag.Value != nil {
			metadata.Tags[*tag.Key] = *tag.Value
		}
	}

	return metadata, nil
}
