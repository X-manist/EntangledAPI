package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

type githubCopilotOAuthSettingsStub struct {
	clientID string
	err      error
}

type githubCopilotOAuthProxyRepo struct {
	ProxyRepository
	getByID func(context.Context, int64) (*Proxy, error)
}

func (r *githubCopilotOAuthProxyRepo) GetByID(ctx context.Context, id int64) (*Proxy, error) {
	return r.getByID(ctx, id)
}

func (s *githubCopilotOAuthSettingsStub) GetGitHubCopilotOAuthClientID(context.Context) (string, error) {
	return s.clientID, s.err
}

type githubCopilotOAuthRequestCapture struct {
	Method        string
	URL           string
	Body          string
	Authorization string
	ProxyURL      string
	Accept        string
	ContentType   string
}

type githubCopilotOAuthHTTPStub struct {
	mu      sync.Mutex
	calls   []githubCopilotOAuthRequestCapture
	handler func(req *http.Request, body, proxyURL string) (*http.Response, error)
}

func (s *githubCopilotOAuthHTTPStub) Do(req *http.Request, proxyURL string, _ int64, _ int) (*http.Response, error) {
	body := ""
	if req.Body != nil {
		encoded, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		body = string(encoded)
		req.Body = io.NopCloser(strings.NewReader(body))
	}
	s.mu.Lock()
	s.calls = append(s.calls, githubCopilotOAuthRequestCapture{
		Method:        req.Method,
		URL:           req.URL.String(),
		Body:          body,
		Authorization: req.Header.Get("Authorization"),
		ProxyURL:      proxyURL,
		Accept:        req.Header.Get("Accept"),
		ContentType:   req.Header.Get("Content-Type"),
	})
	s.mu.Unlock()
	return s.handler(req, body, proxyURL)
}

func (s *githubCopilotOAuthHTTPStub) DoWithTLS(
	req *http.Request,
	proxyURL string,
	accountID int64,
	concurrency int,
	_ *tlsfingerprint.Profile,
) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, concurrency)
}

func (s *githubCopilotOAuthHTTPStub) captured() []githubCopilotOAuthRequestCapture {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]githubCopilotOAuthRequestCapture(nil), s.calls...)
}

func githubCopilotOAuthJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func githubCopilotDeviceStartJSON() string {
	return `{
		"device_code":"server-only-device-code",
		"user_code":"ABCD-EFGH",
		"verification_uri":"https://github.com/login/device",
		"verification_uri_complete":"https://github.com/login/device?user_code=ABCD-EFGH",
		"expires_in":900,
		"interval":5
	}`
}

func githubCopilotEntitlementJSON() string {
	return `{"login":"octocat","access_type_sku":"copilot_for_individual_user","chat_enabled":true,"copilot_plan":"individual","endpoints":{"api":"https://api.githubcopilot.com"}}`
}

func newGitHubCopilotOAuthTestService(
	upstream HTTPUpstream,
	proxyRepo ProxyRepository,
) (*GitHubCopilotOAuthService, *time.Time) {
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	service := NewGitHubCopilotOAuthService(
		&githubCopilotOAuthSettingsStub{clientID: "owned-oauth-app-client-id"},
		proxyRepo,
		upstream,
	)
	service.now = func() time.Time { return now }
	service.randomID = func() (string, error) { return "opaque-oauth-session", nil }
	return service, &now
}

func TestGitHubCopilotOAuthCapabilities(t *testing.T) {
	t.Parallel()

	configured := NewGitHubCopilotOAuthService(
		&githubCopilotOAuthSettingsStub{clientID: " client-id "},
		nil,
		nil,
	)
	capabilities, err := configured.Capabilities(context.Background())
	require.NoError(t, err)
	require.True(t, capabilities.Configured)
	require.Equal(t, GitHubCopilotPATURL, capabilities.PATURL)

	unconfigured := NewGitHubCopilotOAuthService(&githubCopilotOAuthSettingsStub{}, nil, nil)
	capabilities, err = unconfigured.Capabilities(context.Background())
	require.NoError(t, err)
	require.False(t, capabilities.Configured)
	require.NotEmpty(t, capabilities.PATURL)
}

