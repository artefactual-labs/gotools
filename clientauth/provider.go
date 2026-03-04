package clientauth

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	defaultTokenExpiryLeeway       = 30 * time.Second
	defaultRetryMaxAttempts        = 3
	defaultRetryInitialInterval    = 500 * time.Millisecond
	defaultRetryMaxInterval        = 2 * time.Second
	defaultRetryBackoffCoefficient = 2.0
)

// AccessTokenProvider returns access tokens for outbound authenticated
// requests.
type AccessTokenProvider interface {
	// AccessToken returns a bearer token suitable for an Authorization header.
	AccessToken(ctx context.Context) (string, error)
}

// OIDCAccessTokenProviderConfig configures an [OIDCAccessTokenProvider] that
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
	if c.TokenExpiryLeeway == 0 {
		c.TokenExpiryLeeway = defaultTokenExpiryLeeway
	}
	if c.RetryMaxAttempts == 0 {
		c.RetryMaxAttempts = defaultRetryMaxAttempts
	}
	if c.RetryInitialInterval == 0 {
		c.RetryInitialInterval = defaultRetryInitialInterval
	}
	if c.RetryMaxInterval == 0 {
		c.RetryMaxInterval = defaultRetryMaxInterval
	}
	if c.RetryBackoffCoefficient == 0 {
		c.RetryBackoffCoefficient = defaultRetryBackoffCoefficient
	}
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
	if c.RetryMaxAttempts < 1 {
		errs = append(errs, errors.New("invalid OIDC retry max attempts, value must be >= 1"))
	}
	if c.RetryInitialInterval <= 0 || c.RetryMaxInterval <= 0 || c.TokenExpiryLeeway <= 0 {
		errs = append(errs, errors.New("invalid OIDC duration configuration, values must be > 0"))
	}
	if c.RetryMaxInterval < c.RetryInitialInterval {
		errs = append(errs, errors.New(
			"invalid OIDC retry interval configuration, max interval must be >= initial interval",
		))
	}
	if c.RetryBackoffCoefficient < 1 {
		errs = append(errs, errors.New("invalid OIDC retry backoff coefficient, value must be >= 1"))
	}

	return errors.Join(errs...)
}

// oidcAccessTokenProvider fetches and caches access tokens using the OAuth2
// client credentials flow.
type oidcAccessTokenProvider struct {
	mu                      sync.RWMutex // Guards token reads and refresh.
	token                   *oauth2.Token
	cc                      clientcredentials.Config
	tokenExpiryLeeway       time.Duration
	retryMaxAttempts        int
	retryInitialInterval    time.Duration
	retryMaxInterval        time.Duration
	retryBackoffCoefficient float64
}

var _ AccessTokenProvider = (*oidcAccessTokenProvider)(nil)

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

	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		provider, err := oidc.NewProvider(ctx, cfg.ProviderURL)
		if err != nil {
			return nil, fmt.Errorf("discover OIDC provider: %w", err)
		}
		tokenURL = provider.Endpoint().TokenURL
	}
	if tokenURL == "" {
		return nil, errors.New("missing OIDC token endpoint URL")
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
		cc:                      cc,
		tokenExpiryLeeway:       cfg.TokenExpiryLeeway,
		retryMaxAttempts:        cfg.RetryMaxAttempts,
		retryInitialInterval:    cfg.RetryInitialInterval,
		retryMaxInterval:        cfg.RetryMaxInterval,
		retryBackoffCoefficient: cfg.RetryBackoffCoefficient,
	}, nil
}

// AccessToken returns a cached token when still valid, or fetches a new one.
func (p *oidcAccessTokenProvider) AccessToken(ctx context.Context) (string, error) {
	p.mu.RLock()
	if !p.tokenNeedsRefresh() {
		defer p.mu.RUnlock()
		return p.token.AccessToken, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Re-check: another goroutine may have refreshed while we waited.
	if !p.tokenNeedsRefresh() {
		return p.token.AccessToken, nil
	}

	token, err := p.requestToken(ctx)
	if err != nil {
		return "", fmt.Errorf("request OIDC token: %w", err)
	}
	p.token = token

	return p.token.AccessToken, nil
}

// tokenNeedsRefresh reports whether the cached token is missing or near expiry.
func (p *oidcAccessTokenProvider) tokenNeedsRefresh() bool {
	if p.token == nil || p.token.AccessToken == "" {
		return true
	}
	if p.token.Expiry.IsZero() {
		return false
	}

	return p.token.Expiry.Before(time.Now().Add(p.tokenExpiryLeeway))
}

// requestToken fetches a token, retrying transient failures with backoff.
func (p *oidcAccessTokenProvider) requestToken(ctx context.Context) (*oauth2.Token, error) {
	var err error
	var token *oauth2.Token
	ts := p.cc.TokenSource(ctx)
	for attempt := 1; attempt <= p.retryMaxAttempts; attempt++ {
		token, err = ts.Token()
		if err == nil {
			return token, nil
		}
		if !isRetryableErr(err) || attempt == p.retryMaxAttempts {
			break
		}

		if wait := p.backoff(attempt); wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}

	return nil, err
}

// backoff returns the retry wait duration for the given attempt.
func (p *oidcAccessTokenProvider) backoff(attempt int) time.Duration {
	if p.retryInitialInterval <= 0 {
		return 0
	}

	wait := float64(p.retryInitialInterval) * math.Pow(p.retryBackoffCoefficient, float64(attempt-1))
	if p.retryMaxInterval > 0 {
		wait = math.Min(wait, float64(p.retryMaxInterval))
	}

	return time.Duration(wait)
}

// isRetryableErr reports whether err is worth retrying.
func isRetryableErr(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var rerr *oauth2.RetrieveError
	if errors.As(err, &rerr) && rerr.Response != nil {
		return rerr.Response.StatusCode >= http.StatusInternalServerError
	}

	return true
}
