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

// Package bootstrap provides S3 bootstrap script management for clusters.
package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// S3Manager manages bootstrap script uploads to S3.
type S3Manager struct {
	s3Client  *s3.Client
	stsClient *sts.Client
	region    string
}

// NewS3Manager creates a new S3 manager.
func NewS3Manager(ctx context.Context, region string) (*S3Manager, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &S3Manager{
		s3Client:  s3.NewFromConfig(cfg),
		stsClient: sts.NewFromConfig(cfg),
		region:    region,
	}, nil
}

// UploadBootstrapScript uploads a bootstrap script to S3 and returns the S3 URI.
func (m *S3Manager) UploadBootstrapScript(ctx context.Context, clusterName, scriptContent string) (string, error) {
	// Get AWS account ID for unique bucket naming
	accountID, err := m.getAccountID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get AWS account ID: %w", err)
	}

	// Create bucket name: pctl-bootstrap-{region}-{account-id}
	bucketName := fmt.Sprintf("pctl-bootstrap-%s-%s", m.region, accountID)

	// Ensure bucket exists
	if err := m.ensureBucketExists(ctx, bucketName); err != nil {
		return "", fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	// Upload script
	objectKey := fmt.Sprintf("%s/install-software.sh", clusterName)
	if err := m.uploadObject(ctx, bucketName, objectKey, scriptContent); err != nil {
		return "", fmt.Errorf("failed to upload bootstrap script: %w", err)
	}

	// Return S3 URI
	s3URI := fmt.Sprintf("s3://%s/%s", bucketName, objectKey)
	return s3URI, nil
}

// DeleteBootstrapScript deletes a bootstrap script from S3.
func (m *S3Manager) DeleteBootstrapScript(ctx context.Context, s3URI string) error {
	// Parse S3 URI: s3://bucket/key
	if !strings.HasPrefix(s3URI, "s3://") {
		return fmt.Errorf("invalid S3 URI: %s", s3URI)
	}

	parts := strings.SplitN(strings.TrimPrefix(s3URI, "s3://"), "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid S3 URI format: %s", s3URI)
	}

	bucketName := parts[0]
	objectKey := parts[1]

	_, err := m.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

func (m *S3Manager) getAccountID(ctx context.Context) (string, error) {
	result, err := m.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return *result.Account, nil
}

func (m *S3Manager) ensureBucketExists(ctx context.Context, bucketName string) error {
	// Check if bucket exists
	_, err := m.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})

	if err == nil {
		// Bucket exists
		return nil
	}

	// Bucket doesn't exist, create it
	createInput := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	// For regions other than us-east-1, need to specify LocationConstraint
	if m.region != "us-east-1" {
		createInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(m.region),
		}
	}

	_, err = m.s3Client.CreateBucket(ctx, createInput)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	// Enable versioning (optional but recommended)
	_, err = m.s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucketName),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	})
	if err != nil {
		// Non-fatal - continue even if versioning fails
		fmt.Printf("Warning: failed to enable versioning on bucket %s: %v\n", bucketName, err)
	}

	return nil
}

func (m *S3Manager) uploadObject(ctx context.Context, bucketName, objectKey, content string) error {
	_, err := m.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader([]byte(content)),
		ContentType: aws.String("text/x-shellscript"),
	})

	return err
}