func TestGitHubCopilotOAuthStartKeepsDeviceCodeServerSide(t *testing.T) {
	t.Parallel()

	upstream := &githubCopilotOAuthHTTPStub{handler: func(req *http.Request, _ string, _ string) (*http.Response, error) {
		require.Equal(t, githubCopilotDeviceCodeURL, req.URL.String())
		return githubCopilotOAuthJSONResponse(http.StatusOK, githubCopilotDeviceStartJSON()), nil
	}}
	service, _ := newGitHubCopilotOAuthTestService(upstream, nil)

	result, err := service.Start(context.Background(), 42, nil)
	require.NoError(t, err)
	require.Equal(t, "opaque-oauth-session", result.OAuthSessionID)
	require.Equal(t, "ABCD-EFGH", result.UserCode)
	require.Equal(t, int64(900), result.ExpiresIn)
	require.Equal(t, int64(5), result.Interval)

	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "server-only-device-code")
	require.NotContains(t, string(encoded), "client-id")

	service.mu.Lock()
	stored := service.sessions[result.OAuthSessionID]
	require.Equal(t, "server-only-device-code", stored.deviceCode)
	service.mu.Unlock()

	calls := upstream.captured()
	require.Len(t, calls, 1)
	require.Equal(t, http.MethodPost, calls[0].Method)
	require.Equal(t, "application/json", calls[0].Accept)
	require.Equal(t, "application/x-www-form-urlencoded", calls[0].ContentType)
	form, err := url.ParseQuery(calls[0].Body)
	require.NoError(t, err)
	require.Equal(t, "owned-oauth-app-client-id", form.Get("client_id"))
	require.Equal(t, "read:user", form.Get("scope"))
}

func TestGitHubCopilotOAuthAuthorizeCheckoutReleaseFinalize(t *testing.T) {
	proxyID := int64(9)
	proxyRepo := &githubCopilotOAuthProxyRepo{getByID: func(_ context.Context, id int64) (*Proxy, error) {
		require.Equal(t, proxyID, id)
		return &Proxy{ID: id, Protocol: "http", Host: "proxy.example", Port: 8080}, nil
	}}
	upstream := &githubCopilotOAuthHTTPStub{handler: func(req *http.Request, _ string, _ string) (*http.Response, error) {
		switch req.URL.String() {
		case githubCopilotDeviceCodeURL:
			return githubCopilotOAuthJSONResponse(http.StatusOK, githubCopilotDeviceStartJSON()), nil
		case githubCopilotAccessTokenURL:
			return githubCopilotOAuthJSONResponse(http.StatusOK, `{"access_token":"gho-source-token","token_type":"bearer"}`), nil
		case githubCopilotUserURL:
			return githubCopilotOAuthJSONResponse(http.StatusOK, `{"login":"octocat","id":583231}`), nil
		case githubCopilotUserDiscoveryURL:
			return githubCopilotOAuthJSONResponse(http.StatusOK, githubCopilotEntitlementJSON()), nil
		default:
			return nil, fmt.Errorf("unexpected URL")
		}
	}}
	service, now := newGitHubCopilotOAuthTestService(upstream, proxyRepo)
	started, err := service.Start(context.Background(), 42, &proxyID)
	require.NoError(t, err)

	_, err = service.Poll(context.Background(), 7, started.OAuthSessionID)
	require.ErrorIs(t, err, ErrGitHubCopilotOAuthSessionForbidden)

	*now = now.Add(5 * time.Second)
	poll, err := service.Poll(context.Background(), 42, started.OAuthSessionID)
	require.NoError(t, err)
	require.Equal(t, GitHubCopilotOAuthStatusAuthorized, poll.Status)
	require.Equal(t, "octocat", poll.GitHubLogin)
	require.Equal(t, int64(583231), poll.GitHubUserID)
	encoded, err := json.Marshal(poll)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "gho-source-token")

	calls := upstream.captured()
	require.Len(t, calls, 4)
	for _, call := range calls {
		require.Equal(t, "http://proxy.example:8080", call.ProxyURL)
	}
	require.Equal(t, "Bearer gho-source-token", calls[2].Authorization)
	require.Equal(t, "Bearer gho-source-token", calls[3].Authorization)

	type checkoutResult struct {
		credential *GitHubCopilotOAuthCredential
		err        error
	}
	results := make(chan checkoutResult, 2)
	for range 2 {
		go func() {
			credential, checkoutErr := service.CheckoutCredential(42, started.OAuthSessionID)
			results <- checkoutResult{credential: credential, err: checkoutErr}
		}()
	}
	var checkedOut *GitHubCopilotOAuthCredential
	var conflictCount int
	for range 2 {
		result := <-results
		if result.err == nil {
			checkedOut = result.credential
			continue
		}
		require.ErrorIs(t, result.err, ErrGitHubCopilotOAuthCredentialInUse)
		conflictCount++
	}
	require.NotNil(t, checkedOut)
	require.Equal(t, "gho-source-token", checkedOut.AccessToken)
	require.Equal(t, 1, conflictCount)
	credentialJSON, err := json.Marshal(checkedOut)
	require.NoError(t, err)
	require.NotContains(t, string(credentialJSON), "gho-source-token")

	require.NoError(t, service.ReleaseCredential(42, started.OAuthSessionID))
	checkedOut, err = service.CheckoutCredential(42, started.OAuthSessionID)
	require.NoError(t, err)
	require.Equal(t, "octocat", checkedOut.GitHubLogin)
	require.NoError(t, service.FinalizeCredential(42, started.OAuthSessionID))
	_, err = service.CheckoutCredential(42, started.OAuthSessionID)
	require.ErrorIs(t, err, ErrGitHubCopilotOAuthSessionNotFound)
}

