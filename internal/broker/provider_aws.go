// Package broker — AWS IAM STS provider for ephemeral credentials.
// SPDX-License-Identifier: MIT
package broker

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AWSProvider provisions ephemeral credentials via AWS STS AssumeRole.
type AWSProvider struct {
	region    string // AWS region (e.g., "us-east-1")
	accountID string // AWS account ID
	// In a real implementation, this would use the AWS SDK
}

// NewAWSProvider creates a new AWS provider.
func NewAWSProvider(region, accountID string) *AWSProvider {
	return &AWSProvider{
		region:    region,
		accountID: accountID,
	}
}

func (a *AWSProvider) Name() string {
	return "aws-iam"
}

func (a *AWSProvider) ProvisionCredential(ctx context.Context, req *ProvisionRequest) (*Credential, error) {
	// In a real implementation, this would:
	// 1. Parse req.Scope.Resource (e.g., "s3://bucket/object", "arn:aws:s3:::bucket/object")
	// 2. Build an IAM policy with ONLY the specified permissions for that resource
	// 3. Call STS AssumeRole with the inline policy and SessionDuration=60s
	// 4. Return the temporary access key, secret key, and session token

	// Stub implementation for now
	credID := generateRequestID()
	now := time.Now()

	// Extract resource from scope
	resource := req.Scope.Resource
	var roleArn string
	if strings.HasPrefix(resource, "s3://") {
		roleArn = fmt.Sprintf("arn:aws:iam::%s:role/ans-s3-access", a.accountID)
	} else if strings.HasPrefix(resource, "arn:aws:") {
		// Resource is already an ARN
		roleArn = resource
	} else {
		return nil, fmt.Errorf("unsupported AWS resource format: %s", resource)
	}

	return &Credential{
		CredentialID: credID,
		AgentID:      req.AgentID,
		ProviderName: "aws-iam",
		Type:         "aws-sts",
		Secret:       fmt.Sprintf("ASIA%s", credID), // AWS temporary access key format
		Metadata: map[string]string{
			"role_arn":      roleArn,
			"session_name":  fmt.Sprintf("ans-%s-%s", req.AgentID, credID[:8]),
			"secret_key":    "[REDACTED]",
			"session_token": "[REDACTED]",
			"region":        a.region,
		},
		Scope:        req.Scope,
		IssuedAt:     now,
		ExpiresAt:    now.Add(time.Duration(req.TTLSeconds) * time.Second),
		Revoked:      false,
		RequestID:    req.RequestID,
		PreReceiptID: req.PreReceiptID,
	}, nil
}

func (a *AWSProvider) RevokeCredential(ctx context.Context, credentialID string) error {
	// AWS STS credentials cannot be explicitly revoked, but they expire automatically.
	// In a production system, you'd track the session name and potentially add it to a deny list.
	return nil
}

func (a *AWSProvider) ValidateScope(scope Scope) error {
	// Validate that scope.Resource is a valid AWS resource
	if scope.Resource == "" {
		return fmt.Errorf("aws resource cannot be empty")
	}
	if !strings.HasPrefix(scope.Resource, "s3://") &&
		!strings.HasPrefix(scope.Resource, "arn:aws:") {
		return fmt.Errorf("invalid AWS resource format: %s", scope.Resource)
	}
	// Validate permissions
	for _, perm := range scope.Permissions {
		if perm != "read" && perm != "write" && perm != "delete" && perm != "list" {
			return fmt.Errorf("invalid AWS permission: %s", perm)
		}
	}
	return nil
}
