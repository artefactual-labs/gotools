// Package bucket provides functions to open gocloud buckets.
package bucket

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gocloud.dev/blob"
	"gocloud.dev/blob/azureblob"
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

	// Azure configures authentication for azblob URLs when callers need to
	// provide credentials programmatically instead of relying on environment
	// variables or the default Go CDK opener.
	Azure *AzureConfig
}

type AzureConfig struct {
	// StorageAccount identifies the Azure Blob Storage account to connect to.
	// It is required for shared key auth and optional for other auth modes,
	// where the account can also come from the URL or environment.
	StorageAccount string

	// StorageKey enables shared key authentication.
	StorageKey string

	// TenantID, ClientID, and ClientSecret enable Azure AD client secret
	// authentication when all are set. If neither shared key nor a complete
	// client secret tuple is provided, the default Azure credential chain is
	// used.
	TenantID     string
	ClientID     string
	ClientSecret string
}

// NewWithConfig opens a bucket based on the provided configuration. It defaults
// to using AWS SDK v2 via s3blob.OpenBucketV2 unless the URL field is
// specified. URL based configs are opened via blob.OpenBucket, except azblob
// URLs with Azure config overrides, which use a custom Azure URL opener.
func NewWithConfig(ctx context.Context, c *Config) (*blob.Bucket, error) {
	if c == nil {
		return nil, errors.New("config is undefined")
	}

	var (
		b   *blob.Bucket
		err error
	)

	if isAzureBucketURL(c.URL) && c.Azure != nil {
		b, err = openAzureWithOverrides(ctx, c.URL, c.Azure)
	} else if c.URL != "" {
		b, err = openWithURL(ctx, c.URL)
	} else {
		b, err = openWithConfig(ctx, c)
	}

	return b, err
}

func isAzureBucketURL(bucketURL string) bool {
	u, err := url.Parse(bucketURL)
	return err == nil && u.Scheme == azureblob.Scheme
}

// openAzureWithOverrides returns a bucket with a custom opener for azblob URLs
// when Azure config overrides are provided. Client secret auth takes precedence
// over shared key auth when both are configured. Partial shared key or client
// secret configs are rejected; all other cases fall back to Azure's default
// credential chain.
func openAzureWithOverrides(ctx context.Context, u string, c *AzureConfig) (*blob.Bucket, error) {
	if c == nil {
		return nil, fmt.Errorf("open Azure bucket from URL %q with overrides: config is undefined", u)
	}

	makeClient := azureblob.NewDefaultClient
	if c.TenantID != "" || c.ClientID != "" || c.ClientSecret != "" {
		if c.TenantID == "" || c.ClientID == "" || c.ClientSecret == "" {
			return nil, fmt.Errorf(
				"open Azure bucket from URL %q with overrides: "+
					"client secret auth requires tenant ID, client ID, and client secret",
				u,
			)
		}
		makeClient = func(u azureblob.ServiceURL, n azureblob.ContainerName) (*container.Client, error) {
			containerURL, err := url.JoinPath(string(u), string(n))
			if err != nil {
				return nil, err
			}
			cred, err := azidentity.NewClientSecretCredential(c.TenantID, c.ClientID, c.ClientSecret, nil)
			if err != nil {
				return nil, err
			}
			return container.NewClient(containerURL, cred, nil)
		}
	} else if c.StorageKey != "" {
		if c.StorageAccount == "" {
			return nil, fmt.Errorf(
				"open Azure bucket from URL %q with overrides: "+
					"shared key auth requires storage account and storage key",
				u,
			)
		}
		makeClient = func(u azureblob.ServiceURL, n azureblob.ContainerName) (*container.Client, error) {
			containerURL, err := url.JoinPath(string(u), string(n))
			if err != nil {
				return nil, err
			}
			cred, err := azblob.NewSharedKeyCredential(c.StorageAccount, c.StorageKey)
			if err != nil {
				return nil, err
			}
			return container.NewClientWithSharedKeyCredential(containerURL, cred, nil)
		}
	}

	urlMux := new(blob.URLMux)
	urlMux.RegisterBucket(azureblob.Scheme, &azureblob.URLOpener{
		MakeClient:        makeClient,
		ServiceURLOptions: azureblob.ServiceURLOptions{AccountName: c.StorageAccount},
	})
	b, err := urlMux.OpenBucket(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("open Azure bucket from URL %q with overrides: %v", u, err)
	}

	return b, nil
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
