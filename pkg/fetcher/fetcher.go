// Package fetcher retrieves GBFS feeds with optional authentication.
package fetcher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AuthType selects the authentication strategy.
type AuthType string

const (
	AuthNone                       AuthType = "none"
	AuthBasic                      AuthType = "basic_auth"
	AuthBearerToken                AuthType = "bearer_token"
	AuthOAuthClientCredentials     AuthType = "oauth_client_credentials_grant"
	AuthHeaders                    AuthType = "headers"
)

// AuthConfig configures authentication for feed requests.
type AuthConfig struct {
	Type                     AuthType          `json:"type"`
	BasicAuth                *BasicAuthConfig  `json:"basicAuth,omitempty"`
	BearerToken              *BearerTokenConfig `json:"bearerToken,omitempty"`
	OAuthClientCredentials   *OAuthConfig      `json:"oauthClientCredentialsGrant,omitempty"`
	Headers                  []HeaderConfig    `json:"headers,omitempty"`
}

// BasicAuthConfig holds username/password credentials.
type BasicAuthConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// BearerTokenConfig holds a bearer token.
type BearerTokenConfig struct {
	Token string `json:"token"`
}

// OAuthConfig holds client credentials config.
type OAuthConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	TokenURL string `json:"tokenUrl"`
}

// HeaderConfig defines a header to send with requests.
type HeaderConfig struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Fetcher wraps an HTTP client with auth support.
type Fetcher struct {
	client    *http.Client
	auth      *AuthConfig
	userAgent string
	token     string // Cached OAuth token
}

// Option mutates a Fetcher during construction.
type Option func(*Fetcher)

// WithAuth sets authentication config.
func WithAuth(auth *AuthConfig) Option {
	return func(f *Fetcher) {
		f.auth = auth
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(f *Fetcher) {
		f.client.Timeout = timeout
	}
}

// WithUserAgent sets the request user agent.
func WithUserAgent(ua string) Option {
	return func(f *Fetcher) {
		f.userAgent = ua
	}
}

// New constructs a Fetcher with options applied.
func New(opts ...Option) *Fetcher {
	f := &Fetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "GBFS-Validator-Go/1.0",
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// FetchResult captures a fetch outcome.
type FetchResult struct {
	URL        string
	Body       []byte
	StatusCode int
	Error      error
	Exists     bool
}

// Fetch retrieves a URL and returns the raw response body.
func (f *Fetcher) Fetch(ctx context.Context, targetURL string) *FetchResult {
	result := &FetchResult{URL: targetURL}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result
	}

	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "application/json")

	if err := f.applyAuth(ctx, req); err != nil {
		result.Error = fmt.Errorf("failed to apply authentication: %w", err)
		return result
	}

	resp, err := f.client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch URL: %w", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	if resp.StatusCode == http.StatusNotFound {
		result.Exists = false
		return result
	}

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return result
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("failed to read response body: %w", err)
		return result
	}

	result.Body = body
	result.Exists = true
	return result
}

// FetchJSON fetches a URL and unmarshals JSON into v.
func (f *Fetcher) FetchJSON(ctx context.Context, targetURL string, v interface{}) *FetchResult {
	result := f.Fetch(ctx, targetURL)
	if result.Error != nil || !result.Exists {
		return result
	}

	if err := json.Unmarshal(result.Body, v); err != nil {
		result.Error = fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return result
}

// applyAuth adds auth headers to a request.
func (f *Fetcher) applyAuth(ctx context.Context, req *http.Request) error {
	if f.auth == nil || f.auth.Type == AuthNone {
		return nil
	}

	switch f.auth.Type {
	case AuthBasic:
		if f.auth.BasicAuth != nil {
			credentials := base64.StdEncoding.EncodeToString(
				[]byte(f.auth.BasicAuth.User + ":" + f.auth.BasicAuth.Password),
			)
			req.Header.Set("Authorization", "Basic "+credentials)
		}

	case AuthBearerToken:
		if f.auth.BearerToken != nil {
			req.Header.Set("Authorization", "Bearer "+f.auth.BearerToken.Token)
		}

	case AuthOAuthClientCredentials:
		if f.auth.OAuthClientCredentials != nil {
			token, err := f.getOAuthToken(ctx)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+token)
		}

	case AuthHeaders:
		for _, h := range f.auth.Headers {
			if h.Key != "" && h.Value != "" {
				req.Header.Set(h.Key, h.Value)
			}
		}
	}

	return nil
}

// getOAuthToken fetches an OAuth token with client credentials.
func (f *Fetcher) getOAuthToken(ctx context.Context) (string, error) {
	if f.token != "" {
		return f.token, nil
	}

	cfg := f.auth.OAuthClientCredentials
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(cfg.User, cfg.Password)

	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	f.token = tokenResp.AccessToken
	return f.token, nil
}

// BuildFeedURL builds a feed URL from a base URL and feed name.
func BuildFeedURL(baseURL, feedName string) string {
	if strings.HasSuffix(baseURL, "/") {
		return baseURL + feedName + ".json"
	}

	if strings.HasSuffix(baseURL, "gbfs.json") {
		return strings.TrimSuffix(baseURL, "gbfs.json") + feedName + ".json"
	}

	return baseURL + "/" + feedName + ".json"
}
