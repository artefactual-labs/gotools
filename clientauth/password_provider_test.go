package clientauth_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"

	"go.artefactual.dev/tools/clientauth"
)

type passwordTokenResponse struct {
	statusCode       int
	accessToken      string
	refreshToken     string
	expiresIn        int
	errorCode        string
	errorDescription string
	wantRequest      *passwordTokenRequest
	requestStarted   chan<- struct{}
	releaseRequest   <-chan struct{}
}

type passwordTokenRequest struct {
	basicAuth       bool
	clientID        string
	clientSecret    string
	clientSecretSet bool
	grantType       string
	refreshToken    string
	username        string
	password        string
	scope           string
}

type passwordTokenResult struct {
	token            string
	statusCode       int
	errorCode        string
	errorDescription string
}

// passwordTokenServer returns a test server that replies with each successive response.
// It also serves OIDC discovery metadata. Callers must supply enough responses
// to cover all expected token attempts.
func passwordTokenServer(
	t *testing.T,
	responses []passwordTokenResponse,
) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			assert.NilError(t, json.NewEncoder(w).Encode(map[string]any{
				"issuer":                                srv.URL,
				"authorization_endpoint":                srv.URL + "/authorize",
				"token_endpoint":                        srv.URL + "/token",
				"jwks_uri":                              srv.URL + "/keys",
				"id_token_signing_alg_values_supported": []string{"RS256"},
			}))

			return
		}

		call := calls.Add(1)
		if call > int32(len(responses)) {
			t.Errorf("unexpected token request %d", call)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		res := responses[call-1]
		assert.Equal(t, r.Method, http.MethodPost)
		assert.Equal(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		assert.NilError(t, r.ParseForm())
		clientID, clientSecret, basicAuth := r.BasicAuth()
		clientSecretSet := basicAuth
		if !basicAuth {
			clientID = r.Form.Get("client_id")
			clientSecret = r.Form.Get("client_secret")
			_, clientSecretSet = r.Form["client_secret"]
		}
		gotRequest := passwordTokenRequest{
			basicAuth:       basicAuth,
			clientID:        clientID,
			clientSecret:    clientSecret,
			clientSecretSet: clientSecretSet,
			grantType:       r.Form.Get("grant_type"),
			refreshToken:    r.Form.Get("refresh_token"),
			username:        r.Form.Get("username"),
			password:        r.Form.Get("password"),
			scope:           r.Form.Get("scope"),
		}
		if res.wantRequest != nil {
			assert.Equal(t, gotRequest, *res.wantRequest)
		}
		if res.requestStarted != nil {
			close(res.requestStarted)
		}
		if res.releaseRequest != nil {
			<-res.releaseRequest
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(res.statusCode)
		if res.errorCode != "" {
			assert.NilError(t, json.NewEncoder(w).Encode(map[string]any{
				"error":             res.errorCode,
				"error_description": res.errorDescription,
			}))

			return
		}
		if res.statusCode != http.StatusOK {
			return
		}

		body := map[string]any{
			"token_type":   "Bearer",
			"access_token": res.accessToken,
			"expires_in":   res.expiresIn,
		}
		if res.refreshToken != "" {
			body["refresh_token"] = res.refreshToken
		}
		assert.NilError(t, json.NewEncoder(w).Encode(body))
	}))

	t.Cleanup(srv.Close)

	return srv, &calls
}

func assertPasswordTokenResult(
	t *testing.T,
	token string,
	err error,
	want passwordTokenResult,
) {
	t.Helper()

	if want.errorCode == "" {
		assert.NilError(t, err)
		assert.Equal(t, token, want.token)

		return
	}

	var retrieveErr *oauth2.RetrieveError
	assert.Assert(t, errors.As(err, &retrieveErr))
	assert.Equal(t, retrieveErr.Response.StatusCode, want.statusCode)
	assert.Equal(t, retrieveErr.ErrorCode, want.errorCode)
	assert.Equal(t, retrieveErr.ErrorDescription, want.errorDescription)
}

