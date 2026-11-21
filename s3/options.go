package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	goverrors "github.com/haipham22/govern/errors"
)

// Option configures a Client.
type Option func(*options)

type options struct {
	endpoint      string
	region        string
	accessKey     string
	secretKey     string
	pathStyle     bool
	secure        bool
	defaultBucket string
	debugLogging  bool
}

// WithEndpoint sets a custom endpoint (MinIO, R2, localstack). The scheme is
// derived from WithSecure when the endpoint carries none.
func WithEndpoint(u string) Option {
	return func(o *options) { o.endpoint = u }
}

// WithRegion sets the region. Default us-east-1.
func WithRegion(r string) Option {
	return func(o *options) { o.region = r }
}

// WithCredentials sets static access/secret keys. When omitted the SDK's
// default provider chain applies (env, shared config, IAM role).
func WithCredentials(access, secret string) Option {
	return func(o *options) {
		o.accessKey = access
		o.secretKey = secret
	}
}

// WithPathStyle enables path-style addressing. Enable for MinIO.
func WithPathStyle(v bool) Option {
	return func(o *options) { o.pathStyle = v }
}

// WithSecure selects https (true, default) or http (MinIO local).
func WithSecure(v bool) Option {
	return func(o *options) { o.secure = v }
}

// WithDefaultBucket sets the bucket used when a call passes "".
func WithDefaultBucket(b string) Option {
	return func(o *options) { o.defaultBucket = b }
}

// WithDebugLogging turns on SDK request/response logging.
func WithDebugLogging(v bool) Option {
	return func(o *options) { o.debugLogging = v }
}

// resolveOptions applies opts and derives endpoint scheme from secure.
func resolveOptions(opts ...Option) *options {
	cfg := &options{region: "us-east-1", secure: true}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.endpoint != "" && !strings.Contains(cfg.endpoint, "://") {
		if cfg.secure {
			cfg.endpoint = "https://" + cfg.endpoint
		} else {
			cfg.endpoint = "http://" + cfg.endpoint
		}
	}
	return cfg
}

// New creates an S3/MinIO client from options.
func New(opts ...Option) (*Client, error) {
	cfg := resolveOptions(opts...)

	loadOpts := []func(*config.LoadOptions) error{config.WithRegion(cfg.region)}
	if cfg.accessKey != "" {
		loadOpts = append(loadOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.accessKey, cfg.secretKey, "")))
	}
	if cfg.endpoint != "" {
		loadOpts = append(loadOpts, config.WithBaseEndpoint(cfg.endpoint))
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, goverrors.WrapCode(goverrors.CodeInternal, fmt.Errorf("s3: load config: %w", err))
	}

	api := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.pathStyle
		if cfg.debugLogging {
			o.ClientLogMode = aws.LogRequest | aws.LogResponse
		}
	})
	return &Client{
		api:       api,
		presigner: s3.NewPresignClient(api),
		bucket:    cfg.defaultBucket,
		region:    cfg.region,
	}, nil
}
