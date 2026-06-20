package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type STSClient interface {
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

type AWSProvider struct {
	stsClient  STSClient
	region     string
	accountID  string
	httpClient *http.Client
	initOnce   sync.Once
}

type AWSProviderOption func(*AWSProvider)

func WithAWSHTTPClient(client *http.Client) AWSProviderOption {
	return func(p *AWSProvider) {
		p.httpClient = client
	}
}

func WithSTSClient(client STSClient) AWSProviderOption {
	return func(p *AWSProvider) {
		p.stsClient = client
	}
}

func NewAWSProvider(region, accountID string, opts ...AWSProviderOption) *AWSProvider {
	provider := &AWSProvider{
		region:    region,
		accountID: accountID,
	}
	for _, opt := range opts {
		opt(provider)
	}
	if provider.httpClient == nil {
		provider.httpClient = http.DefaultClient
	}
	return provider
}

func (a *AWSProvider) Name() string {
	return "aws-iam"
}

func (a *AWSProvider) ProvisionCredential(ctx context.Context, req *ProvisionRequest) (*Credential, error) {
	var initErr error
	a.initOnce.Do(func() {
		if a.stsClient == nil {
			cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(a.region))
			if err != nil {
				initErr = fmt.Errorf("aws: failed to load AWS config: %w", err)
				return
			}
			a.stsClient = sts.NewFromConfig(cfg, func(o *sts.Options) {
				o.HTTPClient = a.httpClient
			})
		}
	})
	if initErr != nil {
		return nil, initErr
	}

	roleArn, err := a.resolveRoleARN(req.Scope.Resource)
	if err != nil {
		return nil, fmt.Errorf("aws: %w", err)
	}

	policyDoc, err := a.buildPolicy(req.Scope)
	if err != nil {
		return nil, fmt.Errorf("aws: failed to build policy: %w", err)
	}
	policyJSON, _ := json.Marshal(policyDoc)

	durationSec := int32(req.TTLSeconds)
	if durationSec < 900 {
		durationSec = 900
	}

	sessionName := fmt.Sprintf("ans-%s-%s", sanitizeSessionName(req.AgentID), generateRequestID()[:8])

	result, err := a.stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleArn),
		RoleSessionName: aws.String(sessionName),
		DurationSeconds: aws.Int32(durationSec),
		Policy:          aws.String(string(policyJSON)),
	})
	if err != nil {
		return nil, fmt.Errorf("aws: STS AssumeRole failed: %w", err)
	}

	if result.Credentials == nil {
		return nil, fmt.Errorf("aws: STS returned empty credentials")
	}

	now := time.Now()
	credID := generateRequestID()

	return &Credential{
		CredentialID: credID,
		AgentID:      req.AgentID,
		ProviderName: "aws-iam",
		Type:         "aws-sts",
		Secret:       *result.Credentials.SecretAccessKey,
		Metadata: map[string]string{
			"role_arn":      roleArn,
			"session_name":  sessionName,
			"access_key":    *result.Credentials.AccessKeyId,
			"session_token": *result.Credentials.SessionToken,
			"region":        a.region,
			"expiration":    result.Credentials.Expiration.Format(time.RFC3339),
		},
		Scope:        req.Scope,
		IssuedAt:     now,
		ExpiresAt:    now.Add(time.Duration(req.TTLSeconds) * time.Second),
		RequestID:    req.RequestID,
		PreReceiptID: req.PreReceiptID,
	}, nil
}

func (a *AWSProvider) RevokeCredential(ctx context.Context, credentialID string) error {
	return nil
}

func (a *AWSProvider) ValidateScope(scope Scope) error {
	if scope.Resource == "" {
		return fmt.Errorf("aws resource cannot be empty")
	}
	if !strings.HasPrefix(scope.Resource, "s3://") &&
		!strings.HasPrefix(scope.Resource, "arn:aws:") &&
		!strings.HasPrefix(scope.Resource, "arn:aws-cn:") &&
		!strings.HasPrefix(scope.Resource, "arn:aws-us-gov:") {
		return fmt.Errorf("invalid AWS resource format: %s", scope.Resource)
	}
	for _, perm := range scope.Permissions {
		switch perm {
		case "read", "write", "delete", "list", "put", "get", "update", "*":
		default:
			return fmt.Errorf("invalid AWS permission: %s", perm)
		}
	}
	return nil
}

func (a *AWSProvider) resolveRoleARN(resource string) (string, error) {
	if strings.HasPrefix(resource, "arn:") {
		return resource, nil
	}
	if strings.HasPrefix(resource, "s3://") {
		return fmt.Sprintf("arn:aws:iam::%s:role/ans-s3-access", a.accountID), nil
	}
	return "", fmt.Errorf("unsupported AWS resource format: %s", resource)
}

func (a *AWSProvider) buildPolicy(scope Scope) (map[string]interface{}, error) {
	actions := mapPermissionsToActions(scope.Permissions)
	resource := normalizeResource(scope.Resource)

	return map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect":   "Allow",
				"Action":   actions,
				"Resource": resource,
			},
		},
	}, nil
}

func mapPermissionsToActions(perms []string) []string {
	if len(perms) == 0 {
		return []string{"*"}
	}
	actionMap := map[string]string{
		"read":   "s3:GetObject",
		"write":  "s3:PutObject",
		"delete": "s3:DeleteObject",
		"list":   "s3:ListBucket",
		"get":    "s3:GetObject",
		"put":    "s3:PutObject",
	}
	actions := make([]string, 0, len(perms))
	for _, p := range perms {
		if p == "*" {
			return []string{"*"}
		}
		if a, ok := actionMap[p]; ok {
			actions = append(actions, a)
		} else {
			actions = append(actions, p)
		}
	}
	return actions
}

func normalizeResource(resource string) string {
	if strings.HasPrefix(resource, "arn:") {
		return resource
	}
	if strings.HasPrefix(resource, "s3://") {
		bucketPath := strings.TrimPrefix(resource, "s3://")
		return fmt.Sprintf("arn:aws:s3:::%s", bucketPath)
	}
	return resource
}

func sanitizeSessionName(name string) string {
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)
	if len(sanitized) > 32 {
		sanitized = sanitized[:32]
	}
	return sanitized
}