func TestGitHubCopilotOAuthPollThrottleAndSlowDown(t *testing.T) {
	tokenPolls := 0
	upstream := &githubCopilotOAuthHTTPStub{handler: func(req *http.Request, _ string, _ string) (*http.Response, error) {
		if req.URL.String() == githubCopilotDeviceCodeURL {
			return githubCopilotOAuthJSONResponse(http.StatusOK, githubCopilotDeviceStartJSON()), nil
		}
		require.Equal(t, githubCopilotAccessTokenURL, req.URL.String())
		tokenPolls++
		if tokenPolls == 1 {
			return githubCopilotOAuthJSONResponse(http.StatusOK, `{"error":"slow_down"}`), nil
		}
		return githubCopilotOAuthJSONResponse(http.StatusOK, `{"error":"authorization_pending"}`), nil
	}}
	service, now := newGitHubCopilotOAuthTestService(upstream, nil)
	started, err := service.Start(context.Background(), 42, nil)
	require.NoError(t, err)

	poll, err := service.Poll(context.Background(), 42, started.OAuthSessionID)
	require.NoError(t, err)
	require.Equal(t, GitHubCopilotOAuthStatusPending, poll.Status)
	require.Equal(t, 0, tokenPolls)

	*now = now.Add(5 * time.Second)
	poll, err = service.Poll(context.Background(), 42, started.OAuthSessionID)
	require.NoError(t, err)
	require.Equal(t, GitHubCopilotOAuthStatusSlowDown, poll.Status)
	require.Equal(t, int64(10), poll.Interval)
	require.Equal(t, 1, tokenPolls)

	*now = now.Add(5 * time.Second)
	poll, err = service.Poll(context.Background(), 42, started.OAuthSessionID)
	require.NoError(t, err)
	require.Equal(t, GitHubCopilotOAuthStatusPending, poll.Status)
	require.Equal(t, 1, tokenPolls)

	*now = now.Add(5 * time.Second)
	poll, err = service.Poll(context.Background(), 42, started.OAuthSessionID)
	require.NoError(t, err)
	require.Equal(t, GitHubCopilotOAuthStatusPending, poll.Status)
	require.Equal(t, 2, tokenPolls)
}