func TestOIDCPasswordGrantAccessTokenProviderAccessToken(t *testing.T) {
	t.Parallel()

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	confidentialPasswordRequest := passwordTokenRequest{
		basicAuth:       true,
		clientID:        "enduro-worker",
		clientSecret:    "secret",
		clientSecretSet: true,
		grantType:       "password",
		username:        "service-user",
		password:        "password",
	}
	scopedPasswordRequest := confidentialPasswordRequest
	scopedPasswordRequest.scope = "openid profile"
	publicPasswordRequest := passwordTokenRequest{
		clientID:  "public-client",
		grantType: "password",
		username:  "service-user",
		password:  "password",
	}
	refreshRequest := passwordTokenRequest{
		basicAuth:       true,
		clientID:        "enduro-worker",
		clientSecret:    "secret",
		clientSecretSet: true,
		grantType:       "refresh_token",
		refreshToken:    "refresh-1",
	}
	refreshRejected := passwordTokenResponse{
		statusCode:       http.StatusBadRequest,
		errorCode:        "invalid_grant",
		errorDescription: "refresh token rejected: refresh-1",
		wantRequest:      &refreshRequest,
	}
	refreshError := passwordTokenResult{
		statusCode:       http.StatusBadRequest,
		errorCode:        "invalid_grant",
		errorDescription: "refresh token rejected: refresh-1",
	}

	type test struct {
		name            string
		responses       []passwordTokenResponse
		wantResults     []passwordTokenResult
		clientID        string
		publicClient    bool
		providerURL     bool
		scopes          []string
		concurrentCalls int
		requestStarted  <-chan struct{}
		releaseRequest  chan struct{}
	}

	for _, tc := range []test{
		{
			name: "Requests password grant token",
			responses: []passwordTokenResponse{
				{
					statusCode:  http.StatusOK,
					accessToken: "token-1",
					expiresIn:   3600,
					wantRequest: &scopedPasswordRequest,
				},
			},
			wantResults: []passwordTokenResult{{token: "token-1"}},
			scopes:      []string{"openid", "profile"},
		},
		{
			name: "Uses form auth for public client",
			responses: []passwordTokenResponse{
				{
					statusCode:  http.StatusOK,
					accessToken: "token-1",
					expiresIn:   3600,
					wantRequest: &publicPasswordRequest,
				},
			},
			wantResults:  []passwordTokenResult{{token: "token-1"}},
			clientID:     "public-client",
			publicClient: true,
		},
		{
			name: "Discovers token endpoint",
			responses: []passwordTokenResponse{
				{statusCode: http.StatusOK, accessToken: "token-1", expiresIn: 3600},
			},
			wantResults:  []passwordTokenResult{{token: "token-1"}},
			publicClient: true,
			providerURL:  true,
		},
		{
			name: "Refreshes without resending password",
			responses: []passwordTokenResponse{
				{
					statusCode:   http.StatusOK,
					accessToken:  "token-1",
					refreshToken: "refresh-1",
					expiresIn:    1,
					wantRequest:  &confidentialPasswordRequest,
				},
				{
					statusCode:  http.StatusOK,
					accessToken: "token-2",
					expiresIn:   1,
					wantRequest: &refreshRequest,
				},
				{
					statusCode:  http.StatusOK,
					accessToken: "token-3",
					expiresIn:   3600,
					wantRequest: &refreshRequest,
				},
			},
			wantResults: []passwordTokenResult{
				{token: "token-1"},
				{token: "token-2"},
				{token: "token-3"},
			},
		},
		{
			name: "Does not fall back from rejected refresh token",
			responses: []passwordTokenResponse{
				{
					statusCode:   http.StatusOK,
					accessToken:  "token-1",
					refreshToken: "refresh-1",
					expiresIn:    1,
					wantRequest:  &confidentialPasswordRequest,
				},
				refreshRejected,
				refreshRejected,
			},
			wantResults: []passwordTokenResult{
				{token: "token-1"},
				refreshError,
				refreshError,
			},
		},
		{
			name: "Returns retrieve error",
			responses: []passwordTokenResponse{
				{
					statusCode:       http.StatusBadRequest,
					errorCode:        "invalid_grant",
					errorDescription: "token request rejected",
				},
			},
			wantResults: []passwordTokenResult{
				{
					statusCode:       http.StatusBadRequest,
					errorCode:        "invalid_grant",
					errorDescription: "token request rejected",
				},
			},
		},
		{
			name: "Retries server errors",
			responses: []passwordTokenResponse{
				{
					statusCode:       http.StatusBadGateway,
					errorCode:        "server-specific-error",
					errorDescription: "unsafe server detail",
				},
				{statusCode: http.StatusOK, accessToken: "token-1", expiresIn: 3600},
			},
			wantResults: []passwordTokenResult{{token: "token-1"}},
		},
		{
			name: "Coalesces concurrent token requests",
			responses: []passwordTokenResponse{
				{
					statusCode:     http.StatusOK,
					accessToken:    "token-1",
					expiresIn:      3600,
					requestStarted: requestStarted,
					releaseRequest: releaseRequest,
				},
			},
			wantResults:     []passwordTokenResult{{token: "token-1"}},
			publicClient:    true,
			concurrentCalls: 16,
			requestStarted:  requestStarted,
			releaseRequest:  releaseRequest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv, requests := passwordTokenServer(t, tc.responses)
			cfg := clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				TokenURL:                srv.URL,
				ClientID:                "enduro-worker",
				ClientSecret:            "secret",
				Username:                "service-user",
				Password:                "password",
				Scopes:                  tc.scopes,
				TokenExpiryLeeway:       30 * time.Second,
				RetryMaxAttempts:        3,
				RetryInitialInterval:    time.Microsecond,
				RetryMaxInterval:        time.Microsecond,
				RetryBackoffCoefficient: 1.0,
			}
			if tc.clientID != "" {
				cfg.ClientID = tc.clientID
			}
			if tc.publicClient {
				cfg.ClientSecret = ""
			}
			if tc.providerURL {
				cfg.ProviderURL = srv.URL
				cfg.TokenURL = ""
			}

			provider, err := clientauth.NewOIDCPasswordGrantAccessTokenProvider(t.Context(), cfg)
			assert.NilError(t, err)

			if tc.concurrentCalls == 0 {
				for _, want := range tc.wantResults {
					token, err := provider.AccessToken(t.Context())
					assertPasswordTokenResult(t, token, err, want)
				}
			} else {
				results := make(chan struct {
					token string
					err   error
				}, tc.concurrentCalls)
				for range tc.concurrentCalls {
					go func() {
						token, err := provider.AccessToken(t.Context())
						results <- struct {
							token string
							err   error
						}{token: token, err: err}
					}()
				}

				select {
				case <-tc.requestStarted:
				case <-time.After(time.Second):
					close(tc.releaseRequest)
					t.Fatal("timed out waiting for token request")
				}
				close(tc.releaseRequest)

				for range tc.concurrentCalls {
					result := <-results
					assertPasswordTokenResult(t, result.token, result.err, tc.wantResults[0])
				}
			}

			assert.Equal(t, requests.Load(), int32(len(tc.responses)))
		})
	}
}

