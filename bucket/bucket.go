// Package bucket provides functions to open gocloud buckets.
package bucket

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob"
	"gocloud.dev/blob/s3blob"
)

type Config struct {
	// URL specifies the connection string for a bucket as described in
	// https://gocloud.dev/howto/blob/. If provided, this URL will be used to
	// open the bucket directly.
	URL string

	// These fields are used for direct access to S3-compatible services when
	// the URL field is not specified. They provide a more granular
	// configuration using specific credentials and connection details.
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Token     string
	Profile   string
	Region    string
	PathStyle bool
}

// NewWithConfig opens a bucket based on the provided configuration. It defaults
// to using AWS SDK v2 via s3blob.OpenBucketV2 unless the URL field is
// specified, in which case it uses blob.OpenBucket.
func NewWithConfig(ctx context.Context, c *Config) (*blob.Bucket, error) {
	if c == nil {
		return nil, errors.New("config is undefined")
	}

	var (
		b   *blob.Bucket
		err error
	)

	if c.URL != "" {
		b, err = openWithURL(ctx, c.URL)
	} else {
		b, err = openWithConfig(ctx, c)
	}

	return b, err
}

func openWithURL(ctx context.Context, url string) (*blob.Bucket, error) {
	b, err := blob.OpenBucket(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("open bucket from URL %q: %v", url, err)
	}
	return b, nil
}

func openWithConfig(ctx context.Context, c *Config) (*blob.Bucket, error) {
	awscfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithSharedConfigProfile(c.Profile),
		config.WithRegion(c.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				c.AccessKey, c.SecretKey, c.Token,
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS default config: %v", err)
	}

	client := s3.NewFromConfig(awscfg, func(opts *s3.Options) {
		opts.UsePathStyle = c.PathStyle
		if c.Endpoint != "" {
			// BaseEndpoint customizes only this S3 client. Leaving it unset lets
			// the AWS SDK use its normal endpoint resolution for AWS S3.
			opts.BaseEndpoint = aws.String(normalizeEndpoint(c.Endpoint))
		}
	})
	b, err := s3blob.OpenBucketV2(ctx, client, c.Bucket, nil)
	if err != nil {
		return nil, fmt.Errorf("open bucket: %v", err)
	}

	return b, nil
}

func normalizeEndpoint(endpoint string) string {
	if u, err := url.Parse(endpoint); err == nil {
		if !strings.HasPrefix(u.Scheme, "http") {
			return "http://" + endpoint
		}
	}
	return endpoint
}
