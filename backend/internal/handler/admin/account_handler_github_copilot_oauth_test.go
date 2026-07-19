package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type githubCopilotOAuthHandlerStub struct {
	credential      *service.GitHubCopilotOAuthCredential
	checkoutAdminID int64
	checkoutSession string
	finalized       int
	released        int
}

type failMarkSucceededRepo struct {
	*memoryIdempotencyRepoStub
}

func (r *failMarkSucceededRepo) MarkSucceeded(context.Context, int64, int, string, time.Time) error {
	return errors.New("idempotency result store unavailable")
}

func (s *githubCopilotOAuthHandlerStub) Capabilities(context.Context) (*service.GitHubCopilotOAuthCapabilities, error) {
	return &service.GitHubCopilotOAuthCapabilities{Configured: true, PATURL: service.GitHubCopilotPATURL}, nil
}

func (s *githubCopilotOAuthHandlerStub) Start(context.Context, int64, *int64) (*service.GitHubCopilotDeviceStartResult, error) {
	return &service.GitHubCopilotDeviceStartResult{
		OAuthSessionID:  "opaque-session",
		UserCode:        "ABCD-EFGH",
		VerificationURI: "https://github.com/login/device",
		ExpiresIn:       900,
		Interval:        5,
	}, nil
}

func (s *githubCopilotOAuthHandlerStub) Poll(context.Context, int64, string) (*service.GitHubCopilotDevicePollResult, error) {
	return &service.GitHubCopilotDevicePollResult{
		Status:       service.GitHubCopilotOAuthStatusAuthorized,
		GitHubLogin:  "octocat",
		GitHubUserID: 1,
	}, nil
}

func (s *githubCopilotOAuthHandlerStub) CheckoutCredential(adminID int64, sessionID string) (*service.GitHubCopilotOAuthCredential, error) {
	s.checkoutAdminID = adminID
	s.checkoutSession = sessionID
	return s.credential, nil
}

func (s *githubCopilotOAuthHandlerStub) ReleaseCredential(int64, string) error {
	s.released++
	return nil
}

func (s *githubCopilotOAuthHandlerStub) FinalizeCredential(int64, string) error {
	s.finalized++
	return nil
}

func setupGitHubCopilotOAuthAccountRouter(
	adminService service.AdminService,
	oauthService githubCopilotOAuthCredentialService,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewAccountHandler(adminService, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.githubCopilotOAuth = oauthService
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		c.Next()
	})
	router.POST("/accounts", handler.Create)
	router.GET("/accounts/github-copilot/oauth/capabilities", handler.GetGitHubCopilotOAuthCapabilities)
	router.POST("/accounts/github-copilot/oauth/device/start", handler.StartGitHubCopilotOAuth)
	router.POST("/accounts/github-copilot/oauth/device/poll", handler.PollGitHubCopilotOAuth)
	return router
}

func TestAccountCreateResolvesGitHubCopilotOAuthCredentialServerSide(t *testing.T) {
	adminService := newStubAdminService()
	oauthService := &githubCopilotOAuthHandlerStub{credential: &service.GitHubCopilotOAuthCredential{
		AccessToken:  "gho_server_side_secret",
		GitHubLogin:  "octocat",
		GitHubUserID: 1,
	}}
	router := setupGitHubCopilotOAuthAccountRouter(adminService, oauthService)
	body := `{
		"name":"Copilot",
		"platform":"openai",
		"type":"apikey",
		"credentials":{
			"github_copilot_oauth_session_id":"opaque-session",
			"model_mapping":{"gpt-5.4":"gpt-5.4"},
			"openai_capabilities":["chat_completions"]
		},
		"extra":{"upstream_provider":"github_copilot"}
	}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, adminService.createdAccounts, 1)
	credentials := adminService.createdAccounts[0].Credentials
	require.Equal(t, "gho_server_side_secret", credentials["api_key"])
	require.NotContains(t, credentials, githubCopilotOAuthSessionCredentialKey)
	require.Equal(t, int64(42), oauthService.checkoutAdminID)
	require.Equal(t, "opaque-session", oauthService.checkoutSession)
	require.Equal(t, 1, oauthService.finalized)
	require.Zero(t, oauthService.released)
	require.NotContains(t, recorder.Body.String(), "gho_server_side_secret")
	require.NotContains(t, recorder.Body.String(), "opaque-session")
}

func TestAccountCreateReleasesGitHubCopilotOAuthCredentialAfterFailure(t *testing.T) {
	adminService := newStubAdminService()
	adminService.createAccountErr = infraerrors.BadRequest("CREATE_REJECTED", "account rejected")
	oauthService := &githubCopilotOAuthHandlerStub{credential: &service.GitHubCopilotOAuthCredential{
		AccessToken: "gho_server_side_secret",
	}}
	router := setupGitHubCopilotOAuthAccountRouter(adminService, oauthService)
	body := `{
		"name":"Copilot",
		"platform":"openai",
		"type":"apikey",
		"credentials":{"github_copilot_oauth_session_id":"opaque-session"},
		"extra":{"upstream_provider":"github_copilot"}
	}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Zero(t, oauthService.finalized)
	require.Equal(t, 1, oauthService.released)
	require.NotContains(t, recorder.Body.String(), "gho_server_side_secret")
}