func TestOIDCPasswordGrantAccessTokenProviderConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     clientauth.OIDCPasswordGrantAccessTokenProviderConfig
		wantCfg clientauth.OIDCPasswordGrantAccessTokenProviderConfig
		wantErr string
	}{
		{
			name: "Passes validation with provider URL and resource owner credentials",
			cfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				ProviderURL: "https://idp.example.com/realms/enduro",
				ClientID:    "enduro-worker",
				Username:    "service-user",
				Password:    "password",
			},
			wantCfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				ProviderURL:             "https://idp.example.com/realms/enduro",
				ClientID:                "enduro-worker",
				Username:                "service-user",
				Password:                "password",
				TokenExpiryLeeway:       30 * time.Second,
				RetryMaxAttempts:        3,
				RetryInitialInterval:    500 * time.Millisecond,
				RetryMaxInterval:        2 * time.Second,
				RetryBackoffCoefficient: 2.0,
			},
		},
		{
			name: "Passes validation with token URL and resource owner credentials",
			cfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				TokenURL: "https://idp.example.com/token",
				ClientID: "enduro-worker",
				Username: "service-user",
				Password: "password",
			},
			wantCfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				TokenURL:                "https://idp.example.com/token",
				ClientID:                "enduro-worker",
				Username:                "service-user",
				Password:                "password",
				TokenExpiryLeeway:       30 * time.Second,
				RetryMaxAttempts:        3,
				RetryInitialInterval:    500 * time.Millisecond,
				RetryMaxInterval:        2 * time.Second,
				RetryBackoffCoefficient: 2.0,
			},
		},
		{
			name: "Preserves explicitly set values",
			cfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				ProviderURL:             "https://idp.example.com/realms/enduro",
				ClientID:                "enduro-worker",
				ClientSecret:            "secret",
				Username:                "service-user",
				Password:                "password",
				Scopes:                  []string{"openid", "profile"},
				TokenExpiryLeeway:       10 * time.Second,
				RetryMaxAttempts:        5,
				RetryInitialInterval:    time.Second,
				RetryMaxInterval:        10 * time.Second,
				RetryBackoffCoefficient: 3.0,
			},
			wantCfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				ProviderURL:             "https://idp.example.com/realms/enduro",
				ClientID:                "enduro-worker",
				ClientSecret:            "secret",
				Username:                "service-user",
				Password:                "password",
				Scopes:                  []string{"openid", "profile"},
				TokenExpiryLeeway:       10 * time.Second,
				RetryMaxAttempts:        5,
				RetryInitialInterval:    time.Second,
				RetryMaxInterval:        10 * time.Second,
				RetryBackoffCoefficient: 3.0,
			},
		},
		{
			name: "Fails validation when both providerURL and tokenURL are missing",
			cfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				ClientID: "enduro-worker",
				Username: "service-user",
				Password: "password",
			},
			wantErr: "missing OIDC providerURL or tokenURL",
		},
		{
			name: "Fails validation when client ID is missing",
			cfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				ProviderURL: "https://idp.example.com/realms/enduro",
				Username:    "service-user",
				Password:    "password",
			},
			wantErr: "missing OIDC client ID",
		},
		{
			name: "Fails validation when username is missing",
			cfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				ProviderURL: "https://idp.example.com/realms/enduro",
				ClientID:    "enduro-worker",
				Password:    "password",
			},
			wantErr: "missing OIDC resource owner credentials",
		},
		{
			name: "Fails validation when password is missing",
			cfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				ProviderURL: "https://idp.example.com/realms/enduro",
				ClientID:    "enduro-worker",
				Username:    "service-user",
			},
			wantErr: "missing OIDC resource owner credentials",
		},
		{
			name: "Fails validation with invalid retry attempts",
			cfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				ProviderURL:      "https://idp.example.com/realms/enduro",
				ClientID:         "enduro-worker",
				Username:         "service-user",
				Password:         "password",
				RetryMaxAttempts: -1,
			},
			wantErr: "invalid OIDC retry max attempts, value must be >= 1",
		},
		{
			name: "Fails validation with invalid retry backoff coefficient",
			cfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				ProviderURL:             "https://idp.example.com/realms/enduro",
				ClientID:                "enduro-worker",
				Username:                "service-user",
				Password:                "password",
				RetryBackoffCoefficient: 0.5,
				RetryMaxAttempts:        3,
				RetryInitialInterval:    time.Millisecond,
				RetryMaxInterval:        2 * time.Millisecond,
			},
			wantErr: "invalid OIDC retry backoff coefficient, value must be >= 1",
		},
		{
			name: "Joins multiple validation errors",
			cfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				RetryMaxAttempts:        -1,
				RetryBackoffCoefficient: 0.5,
			},
			wantErr: "missing OIDC providerURL or tokenURL\nmissing OIDC client ID\nmissing OIDC resource owner credentials\ninvalid OIDC retry max attempts, value must be >= 1\ninvalid OIDC retry backoff coefficient, value must be >= 1",
		},
		{
			name: "Fails validation with invalid duration values",
			cfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				ProviderURL:          "https://idp.example.com/realms/enduro",
				ClientID:             "enduro-worker",
				Username:             "service-user",
				Password:             "password",
				TokenExpiryLeeway:    -time.Second,
				RetryInitialInterval: -time.Millisecond,
			},
			wantErr: "invalid OIDC duration configuration, values must be > 0",
		},
		{
			name: "Fails validation with max interval lower than initial interval",
			cfg: clientauth.OIDCPasswordGrantAccessTokenProviderConfig{
				ProviderURL:          "https://idp.example.com/realms/enduro",
				ClientID:             "enduro-worker",
				Username:             "service-user",
				Password:             "password",
				RetryInitialInterval: 2 * time.Millisecond,
				RetryMaxInterval:     time.Millisecond,
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
