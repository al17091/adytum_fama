package awsauth

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/processcreds"
)

// ConfigOptions holds settings for resolving AWS configuration
type ConfigOptions struct {
	Region          string
	CertPath        string
	KeyPath         string
	TrustAnchorARN  string
	ProfileARN      string
	RoleARN         string
}

// LoadAWSConfig resolves AWS SDK configuration supporting both standard credentials
// and AWS IAM Roles Anywhere via aws_signing_helper credential_process
func LoadAWSConfig(ctx context.Context, opt ConfigOptions) (aws.Config, error) {
	region := opt.Region
	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = "ap-northeast-1"
		}
	}

	// Read environment variables if not passed explicitly in ConfigOptions
	certPath := opt.CertPath
	if certPath == "" {
		certPath = os.Getenv("AWS_ROLES_ANYWHERE_CERT_PATH")
	}
	keyPath := opt.KeyPath
	if keyPath == "" {
		keyPath = os.Getenv("AWS_ROLES_ANYWHERE_KEY_PATH")
	}
	trustAnchorARN := opt.TrustAnchorARN
	if trustAnchorARN == "" {
		trustAnchorARN = os.Getenv("AWS_ROLES_ANYWHERE_TRUST_ANCHOR_ARN")
	}
	profileARN := opt.ProfileARN
	if profileARN == "" {
		profileARN = os.Getenv("AWS_ROLES_ANYWHERE_PROFILE_ARN")
	}
	roleARN := opt.RoleARN
	if roleARN == "" {
		roleARN = os.Getenv("AWS_ROLES_ANYWHERE_ROLE_ARN")
	}

	// If Roles Anywhere parameters are configured, use aws_signing_helper process provider
	if certPath != "" && keyPath != "" && trustAnchorARN != "" && profileARN != "" && roleARN != "" {
		cmd := fmt.Sprintf(
			"aws_signing_helper credential-process --certificate %s --private-key %s --trust-anchor-arn %s --profile-arn %s --role-arn %s",
			certPath, keyPath, trustAnchorARN, profileARN, roleARN,
		)
		processProvider := processcreds.NewProvider(cmd)

		return config.LoadDefaultConfig(
			ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(processProvider),
		)
	}

	// Fallback to standard AWS credentials chain (env vars, ~/.aws/credentials, IAM Identity Center)
	return config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
	)
}