func TestAccountCreateFinalizesGitHubCredentialAfterIdempotencyResultStoreFailure(t *testing.T) {
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(
		&failMarkSucceededRepo{memoryIdempotencyRepoStub: newMemoryIdempotencyRepoStub()},
		service.DefaultIdempotencyConfig(),
	))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(nil) })

	adminService := newStubAdminService()
	oauthService := &githubCopilotOAuthHandlerStub{credential: &service.GitHubCopilotOAuthCredential{
		AccessToken: "gho_server_side_secret",
	}}
	router := setupGitHubCopilotOAuthAccountRouter(adminService, oauthService)
	body := `{
		"name":"Copilot",
		"platform":"openai",
		"type":"apikey",
		"credentials":{"github_copilot_oauth_session_id":"opaque-session"},
		"extra":{"upstream_provider":"github_copilot"}
	}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "copilot-oauth-create")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Len(t, adminService.createdAccounts, 1, "the business create completed before result storage failed")
	require.Equal(t, 1, oauthService.finalized, "persisted credentials must not be reusable")
	require.Zero(t, oauthService.released)
	require.NotContains(t, recorder.Body.String(), "gho_server_side_secret")
}

func TestAccountCreateRejectsOAuthSessionOutsideGitHubCopilot(t *testing.T) {
	adminService := newStubAdminService()
	oauthService := &githubCopilotOAuthHandlerStub{credential: &service.GitHubCopilotOAuthCredential{
		AccessToken: "gho_server_side_secret",
	}}
	router := setupGitHubCopilotOAuthAccountRouter(adminService, oauthService)
	body := `{
		"name":"OpenAI",
		"platform":"openai",
		"type":"apikey",
		"credentials":{"github_copilot_oauth_session_id":"opaque-session"},
		"extra":{}
	}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Empty(t, adminService.createdAccounts)
	require.Empty(t, oauthService.checkoutSession)
}

func TestGitHubCopilotOAuthResponsesNeverContainCredentials(t *testing.T) {
	oauthService := &githubCopilotOAuthHandlerStub{credential: &service.GitHubCopilotOAuthCredential{
		AccessToken: "gho_server_side_secret",
	}}
	router := setupGitHubCopilotOAuthAccountRouter(newStubAdminService(), oauthService)

	requests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/accounts/github-copilot/oauth/capabilities"},
		{method: http.MethodPost, path: "/accounts/github-copilot/oauth/device/start", body: `{}`},
		{method: http.MethodPost, path: "/accounts/github-copilot/oauth/device/poll", body: `{"oauth_session_id":"opaque-session"}`},
	}
	for _, testCase := range requests {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(testCase.method, testCase.path, strings.NewReader(testCase.body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.NotContains(t, recorder.Body.String(), "gho_server_side_secret")
		var envelope map[string]any
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
		require.NotNil(t, envelope["data"])
	}
}
