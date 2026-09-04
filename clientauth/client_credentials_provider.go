package clientauth

import (
	"cmp"
	"context"
	"errors"
	"net/url"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// OIDCAccessTokenProviderConfig configures an [AccessTokenProvider] that
// uses the OAuth2 client credentials flow.
type OIDCAccessTokenProviderConfig struct {
	// ProviderURL is the OIDC issuer URL for endpoint discovery.
	// Ignored when TokenURL is set.
	ProviderURL string
	// TokenURL is the token endpoint. Discovered from ProviderURL if empty.
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string // Optional token scopes.
	Audience     string   // Optional audience endpoint parameter.
	// TokenExpiryLeeway is a safety window to refresh tokens before expiry.
	TokenExpiryLeeway       time.Duration
	RetryMaxAttempts        int           // Total token endpoint attempts.
	RetryInitialInterval    time.Duration // Initial retry backoff.
	RetryMaxInterval        time.Duration // Upper bound for retry backoff.
	RetryBackoffCoefficient float64       // Exponential backoff multiplier.
}

// setDefaults fills zero-valued fields with defaults.
func (c *OIDCAccessTokenProviderConfig) setDefaults() {
	c.TokenExpiryLeeway = cmp.Or(c.TokenExpiryLeeway, defaultTokenExpiryLeeway)
	c.RetryMaxAttempts = cmp.Or(c.RetryMaxAttempts, defaultRetryMaxAttempts)
	c.RetryInitialInterval = cmp.Or(c.RetryInitialInterval, defaultRetryInitialInterval)
	c.RetryMaxInterval = cmp.Or(c.RetryMaxInterval, defaultRetryMaxInterval)
	c.RetryBackoffCoefficient = cmp.Or(c.RetryBackoffCoefficient, defaultRetryBackoffCoefficient)
}

// Validate fills zero-valued fields with defaults and validates the config.
func (c *OIDCAccessTokenProviderConfig) Validate() error {
	c.setDefaults()

	var errs []error
	if c.ProviderURL == "" && c.TokenURL == "" {
		errs = append(errs, errors.New("missing OIDC providerURL or tokenURL"))
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		errs = append(errs, errors.New("missing OIDC client credentials"))
	}
	errs = append(errs, validateTokenProviderSettings(
		c.TokenExpiryLeeway,
		c.RetryMaxAttempts,
		c.RetryInitialInterval,
		c.RetryMaxInterval,
		c.RetryBackoffCoefficient,
	)...)

	return errors.Join(errs...)
}

// NewOIDCAccessTokenProvider builds an [AccessTokenProvider] from OIDC/OAuth2
// client credentials settings. It calls [OIDCAccessTokenProviderConfig.Validate]
// before proceeding.
func NewOIDCAccessTokenProvider(
	ctx context.Context,
	cfg OIDCAccessTokenProviderConfig,
) (AccessTokenProvider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	tokenURL, err := resolveOIDCTokenURL(ctx, cfg.ProviderURL, cfg.TokenURL)
	if err != nil {
		return nil, err
	}

	cc := clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     tokenURL,
		Scopes:       cfg.Scopes,
	}
	if cfg.Audience != "" {
		cc.EndpointParams = url.Values{"audience": []string{cfg.Audience}}
	}

	return &oidcAccessTokenProvider{
		tokenRequest: func(ctx context.Context, _ *oauth2.Token) (*oauth2.Token, error) {
			// Each call creates a token source with no cached token, so Token always
			// issues a new client credentials request instead of reusing the previous
			// token. The password flow needs different handling to force a refresh.
			return cc.TokenSource(ctx).Token()
		},
		tokenExpiryLeeway:       cfg.TokenExpiryLeeway,
		retryMaxAttempts:        cfg.RetryMaxAttempts,
		retryInitialInterval:    cfg.RetryInitialInterval,
		retryMaxInterval:        cfg.RetryMaxInterval,
		retryBackoffCoefficient: cfg.RetryBackoffCoefficient,
	}, nil
}
