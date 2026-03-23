package s3x

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	endpoint      string
	publicBucket  string
	privateBucket string
	pathPrefix    string
}

type Config struct {
	Endpoint      string
	AccessKey     string
	SecretKey     string
	Region        string
	PublicBucket  string
	PrivateBucket string
	PathPrefix    string
}

func New(ctx context.Context, cfg *Config) (*Client, error) {
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(cfg.Endpoint)
	})

	return &Client{
		client:        s3Client,
		presignClient: s3.NewPresignClient(s3Client),
		endpoint:      strings.TrimRight(cfg.Endpoint, "/"),
		publicBucket:  cfg.PublicBucket,
		privateBucket: cfg.PrivateBucket,
		pathPrefix:    cfg.PathPrefix,
	}, nil
}

func (c *Client) Close() {}