func TestGitHubCopilotOAuthEntitlementFailureIsTerminalAndRedacted(t *testing.T) {
	upstream := &githubCopilotOAuthHTTPStub{handler: func(req *http.Request, _ string, _ string) (*http.Response, error) {
		switch req.URL.String() {
		case githubCopilotDeviceCodeURL:
			return githubCopilotOAuthJSONResponse(http.StatusOK, githubCopilotDeviceStartJSON()), nil
		case githubCopilotAccessTokenURL:
			return githubCopilotOAuthJSONResponse(http.StatusOK, `{"access_token":"gho-sensitive-source"}`), nil
		case githubCopilotUserURL:
			return githubCopilotOAuthJSONResponse(http.StatusOK, `{"login":"no-plan","id":99}`), nil
		case githubCopilotUserDiscoveryURL:
			return githubCopilotOAuthJSONResponse(http.StatusForbidden, `{"message":"gho-sensitive-source has no entitlement"}`), nil
		default:
			return nil, fmt.Errorf("unexpected URL")
		}
	}}
	service, now := newGitHubCopilotOAuthTestService(upstream, nil)
	started, err := service.Start(context.Background(), 42, nil)
	require.NoError(t, err)
	*now = now.Add(5 * time.Second)

	_, err = service.Poll(context.Background(), 42, started.OAuthSessionID)
	require.ErrorIs(t, err, ErrGitHubCopilotOAuthEntitlementRequired)
	require.NotContains(t, err.Error(), "gho-sensitive-source")

	service.mu.Lock()
	stored := service.sessions[started.OAuthSessionID]
	require.Equal(t, GitHubCopilotOAuthStatusError, stored.status)
	require.Empty(t, stored.accessToken)
	require.Empty(t, stored.deviceCode)
	service.mu.Unlock()

	before := len(upstream.captured())
	poll, err := service.Poll(context.Background(), 42, started.OAuthSessionID)
	require.NoError(t, err)
	require.Equal(t, GitHubCopilotOAuthStatusError, poll.Status)
	require.Len(t, upstream.captured(), before)
}

func TestGitHubCopilotOAuthEntitlementRateLimitKeepsAuthorizedTokenForRetry(t *testing.T) {
	entitlementCalls := 0
	upstream := &githubCopilotOAuthHTTPStub{handler: func(req *http.Request, _ string, _ string) (*http.Response, error) {
		switch req.URL.String() {
		case githubCopilotDeviceCodeURL:
			return githubCopilotOAuthJSONResponse(http.StatusOK, githubCopilotDeviceStartJSON()), nil
		case githubCopilotAccessTokenURL:
			return githubCopilotOAuthJSONResponse(http.StatusOK, `{"access_token":"gho-sensitive-source"}`), nil
		case githubCopilotUserURL:
			return githubCopilotOAuthJSONResponse(http.StatusOK, `{"login":"paid-user","id":99}`), nil
		case githubCopilotUserDiscoveryURL:
			entitlementCalls++
			if entitlementCalls == 1 {
				return githubCopilotOAuthJSONResponse(
					http.StatusForbidden,
					`{"message":"You have exceeded a secondary rate limit. Please wait a few minutes before you try again."}`,
				), nil
			}
			return githubCopilotOAuthJSONResponse(http.StatusOK, githubCopilotEntitlementJSON()), nil
		default:
			return nil, fmt.Errorf("unexpected URL")
		}
	}}
	service, now := newGitHubCopilotOAuthTestService(upstream, nil)
	started, err := service.Start(context.Background(), 42, nil)
	require.NoError(t, err)
	*now = now.Add(5 * time.Second)

	_, err = service.Poll(context.Background(), 42, started.OAuthSessionID)
	require.ErrorIs(t, err, ErrGitHubCopilotOAuthUnavailable)
	service.mu.Lock()
	stored := service.sessions[started.OAuthSessionID]
	require.Equal(t, GitHubCopilotOAuthStatusPending, stored.status)
	require.Equal(t, "gho-sensitive-source", stored.accessToken)
	require.Empty(t, stored.deviceCode)
	service.mu.Unlock()

	*now = now.Add(5 * time.Second)
	poll, err := service.Poll(context.Background(), 42, started.OAuthSessionID)
	require.NoError(t, err)
	require.Equal(t, GitHubCopilotOAuthStatusAuthorized, poll.Status)
	require.Equal(t, 2, entitlementCalls)
}

