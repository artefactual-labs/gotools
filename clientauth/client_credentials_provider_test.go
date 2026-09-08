package clientauth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"go.artefactual.dev/tools/clientauth"
)

type tokenResponse struct {
	statusCode  int
	accessToken string
	expiresIn   int
}

// tokenServer returns a test server that replies with each successive response.
// Callers must supply enough entries to cover all expected token attempts.
func tokenServer(t *testing.T, responses []tokenResponse) *httptest.Server {
	t.Helper()

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		res := responses[calls-1]
		if res.statusCode != http.StatusOK {
			w.WriteHeader(res.statusCode)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]any{
			"token_type":   "Bearer",
			"access_token": res.accessToken,
			"expires_in":   res.expiresIn,
		})
		assert.NilError(t, err)
	}))

	t.Cleanup(srv.Close)

	return srv
}

func TestOIDCAccessTokenProviderAccessToken(t *testing.T) {
	t.Parallel()

	type test struct {
		name       string
		responses  []tokenResponse
		calls      int
		wantTokens []string
		wantErr    string
	}

	for _, tc := range []test{
		{
			name: "Caches token without expiry",
			responses: []tokenResponse{
				{statusCode: http.StatusOK, accessToken: "token-1"},
			},
			calls:      2,
			wantTokens: []string{"token-1", "token-1"},
		},
		{
			name: "Refreshes token that is within leeway window",
			responses: []tokenResponse{
				{statusCode: http.StatusOK, accessToken: "token-1", expiresIn: 1},
				{statusCode: http.StatusOK, accessToken: "token-2", expiresIn: 3600},
			},
			calls:      2,
			wantTokens: []string{"token-1", "token-2"},
		},
		{
			name: "Retries transient token endpoint failures",
			responses: []tokenResponse{
				{statusCode: http.StatusBadGateway},
				{statusCode: http.StatusBadGateway},
				{statusCode: http.StatusOK, accessToken: "token-ok", expiresIn: 3600},
			},
			calls:      1,
			wantTokens: []string{"token-ok"},
		},
		{
			name: "Returns error if token endpoint returns non-retryable error",
			responses: []tokenResponse{
				{statusCode: http.StatusBadRequest},
				{statusCode: http.StatusBadRequest},
			},
			calls:   1,
			wantErr: "request OIDC token: oauth2: cannot fetch token: 400 Bad Request",
		},
		{
			name: "Returns error after retry attempts are exhausted",
			responses: []tokenResponse{
				{statusCode: http.StatusBadGateway},
				{statusCode: http.StatusBadGateway},
				{statusCode: http.StatusBadGateway},
				{statusCode: http.StatusBadGateway},
				{statusCode: http.StatusBadGateway},
				{statusCode: http.StatusBadGateway},
			},
			calls:   1,
			wantErr: "request OIDC token: oauth2: cannot fetch token: 502 Bad Gateway",
		},
		{
			name: "Returns error if token endpoint returns an empty token",
			responses: []tokenResponse{
				{statusCode: http.StatusOK, accessToken: ""},
				{statusCode: http.StatusOK, accessToken: ""},
				{statusCode: http.StatusOK, accessToken: ""},
				{statusCode: http.StatusOK, accessToken: ""},
				{statusCode: http.StatusOK, accessToken: ""},
				{statusCode: http.StatusOK, accessToken: ""},
			},
			calls:   1,
			wantErr: "request OIDC token: oauth2: server response missing access_token",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := tokenServer(t, tc.responses)

			cfg := clientauth.OIDCAccessTokenProviderConfig{
				TokenURL:                srv.URL,
				ClientID:                "client-id",
				ClientSecret:            "client-secret",
				RetryMaxAttempts:        3,
				RetryInitialInterval:    time.Microsecond,
				RetryMaxInterval:        time.Microsecond,
				RetryBackoffCoefficient: 1.0,
				TokenExpiryLeeway:       30 * time.Second,
			}

			provider, err := clientauth.NewOIDCAccessTokenProvider(t.Context(), cfg)
			assert.NilError(t, err)

			var tokens []string
			for range tc.calls {
				token, err := provider.AccessToken(t.Context())
				if tc.wantErr != "" {
					assert.ErrorContains(t, err, tc.wantErr)
					return
				}

				assert.NilError(t, err)
				tokens = append(tokens, token)
			}

			assert.DeepEqual(t, tokens, tc.wantTokens)
		})
	}
}

func TestOIDCAccessTokenProviderConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     clientauth.OIDCAccessTokenProviderConfig
		wantCfg clientauth.OIDCAccessTokenProviderConfig
		wantErr string
	}{
		{
			name: "Passes validation with provider URL and client credentials",
			cfg: clientauth.OIDCAccessTokenProviderConfig{
				ProviderURL:  "https://idp.example.com/realms/enduro",
				ClientID:     "enduro-worker",
				ClientSecret: "secret",
			},
			wantCfg: clientauth.OIDCAccessTokenProviderConfig{
				ProviderURL:             "https://idp.example.com/realms/enduro",
				ClientID:                "enduro-worker",
				ClientSecret:            "secret",
				TokenExpiryLeeway:       30 * time.Second,
				RetryMaxAttempts:        3,
				RetryInitialInterval:    500 * time.Millisecond,
				RetryMaxInterval:        2 * time.Second,
				RetryBackoffCoefficient: 2.0,
			},
		},
		{
			name: "Passes validation with token URL and client credentials",
			cfg: clientauth.OIDCAccessTokenProviderConfig{
				TokenURL:     "https://idp.example.com/token",
				ClientID:     "enduro-worker",
				ClientSecret: "secret",
			},
			wantCfg: clientauth.OIDCAccessTokenProviderConfig{
				TokenURL:                "https://idp.example.com/token",
				ClientID:                "enduro-worker",
				ClientSecret:            "secret",
				TokenExpiryLeeway:       30 * time.Second,
				RetryMaxAttempts:        3,
				RetryInitialInterval:    500 * time.Millisecond,
				RetryMaxInterval:        2 * time.Second,
				RetryBackoffCoefficient: 2.0,
			},
		},
		{
			name: "Preserves explicitly set values",
			cfg: clientauth.OIDCAccessTokenProviderConfig{
				ProviderURL:             "https://idp.example.com/realms/enduro",
				ClientID:                "enduro-worker",
				ClientSecret:            "secret",
				TokenExpiryLeeway:       10 * time.Second,
				RetryMaxAttempts:        5,
				RetryInitialInterval:    1 * time.Second,
				RetryMaxInterval:        10 * time.Second,
				RetryBackoffCoefficient: 3.0,
			},
			wantCfg: clientauth.OIDCAccessTokenProviderConfig{
				ProviderURL:             "https://idp.example.com/realms/enduro",
				ClientID:                "enduro-worker",
				ClientSecret:            "secret",
				TokenExpiryLeeway:       10 * time.Second,
				RetryMaxAttempts:        5,
				RetryInitialInterval:    1 * time.Second,
				RetryMaxInterval:        10 * time.Second,
				RetryBackoffCoefficient: 3.0,
			},
		},
		{
			name: "Fails validation when both providerURL and tokenURL are missing",
			cfg: clientauth.OIDCAccessTokenProviderConfig{
				ClientID:     "enduro-worker",
				ClientSecret: "secret",
			},
			wantErr: "missing OIDC providerURL or tokenURL",
		},
		{
			name: "Fails validation when client credentials are missing",
			cfg: clientauth.OIDCAccessTokenProviderConfig{
				ProviderURL: "https://idp.example.com/realms/enduro",
			},
			wantErr: "missing OIDC client credentials",
		},
		{
			name: "Fails validation with invalid retry attempts",
			cfg: clientauth.OIDCAccessTokenProviderConfig{
				ProviderURL:      "https://idp.example.com/realms/enduro",
				ClientID:         "enduro-worker",
				ClientSecret:     "secret",
				RetryMaxAttempts: -1,
			},
			wantErr: "invalid OIDC retry max attempts, value must be >= 1",
		},
		{
			name: "Fails validation with invalid retry backoff coefficient",
			cfg: clientauth.OIDCAccessTokenProviderConfig{
				ProviderURL:             "https://idp.example.com/realms/enduro",
				ClientID:                "enduro-worker",
				ClientSecret:            "secret",
				RetryBackoffCoefficient: 0.5,
				RetryMaxAttempts:        3,
				RetryInitialInterval:    1 * time.Millisecond,
				RetryMaxInterval:        2 * time.Millisecond,
			},
			wantErr: "invalid OIDC retry backoff coefficient, value must be >= 1",
		},
		{
			name: "Joins multiple validation errors",
			cfg: clientauth.OIDCAccessTokenProviderConfig{
				RetryMaxAttempts:        -1,
				RetryBackoffCoefficient: 0.5,
			},
			wantErr: "missing OIDC providerURL or tokenURL\nmissing OIDC client credentials\ninvalid OIDC retry max attempts, value must be >= 1\ninvalid OIDC retry backoff coefficient, value must be >= 1",
		},
		{
			name: "Fails validation with invalid duration values",
			cfg: clientauth.OIDCAccessTokenProviderConfig{
				ProviderURL:          "https://idp.example.com/realms/enduro",
				ClientID:             "enduro-worker",
				ClientSecret:         "secret",
				TokenExpiryLeeway:    -1 * time.Second,
				RetryInitialInterval: -1 * time.Millisecond,
			},
			wantErr: "invalid OIDC duration configuration, values must be > 0",
		},
		{
			name: "Fails validation with max interval lower than initial interval",
			cfg: clientauth.OIDCAccessTokenProviderConfig{
				ProviderURL:          "https://idp.example.com/realms/enduro",
				ClientID:             "enduro-worker",
				ClientSecret:         "secret",
				RetryInitialInterval: 2 * time.Millisecond,
				RetryMaxInterval:     1 * time.Millisecond,
			},
			wantErr: "invalid OIDC retry interval configuration, max interval must be >= initial interval",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			if tc.wantErr != "" {
				assert.Error(t, err, tc.wantErr)
				return
			}

			assert.NilError(t, err)
			assert.DeepEqual(t, tc.cfg, tc.wantCfg)
		})
	}
}
