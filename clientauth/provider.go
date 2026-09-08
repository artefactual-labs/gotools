package clientauth

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	defaultTokenExpiryLeeway       = 30 * time.Second
	defaultRetryMaxAttempts        = 3
	defaultRetryInitialInterval    = 500 * time.Millisecond
	defaultRetryMaxInterval        = 2 * time.Second
	defaultRetryBackoffCoefficient = 2.0
)

func validateTokenProviderSettings(
	tokenExpiryLeeway time.Duration,
	retryMaxAttempts int,
	retryInitialInterval time.Duration,
	retryMaxInterval time.Duration,
	retryBackoffCoefficient float64,
) []error {
	var errs []error
	if retryMaxAttempts < 1 {
		errs = append(errs, errors.New("invalid OIDC retry max attempts, value must be >= 1"))
	}
	if retryInitialInterval <= 0 || retryMaxInterval <= 0 || tokenExpiryLeeway <= 0 {
		errs = append(errs, errors.New("invalid OIDC duration configuration, values must be > 0"))
	}
	if retryMaxInterval < retryInitialInterval {
		errs = append(errs, errors.New(
			"invalid OIDC retry interval configuration, max interval must be >= initial interval",
		))
	}
	if retryBackoffCoefficient < 1 {
		errs = append(errs, errors.New("invalid OIDC retry backoff coefficient, value must be >= 1"))
	}

	return errs
}

// AccessTokenProvider returns access tokens for outbound authenticated
// requests.
type AccessTokenProvider interface {
	// AccessToken returns a bearer token suitable for an Authorization header.
	AccessToken(ctx context.Context) (string, error)
}

type tokenRequestFunc func(context.Context, *oauth2.Token) (*oauth2.Token, error)

// oidcAccessTokenProvider fetches and caches OAuth2 access tokens.
type oidcAccessTokenProvider struct {
	mu                      sync.RWMutex // Guards token reads and refresh.
	token                   *oauth2.Token
	tokenRequest            tokenRequestFunc
	tokenExpiryLeeway       time.Duration
	retryMaxAttempts        int
	retryInitialInterval    time.Duration
	retryMaxInterval        time.Duration
	retryBackoffCoefficient float64
}

var _ AccessTokenProvider = (*oidcAccessTokenProvider)(nil)

func resolveOIDCTokenURL(ctx context.Context, providerURL, tokenURL string) (string, error) {
	if tokenURL == "" {
		provider, err := oidc.NewProvider(ctx, providerURL)
		if err != nil {
			return "", fmt.Errorf("discover OIDC provider: %w", err)
		}
		tokenURL = provider.Endpoint().TokenURL
	}
	if tokenURL == "" {
		return "", errors.New("missing OIDC token endpoint URL")
	}

	return tokenURL, nil
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
	for attempt := 1; attempt <= p.retryMaxAttempts; attempt++ {
		token, err = p.tokenRequest(ctx, p.token)
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