func TestIsGitHubRateLimitResponse(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name    string
		headers http.Header
		body    string
		want    bool
	}{
		{
			name:    "retry after header",
			headers: http.Header{"Retry-After": []string{"60"}},
			want:    true,
		},
		{
			name:    "primary limit remaining header",
			headers: http.Header{"X-Ratelimit-Remaining": []string{"0"}},
			want:    true,
		},
		{
			name: "secondary limit without headers",
			body: `{"message":"You have exceeded a secondary rate limit."}`,
			want: true,
		},
		{
			name: "entitlement failure",
			body: `{"message":"the GitHub account has no Copilot entitlement"}`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.want, isGitHubRateLimitResponse(testCase.headers, []byte(testCase.body)))
		})
	}
}

func TestGitHubCopilotOAuthTransportErrorDoesNotLeakDeviceCode(t *testing.T) {
	upstream := &githubCopilotOAuthHTTPStub{handler: func(req *http.Request, body string, _ string) (*http.Response, error) {
		if req.URL.String() == githubCopilotDeviceCodeURL {
			return githubCopilotOAuthJSONResponse(http.StatusOK, githubCopilotDeviceStartJSON()), nil
		}
		return nil, fmt.Errorf("transport dumped request body: %s", body)
	}}
	service, now := newGitHubCopilotOAuthTestService(upstream, nil)
	started, err := service.Start(context.Background(), 42, nil)
	require.NoError(t, err)
	*now = now.Add(5 * time.Second)

	_, err = service.Poll(context.Background(), 42, started.OAuthSessionID)
	require.ErrorIs(t, err, ErrGitHubCopilotOAuthUnavailable)
	require.NotContains(t, err.Error(), "server-only-device-code")
}

func TestGitHubCopilotOAuthExpiryClearsSecrets(t *testing.T) {
	t.Parallel()

	upstream := &githubCopilotOAuthHTTPStub{handler: func(_ *http.Request, _ string, _ string) (*http.Response, error) {
		return githubCopilotOAuthJSONResponse(http.StatusOK, githubCopilotDeviceStartJSON()), nil
	}}
	service, now := newGitHubCopilotOAuthTestService(upstream, nil)
	started, err := service.Start(context.Background(), 42, nil)
	require.NoError(t, err)
	*now = now.Add(15 * time.Minute)

	poll, err := service.Poll(context.Background(), 42, started.OAuthSessionID)
	require.NoError(t, err)
	require.Equal(t, GitHubCopilotOAuthStatusExpired, poll.Status)
	service.mu.Lock()
	require.Empty(t, service.sessions[started.OAuthSessionID].deviceCode)
	service.mu.Unlock()
	_, err = service.CheckoutCredential(42, started.OAuthSessionID)
	require.ErrorIs(t, err, ErrGitHubCopilotOAuthSessionExpired)
}

func TestGitHubCopilotOAuthScheduledCleanupRemovesAbandonedSession(t *testing.T) {
	service := NewGitHubCopilotOAuthService(nil, nil, nil)
	service.sessions["abandoned"] = &githubCopilotOAuthSession{
		deviceCode:  "server-only-device-code",
		accessToken: "gho-sensitive-source",
		expiresAt:   time.Now().Add(-githubCopilotExpiredRetention),
	}
	service.scheduleCleanup("abandoned", time.Now().Add(10*time.Millisecond))

	require.Eventually(t, func() bool {
		service.mu.Lock()
		defer service.mu.Unlock()
		_, exists := service.sessions["abandoned"]
		return !exists
	}, time.Second, 10*time.Millisecond)
}

func TestGitHubCopilotOAuthRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	upstream := &githubCopilotOAuthHTTPStub{handler: func(_ *http.Request, _ string, _ string) (*http.Response, error) {
		return githubCopilotOAuthJSONResponse(http.StatusOK, strings.Repeat("x", githubCopilotMaxResponseBytes+1)), nil
	}}
	service, _ := newGitHubCopilotOAuthTestService(upstream, nil)
	_, err := service.Start(context.Background(), 42, nil)
	require.ErrorIs(t, err, ErrGitHubCopilotOAuthInvalidResponse)
}

