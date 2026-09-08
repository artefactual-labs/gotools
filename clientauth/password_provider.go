package clientauth

import (
	"cmp"
	"context"
	"errors"
	"time"

	"golang.org/x/oauth2"
)

// OIDCPasswordGrantAccessTokenProviderConfig configures an access token
// provider that uses the OAuth2 resource owner password credentials grant.
//
// Deprecated: The OAuth2 security best current practice prohibits use of the
// password grant. Use it only when integrating with a legacy authorization
// server and no safer grant is available.
// See https://www.rfc-editor.org/rfc/rfc9700.html#section-2.4.
type OIDCPasswordGrantAccessTokenProviderConfig struct {
	// ProviderURL is the OIDC issuer URL for endpoint discovery.
	// Ignored when TokenURL is set.
	ProviderURL string
	// TokenURL is the token endpoint. Discovered from ProviderURL if empty.
	TokenURL     string
	ClientID     string
	ClientSecret string // Optional for public clients.
	Username     string
	Password     string
	Scopes       []string // Optional token scopes.
	// TokenExpiryLeeway is a safety window to refresh tokens before expiry.
	TokenExpiryLeeway       time.Duration
	RetryMaxAttempts        int           // Total token endpoint attempts.
	RetryInitialInterval    time.Duration // Initial retry backoff.
	RetryMaxInterval        time.Duration // Upper bound for retry backoff.
	RetryBackoffCoefficient float64       // Exponential backoff multiplier.
}

// setDefaults fills zero-valued fields with defaults.
func (c *OIDCPasswordGrantAccessTokenProviderConfig) setDefaults() {
	c.TokenExpiryLeeway = cmp.Or(c.TokenExpiryLeeway, defaultTokenExpiryLeeway)
	c.RetryMaxAttempts = cmp.Or(c.RetryMaxAttempts, defaultRetryMaxAttempts)
	c.RetryInitialInterval = cmp.Or(c.RetryInitialInterval, defaultRetryInitialInterval)
	c.RetryMaxInterval = cmp.Or(c.RetryMaxInterval, defaultRetryMaxInterval)
	c.RetryBackoffCoefficient = cmp.Or(c.RetryBackoffCoefficient, defaultRetryBackoffCoefficient)
}

// Validate fills zero-valued fields with defaults and validates the config.
func (c *OIDCPasswordGrantAccessTokenProviderConfig) Validate() error {
	c.setDefaults()

	var errs []error
	if c.ProviderURL == "" && c.TokenURL == "" {
		errs = append(errs, errors.New("missing OIDC providerURL or tokenURL"))
	}
	if c.ClientID == "" {
		errs = append(errs, errors.New("missing OIDC client ID"))
	}
	if c.Username == "" || c.Password == "" {
		errs = append(errs, errors.New("missing OIDC resource owner credentials"))
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

// NewOIDCPasswordGrantAccessTokenProvider builds an [AccessTokenProvider]
// using the OAuth2 resource owner password credentials grant. It calls
// [OIDCPasswordGrantAccessTokenProviderConfig.Validate] before proceeding.
//
// Deprecated: The OAuth2 security best current practice prohibits use of the
// password grant. Use it only when integrating with a legacy authorization
// server and no safer grant is available.
// See https://www.rfc-editor.org/rfc/rfc9700.html#section-2.4.
func NewOIDCPasswordGrantAccessTokenProvider(
	ctx context.Context,
	cfg OIDCPasswordGrantAccessTokenProviderConfig,
) (AccessTokenProvider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	tokenURL, err := resolveOIDCTokenURL(ctx, cfg.ProviderURL, cfg.TokenURL)
	if err != nil {
		return nil, err
	}

	// Select an explicit authentication style so oauth2 does not retry the
	// password request during auto-detection, which could send resource owner
	// credentials more than once. Confidential clients use HTTP Basic; public
	// clients send their client ID in the request parameters.
	authStyle := oauth2.AuthStyleInHeader
	if cfg.ClientSecret == "" {
		authStyle = oauth2.AuthStyleInParams
	}

	oauthConfig := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL:  tokenURL,
			AuthStyle: authStyle,
		},
		Scopes: cfg.Scopes,
	}

	return &oidcAccessTokenProvider{
		tokenRequest: func(ctx context.Context, previous *oauth2.Token) (*oauth2.Token, error) {
			if previous != nil && previous.RefreshToken != "" {
				// tokenNeedsRefresh has already determined that previous is within the
				// configured expiry leeway. oauth2.TokenSource performs its own validity
				// check, so it may otherwise return previous without refreshing it. Clear
				// AccessToken on a copy to force a refresh without mutating the cached token.
				expired := *previous
				expired.AccessToken = ""

				return oauthConfig.TokenSource(ctx, &expired).Token()
			}

			return oauthConfig.PasswordCredentialsToken(ctx, cfg.Username, cfg.Password)
		},
		tokenExpiryLeeway:       cfg.TokenExpiryLeeway,
		retryMaxAttempts:        cfg.RetryMaxAttempts,
		retryInitialInterval:    cfg.RetryInitialInterval,
		retryMaxInterval:        cfg.RetryMaxInterval,
		retryBackoffCoefficient: cfg.RetryBackoffCoefficient,
	}, nil
}
