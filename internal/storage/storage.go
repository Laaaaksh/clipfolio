// Package storage puts and fetches video assets (originals, HLS renditions,
// thumbnails) in an S3-compatible bucket. It works unmodified against AWS S3,
// Cloudflare R2, Backblaze B2, and MinIO - clipfolio never assumes free
// unlimited storage or bandwidth; the operator brings their own bucket.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/laaaaksh/clipfolio/internal/config"
)

// Store puts, fetches, and deletes objects in an S3-compatible bucket.
type Store struct {
	client     *s3.Client
	bucket     string
	publicBase string
}

// New builds a Store from S3-compatible credentials in cfg.
func New(ctx context.Context, cfg config.Config) (*Store, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		}
		o.UsePathStyle = cfg.S3ForcePathStyle
	})

	publicBase := cfg.S3PublicBaseURL
	if publicBase == "" {
		publicBase = fmt.Sprintf("%s/%s", cfg.S3Endpoint, cfg.S3Bucket)
	}

	return &Store{client: client, bucket: cfg.S3Bucket, publicBase: publicBase}, nil
}

// Put uploads data to key with the given content type.
func (s *Store) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}

// Get fetches the object at key.
func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", key, err)
	}
	defer func() { _ = out.Body.Close() }()
	return io.ReadAll(out.Body)
}

// DeletePrefix removes every object under the given key prefix (a video's
// whole HLS directory, for instance).
func (s *Store) DeletePrefix(ctx context.Context, prefix string) error {
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list %s: %w", prefix, err)
		}
		for _, obj := range page.Contents {
			if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(s.bucket),
				Key:    obj.Key,
			}); err != nil {
				return fmt.Errorf("delete %s: %w", *obj.Key, err)
			}
		}
	}
	return nil
}

// PublicURL returns the URL viewers fetch this key from.
func (s *Store) PublicURL(key string) string {
	return fmt.Sprintf("%s/%s", s.publicBase, key)
}
