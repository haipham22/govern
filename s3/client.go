// Package s3 wraps S3/MinIO object storage on AWS SDK v2: upload, download,
// stat, remove, presigned GET and idempotent bucket creation.
package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	goverrors "github.com/haipham22/govern/errors"
)

// ObjectInfo describes a stored object (Stat result).
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
}

// s3API is the subset of the S3 API the wrapper uses; injectable in tests.
type s3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
}

// Client is a thin S3/MinIO wrapper. Create with New.
type Client struct {
	api       s3API
	presigner *s3.PresignClient
	bucket    string // default bucket; per-call bucket overrides it
	region    string
}

// bucketOr resolves the per-call bucket, falling back to the default.
func (c *Client) bucketOr(bucket string) string {
	if bucket != "" {
		return bucket
	}
	return c.bucket
}

// Upload PUTs r as the object body. Empty contentType defaults to
// application/octet-stream. An empty bucket uses the configured default.
func (c *Client) Upload(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := c.api.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucketOr(bucket)),
		Key:           aws.String(key),
		Body:          r,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return goverrors.WrapCode(goverrors.CodeInternal, fmt.Errorf("s3: upload %s: %w", key, err))
	}
	return nil
}

// Download opens the object body. The caller MUST Close the reader.
// A missing object surfaces the NOT_FOUND code.
func (c *Client) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	out, err := c.api.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucketOr(bucket)),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, goverrors.WrapCode(goverrors.CodeNotFound, fmt.Errorf("s3: download %s: %w", key, err))
		}
		return nil, goverrors.WrapCode(goverrors.CodeInternal, fmt.Errorf("s3: download %s: %w", key, err))
	}
	return out.Body, nil
}

// Stat describes the object without fetching its body.
// A missing object surfaces the NOT_FOUND code.
func (c *Client) Stat(ctx context.Context, bucket, key string) (ObjectInfo, error) {
	out, err := c.api.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucketOr(bucket)),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return ObjectInfo{}, goverrors.WrapCode(goverrors.CodeNotFound, fmt.Errorf("s3: stat %s: %w", key, err))
		}
		return ObjectInfo{}, goverrors.WrapCode(goverrors.CodeInternal, fmt.Errorf("s3: stat %s: %w", key, err))
	}
	return ObjectInfo{
		Key:          key,
		Size:         aws.ToInt64(out.ContentLength),
		ContentType:  aws.ToString(out.ContentType),
		LastModified: aws.ToTime(out.LastModified),
		ETag:         aws.ToString(out.ETag),
	}, nil
}

// Remove deletes the object. S3 delete of a missing key is a no-op, so
// this is idempotent.
func (c *Client) Remove(ctx context.Context, bucket, key string) error {
	_, err := c.api.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucketOr(bucket)),
		Key:    aws.String(key),
	})
	if err != nil {
		return goverrors.WrapCode(goverrors.CodeInternal, fmt.Errorf("s3: remove %s: %w", key, err))
	}
	return nil
}

// Presign returns a presigned GET URL valid for expiry.
func (c *Client) Presign(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	req, err := c.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucketOr(bucket)),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", goverrors.WrapCode(goverrors.CodeInternal, fmt.Errorf("s3: presign %s: %w", key, err))
	}
	return req.URL, nil
}

// EnsureBucket creates the bucket if missing (idempotent). Outside us-east-1
// the region is set as the bucket location constraint, per AWS semantics.
func (c *Client) EnsureBucket(ctx context.Context, bucket string) error {
	bucket = c.bucketOr(bucket)
	_, err := c.api.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err == nil {
		return nil
	}
	if !isNotFound(err) {
		return goverrors.WrapCode(goverrors.CodeInternal, fmt.Errorf("s3: head bucket %s: %w", bucket, err))
	}
	input := &s3.CreateBucketInput{Bucket: aws.String(bucket)}
	if c.region != "" && c.region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(c.region),
		}
	}
	if _, err := c.api.CreateBucket(ctx, input); err != nil {
		return goverrors.WrapCode(goverrors.CodeInternal, fmt.Errorf("s3: create bucket %s: %w", bucket, err))
	}
	return nil
}

// isNotFound reports whether err means "object or bucket does not exist"
// across GET/HEAD shapes of the SDK error.
func isNotFound(err error) bool {
	var notFound *types.NotFound
	if goverrors.As(err, &notFound) {
		return true
	}
	var noKey *types.NoSuchKey
	if goverrors.As(err, &noKey) {
		return true
	}
	var apiErr smithy.APIError
	return goverrors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFound"
}