func TestGitHubCopilotOAuthKnownTerminalResponses(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		oauthError string
		wantStatus string
	}{
		{name: "denied", oauthError: "access_denied", wantStatus: GitHubCopilotOAuthStatusDenied},
		{name: "expired", oauthError: "expired_token", wantStatus: GitHubCopilotOAuthStatusExpired},
		{name: "unknown", oauthError: "incorrect_device_code", wantStatus: GitHubCopilotOAuthStatusError},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			upstream := &githubCopilotOAuthHTTPStub{handler: func(req *http.Request, _ string, _ string) (*http.Response, error) {
				if req.URL.String() == githubCopilotDeviceCodeURL {
					return githubCopilotOAuthJSONResponse(http.StatusOK, githubCopilotDeviceStartJSON()), nil
				}
				return githubCopilotOAuthJSONResponse(http.StatusOK, fmt.Sprintf(`{"error":%q}`, testCase.oauthError)), nil
			}}
			service, now := newGitHubCopilotOAuthTestService(upstream, nil)
			started, err := service.Start(context.Background(), 42, nil)
			require.NoError(t, err)
			*now = now.Add(5 * time.Second)
			poll, err := service.Poll(context.Background(), 42, started.OAuthSessionID)
			require.NoError(t, err)
			require.Equal(t, testCase.wantStatus, poll.Status)
			_, err = service.CheckoutCredential(42, started.OAuthSessionID)
			require.ErrorIs(t, err, ErrGitHubCopilotOAuthNotAuthorized)
		})
	}
}

type githubCopilotOAuthSettingRepo struct {
	values map[string]string
}

func (r *githubCopilotOAuthSettingRepo) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}
func (r *githubCopilotOAuthSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	return r.values[key], nil
}
func (r *githubCopilotOAuthSettingRepo) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}
func (r *githubCopilotOAuthSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}
func (r *githubCopilotOAuthSettingRepo) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		r.values[key] = value
	}
	return nil
}
func (r *githubCopilotOAuthSettingRepo) GetAll(context.Context) (map[string]string, error) {
	result := make(map[string]string, len(r.values))
	for key, value := range r.values {
		result[key] = value
	}
	return result, nil
}
func (r *githubCopilotOAuthSettingRepo) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestGetGitHubCopilotOAuthClientIDIgnoresLoginToggleAndSecret(t *testing.T) {
	t.Parallel()

	repository := &githubCopilotOAuthSettingRepo{values: map[string]string{
		SettingKeyGitHubOAuthEnabled:  "false",
		SettingKeyGitHubOAuthClientID: " database-client-id ",
	}}
	settingService := NewSettingService(repository, &config.Config{
		GitHubOAuth: config.EmailOAuthProviderConfig{
			Enabled:      false,
			ClientID:     "config-client-id",
			ClientSecret: "",
		},
	})

	clientID, err := settingService.GetGitHubCopilotOAuthClientID(context.Background())
	require.NoError(t, err)
	require.Equal(t, "database-client-id", clientID)

	delete(repository.values, SettingKeyGitHubOAuthClientID)
	clientID, err = settingService.GetGitHubCopilotOAuthClientID(context.Background())
	require.NoError(t, err)
	require.Equal(t, "config-client-id", clientID)
}

func TestGitHubCopilotOAuthTypedErrors(t *testing.T) {
	t.Parallel()

	unconfigured := NewGitHubCopilotOAuthService(&githubCopilotOAuthSettingsStub{}, nil, nil)
	_, err := unconfigured.Start(context.Background(), 42, nil)
	require.ErrorIs(t, err, ErrGitHubCopilotOAuthNotConfigured)

	settingsFailure := NewGitHubCopilotOAuthService(
		&githubCopilotOAuthSettingsStub{err: errors.New("database unavailable")},
		nil,
		nil,
	)
	_, err = settingsFailure.Start(context.Background(), 42, nil)
	require.Equal(t, "GITHUB_COPILOT_OAUTH_SETTINGS_UNAVAILABLE", infraErrorReason(err))
}

func infraErrorReason(err error) string {
	return infraerrors.Reason(err)
}
