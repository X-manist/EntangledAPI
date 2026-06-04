//go:build unit

package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type copilotHTTPUpstreamStub struct {
	resp     *http.Response
	err      error
	lastReq  *http.Request
	lastBody []byte
}

func (s *copilotHTTPUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	s.lastReq = req
	if req.Body != nil {
		s.lastBody, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(s.lastBody))
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.resp, nil
}

func (s *copilotHTTPUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, accountConcurrency)
}

func TestEnsureGitHubCopilotAccessToken_RefreshesOAuthToken(t *testing.T) {
	t.Parallel()

	account := &Account{
		ID:       99,
		Platform: PlatformGitHubCopilot,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"github_token": "ghu_test",
		},
	}
	upstream := &copilotHTTPUpstreamStub{resp: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"token":"copilot_session","expires_at":"2035-01-02T03:04:05Z","sku":"copilot_pro_plus"}`)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	token, err := svc.ensureGitHubCopilotAccessToken(context.Background(), account)
	require.NoError(t, err)
	require.Equal(t, "copilot_session", token)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "token ghu_test", upstream.lastReq.Header.Get("Authorization"))
}

func TestEnsureGitHubCopilotAccessToken_RejectsGitHubPAT(t *testing.T) {
	t.Parallel()

	account := &Account{
		ID:       99,
		Platform: PlatformGitHubCopilot,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"github_token": "ghp_test_token",
		},
	}
	upstream := &copilotHTTPUpstreamStub{resp: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"token":"copilot_session"}`)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	token, err := svc.ensureGitHubCopilotAccessToken(context.Background(), account)
	require.Error(t, err)
	require.Empty(t, token)
	require.Contains(t, err.Error(), "does not accept GitHub personal access tokens")
	require.Nil(t, upstream.lastReq)
}

func TestEnsureGitHubCopilotAccessToken_ParsesUnixExpiry(t *testing.T) {
	t.Parallel()

	account := &Account{
		ID:       99,
		Platform: PlatformGitHubCopilot,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"github_token": "ghu_test",
		},
	}
	upstream := &copilotHTTPUpstreamStub{resp: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"token":"copilot_session","expires_at":2051222400}`)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	token, err := svc.ensureGitHubCopilotAccessToken(context.Background(), account)
	require.NoError(t, err)
	require.Equal(t, "copilot_session", token)
}

func TestForwardAsRawChatCompletions_GitHubCopilotUsesChatCompletionsEndpoint(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"chatcmpl_1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:       7,
		Name:     "copilot-oauth",
		Platform: PlatformGitHubCopilot,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "copilot_session",
			"expires_at":   time.Now().Add(time.Hour).Format(time.RFC3339),
		},
	}

	result, err := svc.forwardAsRawChatCompletions(context.Background(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://api.githubcopilot.com/v1/chat/completions", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer copilot_session", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "vscode-chat", upstream.lastReq.Header.Get("Copilot-Integration-Id"))
}
