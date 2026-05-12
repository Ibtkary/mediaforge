// Package r2 wires Cloudflare R2 into MediaForge through the S3-compatible API.
package r2

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Ibtkary/mediaforge"
	s3provider "github.com/Ibtkary/mediaforge/providers/s3"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

const region = "auto"

// Config contains the Cloudflare R2 settings needed to build a MediaForge client.
type Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Endpoint        string
}

// NewStorageAndSigner creates MediaForge storage and URL signer adapters for R2.
func NewStorageAndSigner(ctx context.Context, cfg Config) (mediaforge.ObjectStorage, mediaforge.URLSigner, error) {
	endpoint, err := resolveEndpoint(cfg)
	if err != nil {
		return nil, nil, err
	}
	if err := validateCredentials(cfg); err != nil {
		return nil, nil, err
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			strings.TrimSpace(cfg.AccessKeyID),
			strings.TrimSpace(cfg.SecretAccessKey),
			"",
		)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("r2 config: load aws config: %w", err)
	}

	client := awss3.NewFromConfig(awsCfg, func(o *awss3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
	bucket := strings.TrimSpace(cfg.Bucket)

	return s3provider.NewStorage(client, bucket), s3provider.NewSigner(awss3.NewPresignClient(client), bucket), nil
}

// NewClient creates a MediaForge client backed by Cloudflare R2.
func NewClient(ctx context.Context, cfg Config, opts ...mediaforge.Option) (*mediaforge.Client, error) {
	storage, signer, err := NewStorageAndSigner(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return mediaforge.NewClient(storage, signer, opts...)
}

func validateCredentials(cfg Config) error {
	var missing []string
	if strings.TrimSpace(cfg.AccessKeyID) == "" {
		missing = append(missing, "access_key_id")
	}
	if strings.TrimSpace(cfg.SecretAccessKey) == "" {
		missing = append(missing, "secret_access_key")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		missing = append(missing, "bucket")
	}
	if len(missing) > 0 {
		return fmt.Errorf("r2 config: missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func resolveEndpoint(cfg Config) (string, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		accountID := strings.TrimSpace(cfg.AccountID)
		if accountID == "" {
			return "", fmt.Errorf("r2 config: account_id is required when endpoint is empty")
		}
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	}
	endpoint = strings.TrimRight(endpoint, "/")

	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("r2 config: endpoint must be a valid absolute URL")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", fmt.Errorf("r2 config: endpoint scheme must be http or https")
	}
	return endpoint, nil
}
