// Package storage wraps an S3-compatible object store (Cloudflare R2 in
// production, MinIO locally) behind a small interface. The API uses it to
// issue presigned upload URLs and short-lived presigned download URLs for
// hidden/internal attachments.
//
// We deliberately do not stream large files through the API. Uploads go
// directly browser → R2 via a presigned PUT URL, and the API only tracks
// metadata in case_attachments after R2 confirms the object.
package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config carries the runtime settings for an S3-compatible client.
type Config struct {
	Endpoint        string
	Region          string
	AccessKey       string
	SecretKey       string
	Bucket          string
	PublicBaseURL   string
	ForcePathStyle  bool
}

// Client wraps an *s3.Client + presigner with our naming conventions.
type Client struct {
	cfg       Config
	s3        *s3.Client
	presigner *s3.PresignClient
}

// NewClient builds a Client from cfg. It does NOT verify bucket access —
// callers should call EnsureBucket separately when running in dev.
func NewClient(ctx context.Context, c Config) (*Client, error) {
	if c.AccessKey == "" || c.SecretKey == "" {
		return nil, errors.New("storage: access key / secret key are required")
	}
	if c.Bucket == "" {
		return nil, errors.New("storage: bucket is required")
	}
	loadOpts := []func(*config.LoadOptions) error{
		config.WithRegion(orDefault(c.Region, "auto")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.AccessKey, c.SecretKey, "")),
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("storage: load aws config: %w", err)
	}
	s3c := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if c.Endpoint != "" {
			o.BaseEndpoint = aws.String(c.Endpoint)
		}
		o.UsePathStyle = c.ForcePathStyle
	})
	return &Client{cfg: c, s3: s3c, presigner: s3.NewPresignClient(s3c)}, nil
}

// EnsureBucket creates the bucket if it does not already exist. Idempotent.
// Useful in local docker-compose; in production the bucket is provisioned
// out of band.
func (c *Client) EnsureBucket(ctx context.Context) error {
	_, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(c.cfg.Bucket)})
	if err == nil {
		return nil
	}
	_, err = c.s3.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(c.cfg.Bucket)})
	if err != nil && !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
		return fmt.Errorf("storage: create bucket: %w", err)
	}
	return nil
}

// PresignedPutInput is what callers pass to PresignPut.
type PresignedPutInput struct {
	Key         string
	ContentType string
	Expiry      time.Duration
}

// PresignedURL bundles a signed URL with its expiry.
type PresignedURL struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PresignPut returns a short-lived URL the browser can PUT bytes to.
func (c *Client) PresignPut(ctx context.Context, in PresignedPutInput) (*PresignedURL, error) {
	if in.Expiry == 0 {
		in.Expiry = 10 * time.Minute
	}
	req, err := c.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.cfg.Bucket),
		Key:         aws.String(in.Key),
		ContentType: aws.String(in.ContentType),
	}, s3.WithPresignExpires(in.Expiry))
	if err != nil {
		return nil, fmt.Errorf("storage: presign PUT: %w", err)
	}
	return &PresignedURL{URL: req.URL, ExpiresAt: time.Now().Add(in.Expiry)}, nil
}

// PresignGet returns a short-lived URL the browser can GET bytes from.
// Used for hidden / internal attachments that must not be on the public CDN.
func (c *Client) PresignGet(ctx context.Context, key string, expiry time.Duration) (*PresignedURL, error) {
	if expiry == 0 {
		expiry = 5 * time.Minute
	}
	req, err := c.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.cfg.Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return nil, fmt.Errorf("storage: presign GET: %w", err)
	}
	return &PresignedURL{URL: req.URL, ExpiresAt: time.Now().Add(expiry)}, nil
}

// HeadObject returns true when the object exists (used by finalize).
func (c *Client) HeadObject(ctx context.Context, key string) (bool, int64, string, error) {
	out, err := c.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Smithy/AWS SDK doesn't always expose typed not-found cleanly; use
		// substring check as a fallback.
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
			return false, 0, "", nil
		}
		return false, 0, "", fmt.Errorf("storage: head object: %w", err)
	}
	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	etag := ""
	if out.ETag != nil {
		etag = strings.Trim(*out.ETag, `"`)
	}
	return true, size, etag, nil
}

// DeleteObject removes a single object.
func (c *Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.cfg.Bucket),
		Key:    aws.String(key),
	})
	return err
}

// PublicURL returns the canonical URL where a `public` object is served.
// In production this is a Cloudflare custom domain in front of R2.
func (c *Client) PublicURL(key string) string {
	if c.cfg.PublicBaseURL == "" {
		return ""
	}
	return strings.TrimRight(c.cfg.PublicBaseURL, "/") + "/" + strings.TrimLeft(key, "/")
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
