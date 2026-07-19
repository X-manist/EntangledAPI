package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type githubCopilotTestHTTPUpstream struct {
	do    func(*http.Request) (*http.Response, error)
	calls int
}

func (u *githubCopilotTestHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.calls++
	return u.do(req)
}

func (u *githubCopilotTestHTTPUpstream) DoWithTLS(
	req *http.Request,
	_ string,
	_ int64,
	_ int,
	_ *tlsfingerprint.Profile,
) (*http.Response, error) {
	return u.Do(req, "", 0, 0)
}

func TestDiscoverGitHubCopilotSession(t *testing.T) {
	startedAt := time.Now()
	upstream := &githubCopilotTestHTTPUpstream{}
	upstream.do = func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, req.Method)
		require.Equal(t, githubCopilotUserDiscoveryURL, req.URL.String())
		require.Equal(t, "Bearer github_pat_source", req.Header.Get("Authorization"))
		require.Empty(t, req.Header.Get("Copilot-Integration-Id"))
		body := `{"chat_enabled":true,"endpoints":{"api":"https://api.individual.githubcopilot.com"}}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	account := newGitHubCopilotTestAccount("github_pat_source")

	session, err := discoverGitHubCopilotSessionAt(context.Background(), upstream, account, githubCopilotUserDiscoveryURL)
	require.NoError(t, err)
	require.Equal(t, "github_pat_source", session.token)
	require.Equal(t, "https://api.individual.githubcopilot.com", session.apiBase)
	require.WithinDuration(t, startedAt.Add(githubCopilotDiscoveryTTL), session.expiresAt, time.Second)
}

func TestDiscoverGitHubCopilotSessionAccessMetadataAndTokenTypes(t *testing.T) {
	t.Run("no access", func(t *testing.T) {
		upstream := &githubCopilotTestHTTPUpstream{do: func(_ *http.Request) (*http.Response, error) {
			body := `{"access_type_sku":"no_access","can_signup_for_limited":true,"chat_enabled":true,"endpoints":{"api":"https://api.githubcopilot.com"}}`
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		}}
		_, err := discoverGitHubCopilotSessionAt(context.Background(), upstream, newGitHubCopilotTestAccount("github_pat_source"), githubCopilotUserDiscoveryURL)
		require.Error(t, err)
		var sessionErr *githubCopilotSessionError
		require.ErrorAs(t, err, &sessionErr)
		require.Equal(t, http.StatusForbidden, sessionErr.statusCode)
		require.Contains(t, err.Error(), "must be enabled")
	})

	t.Run("chat disabled", func(t *testing.T) {
		upstream := &githubCopilotTestHTTPUpstream{do: func(_ *http.Request) (*http.Response, error) {
			body := `{"access_type_sku":"copilot_for_individual_user","chat_enabled":false,"endpoints":{"api":"https://api.githubcopilot.com"}}`
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		}}
		session, err := discoverGitHubCopilotSessionAt(context.Background(), upstream, newGitHubCopilotTestAccount("gho_source"), githubCopilotUserDiscoveryURL)
		require.NoError(t, err)
		require.Equal(t, "gho_source", session.token)
		require.Equal(t, githubCopilotDefaultAPIBase, session.apiBase)
	})

	t.Run("classic PAT", func(t *testing.T) {
		upstream := &githubCopilotTestHTTPUpstream{do: func(_ *http.Request) (*http.Response, error) {
			t.Fatal("classic PAT must be rejected before an upstream request")
			return nil, nil
		}}
		_, err := discoverGitHubCopilotSessionAt(context.Background(), upstream, newGitHubCopilotTestAccount("ghp_classic"), githubCopilotUserDiscoveryURL)
		require.Error(t, err)
		var sessionErr *githubCopilotSessionError
		require.ErrorAs(t, err, &sessionErr)
		require.Equal(t, http.StatusBadRequest, sessionErr.statusCode)
		require.Contains(t, err.Error(), "does not support classic")
	})

	t.Run("null response", func(t *testing.T) {
		upstream := &githubCopilotTestHTTPUpstream{do: func(_ *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("null"))}, nil
		}}
		_, err := discoverGitHubCopilotSessionAt(context.Background(), upstream, newGitHubCopilotTestAccount("gho_source"), githubCopilotUserDiscoveryURL)
		require.ErrorContains(t, err, "response must be a JSON object")
	})
}

func TestApplyGitHubCopilotHeadersUsesTruthfulIdentityAndRequestIDs(t *testing.T) {
	first := make(http.Header)
	second := make(http.Header)

	applyGitHubCopilotHeaders(first)
	applyGitHubCopilotHeaders(second)

	for _, header := range []http.Header{first, second} {
		require.Equal(t, "application/json", header.Get("Content-Type"))
		require.Equal(t, "sub2api", header.Get("Copilot-Integration-Id"))
		require.Equal(t, "sub2api/1.0", header.Get("Editor-Version"))
		require.Equal(t, "sub2api/1.0", header.Get("User-Agent"))
		require.Equal(t, "2026-07-01", header.Get("X-GitHub-Api-Version"))
		require.Equal(t, "conversation-agent", header.Get("OpenAI-Intent"))
		require.Equal(t, "user", header.Get("X-Initiator"))

		machineID, err := uuid.Parse(header.Get("X-Client-Machine-Id"))
		require.NoError(t, err)
		require.Equal(t, uuid.Version(4), machineID.Version())

		interactionID, err := uuid.Parse(header.Get("X-Interaction-Id"))
		require.NoError(t, err)
		require.Equal(t, uuid.Version(4), interactionID.Version())
	}

	require.Equal(t, first.Get("X-Client-Machine-Id"), second.Get("X-Client-Machine-Id"))
	require.NotEqual(t, first.Get("X-Interaction-Id"), second.Get("X-Interaction-Id"))
}

func TestEnsureGitHubCopilotSessionCachesAndRotatesCredential(t *testing.T) {
	upstream := &githubCopilotTestHTTPUpstream{}
	upstream.do = func(req *http.Request) (*http.Response, error) {
		require.True(t, strings.HasPrefix(req.Header.Get("Authorization"), "Bearer "))
		body := `{"chat_enabled":true,"endpoints":{"api":"https://api.githubcopilot.com"}}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	service := &OpenAIGatewayService{httpUpstream: upstream}
	account := newGitHubCopilotTestAccount("first")

	first, err := service.ensureGitHubCopilotSession(context.Background(), account)
	require.NoError(t, err)
	second, err := service.ensureGitHubCopilotSession(context.Background(), account)
	require.NoError(t, err)
	require.Equal(t, first.token, second.token)
	require.Equal(t, 1, upstream.calls)

	account.Credentials["api_key"] = "rotated"
	rotated, err := service.ensureGitHubCopilotSession(context.Background(), account)
	require.NoError(t, err)
	require.Equal(t, "rotated", rotated.token)
	require.Equal(t, 2, upstream.calls)
}

func TestValidateGitHubCopilotAPIBaseRejectsUntrustedEndpoint(t *testing.T) {
	valid, err := validateGitHubCopilotAPIBase("https://api.business.githubcopilot.com/")
	require.NoError(t, err)
	require.Equal(t, "https://api.business.githubcopilot.com", valid)

	for _, raw := range []string{
		"http://api.githubcopilot.com",
		"https://githubcopilot.com.example.org",
		"https://api.githubcopilot.com.evil.test",
		"https://user@api.githubcopilot.com",
	} {
		_, err := validateGitHubCopilotAPIBase(raw)
		require.Error(t, err, raw)
	}
}

func TestGitHubCopilotEndpointOmitsOpenAIV1Prefix(t *testing.T) {
	endpoint, err := githubCopilotAPIEndpoint("https://api.githubcopilot.com", "/chat/completions")
	require.NoError(t, err)
	require.Equal(t, "https://api.githubcopilot.com/chat/completions", endpoint)
}

func TestForwardAsRawChatCompletionsUsesOriginalGitHubToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &githubCopilotTestHTTPUpstream{}
	upstream.do = func(req *http.Request) (*http.Response, error) {
		switch upstream.calls {
		case 1:
			require.Equal(t, githubCopilotUserDiscoveryURL, req.URL.String())
			require.Equal(t, "Bearer github-source-token", req.Header.Get("Authorization"))
			body := `{"chat_enabled":true,"endpoints":{"api":"https://api.individual.githubcopilot.com"}}`
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		case 2:
			require.Equal(t, "https://api.individual.githubcopilot.com/chat/completions", req.URL.String())
			require.Equal(t, "Bearer github-source-token", req.Header.Get("Authorization"))
			require.Equal(t, githubCopilotIntegrationID, req.Header.Get("Copilot-Integration-Id"))
			body := `{"id":"chatcmpl-copilot","object":"chat.completion","model":"gpt-5.4","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		default:
			return nil, fmt.Errorf("unexpected upstream call %d", upstream.calls)
		}
	}

	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	service := &OpenAIGatewayService{httpUpstream: upstream}
	account := newGitHubCopilotTestAccount("github-source-token")
	account.Credentials["model_mapping"] = map[string]any{"gpt-5.4": "gpt-5.4"}

	result, err := service.forwardAsRawChatCompletions(context.Background(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 3, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)
	require.Equal(t, 2, upstream.calls)
	require.Contains(t, recorder.Body.String(), `"content":"ok"`)
}

func TestForwardAsRawChatCompletionsRefreshesDiscoveryOnceAfter401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &githubCopilotTestHTTPUpstream{}
	upstream.do = func(req *http.Request) (*http.Response, error) {
		switch upstream.calls {
		case 1, 3:
			apiBase := "https://api.githubcopilot.com"
			if upstream.calls == 3 {
				apiBase = "https://api.individual.githubcopilot.com"
			}
			body := fmt.Sprintf(`{"chat_enabled":true,"endpoints":{"api":%q}}`, apiBase)
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		case 2:
			require.Equal(t, "Bearer github-source-token", req.Header.Get("Authorization"))
			return &http.Response{StatusCode: http.StatusUnauthorized, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"expired"}`))}, nil
		case 4:
			require.Equal(t, "Bearer github-source-token", req.Header.Get("Authorization"))
			require.Equal(t, "https://api.individual.githubcopilot.com/chat/completions", req.URL.String())
			body := `{"id":"chatcmpl-copilot","object":"chat.completion","model":"gpt-5.4","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		default:
			return nil, fmt.Errorf("unexpected upstream call %d", upstream.calls)
		}
	}

	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	service := &OpenAIGatewayService{httpUpstream: upstream}
	account := newGitHubCopilotTestAccount("github-source-token")
	account.Credentials["model_mapping"] = map[string]any{"gpt-5.4": "gpt-5.4"}

	result, err := service.forwardAsRawChatCompletions(context.Background(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 4, upstream.calls)
}

func TestForwardAsRawChatCompletionsCopilotDiscoveryFailureTriggersFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &githubCopilotTestHTTPUpstream{}
	upstream.do = func(req *http.Request) (*http.Response, error) {
		require.Equal(t, githubCopilotUserDiscoveryURL, req.URL.String())
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"X-Request-Id": []string{"gho-sensitive-reflection"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":"gho-sensitive-reflection"}`)),
		}, nil
	}
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	service := &OpenAIGatewayService{httpUpstream: upstream}

	result, err := service.forwardAsRawChatCompletions(context.Background(), c, newGitHubCopilotTestAccount("denied"), body, "")
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusForbidden, failoverErr.StatusCode)
	require.JSONEq(t, `{"error":{"message":"GitHub Copilot access is unavailable for this account","type":"permission_error"}}`, string(failoverErr.ResponseBody))
	require.NotContains(t, string(failoverErr.ResponseBody), "gho-sensitive-reflection")
	require.Empty(t, failoverErr.ResponseHeaders.Get("X-Request-Id"))
}

func TestGitHubCopilotFailoverErrorNormalizesDiscoveryStatusesAndHeaders(t *testing.T) {
	tests := []struct {
		name       string
		sessionErr *githubCopilotSessionError
		wantStatus int
		wantRetry  string
	}{
		{
			name: "secondary rate limit",
			sessionErr: &githubCopilotSessionError{
				statusCode:      http.StatusForbidden,
				responseBody:    []byte(`{"message":"rate limit gho-sensitive-source"}`),
				responseHeaders: http.Header{"Retry-After": []string{"17"}, "X-Request-Id": []string{"gho-sensitive-source"}},
				operation:       "GitHub Copilot access discovery returned HTTP 403",
			},
			wantStatus: http.StatusTooManyRequests,
			wantRetry:  "17",
		},
		{
			name: "permission-hidden not found",
			sessionErr: &githubCopilotSessionError{
				statusCode:   http.StatusNotFound,
				responseBody: []byte(`{"message":"Not Found"}`),
				operation:    "GitHub Copilot access discovery returned HTTP 404",
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "unsupported classic PAT",
			sessionErr: &githubCopilotSessionError{
				statusCode: http.StatusBadRequest,
				operation:  "GitHub Copilot does not support classic personal access tokens",
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var failoverErr *UpstreamFailoverError
			err := (&OpenAIGatewayService{}).githubCopilotFailoverError(context.Background(), newGitHubCopilotTestAccount("source"), test.sessionErr)
			require.ErrorAs(t, err, &failoverErr)
			require.Equal(t, test.wantStatus, failoverErr.StatusCode)
			require.Equal(t, test.wantRetry, failoverErr.ResponseHeaders.Get("Retry-After"))
			require.Equal(t, "application/json", failoverErr.ResponseHeaders.Get("Content-Type"))
			require.Empty(t, failoverErr.ResponseHeaders.Get("X-Request-Id"))
			require.NotContains(t, string(failoverErr.ResponseBody), "gho-sensitive-source")
		})
	}
}

func TestForwardAsRawChatCompletionsCopilotRefreshFailureTriggersFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &githubCopilotTestHTTPUpstream{}
	upstream.do = func(req *http.Request) (*http.Response, error) {
		switch upstream.calls {
		case 1:
			body := `{"chat_enabled":true,"endpoints":{"api":"https://api.githubcopilot.com"}}`
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		case 2:
			return &http.Response{StatusCode: http.StatusUnauthorized, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"expired"}`))}, nil
		case 3:
			return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"unavailable"}`))}, nil
		default:
			return nil, fmt.Errorf("unexpected upstream call %d", upstream.calls)
		}
	}
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	service := &OpenAIGatewayService{httpUpstream: upstream}

	result, err := service.forwardAsRawChatCompletions(context.Background(), c, newGitHubCopilotTestAccount("github-source-token"), body, "")
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusServiceUnavailable, failoverErr.StatusCode)
	require.Equal(t, 3, upstream.calls)
}

type blockingGitHubCopilotUpstream struct {
	calls   atomic.Int32
	started chan int64
	release chan struct{}
}

func (u *blockingGitHubCopilotUpstream) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.calls.Add(1)
	u.started <- accountID
	select {
	case <-u.release:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
	body := `{"chat_enabled":true,"endpoints":{"api":"https://api.githubcopilot.com"}}`
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
}

func (u *blockingGitHubCopilotUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

func TestEnsureGitHubCopilotSessionSingleflightsSameAccount(t *testing.T) {
	upstream := &blockingGitHubCopilotUpstream{started: make(chan int64, 16), release: make(chan struct{})}
	service := &OpenAIGatewayService{httpUpstream: upstream}
	account := newGitHubCopilotTestAccount("same-source")
	const workers = 16
	begin := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-begin
			_, err := service.ensureGitHubCopilotSession(context.Background(), account)
			errs <- err
		}()
	}
	close(begin)
	require.Equal(t, account.ID, <-upstream.started)
	close(upstream.release)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, int32(1), upstream.calls.Load())
}

func TestEnsureGitHubCopilotSessionDoesNotSerializeDifferentAccounts(t *testing.T) {
	upstream := &blockingGitHubCopilotUpstream{started: make(chan int64, 2), release: make(chan struct{})}
	service := &OpenAIGatewayService{httpUpstream: upstream}
	first := newGitHubCopilotTestAccount("first-source")
	second := newGitHubCopilotTestAccount("second-source")
	second.ID = first.ID + 1
	errs := make(chan error, 2)
	for _, account := range []*Account{first, second} {
		go func(account *Account) {
			_, err := service.ensureGitHubCopilotSession(context.Background(), account)
			errs <- err
		}(account)
	}
	seen := make(map[int64]bool, 2)
	for range 2 {
		select {
		case accountID := <-upstream.started:
			seen[accountID] = true
		case <-time.After(time.Second):
			close(upstream.release)
			t.Fatal("access discovery for different accounts was serialized")
		}
	}
	close(upstream.release)
	require.True(t, seen[first.ID])
	require.True(t, seen[second.ID])
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)
	require.Equal(t, int32(2), upstream.calls.Load())
}

func TestInvalidateGitHubCopilotSessionOnlyDeletesObservedSession(t *testing.T) {
	service := &OpenAIGatewayService{}
	sourceHash := sha256.Sum256([]byte("same-token"))
	fresh := githubCopilotSession{token: "same-token", apiBase: githubCopilotDefaultAPIBase, sourceHash: sourceHash, refreshSeq: 2}
	stale := githubCopilotSession{token: "same-token", apiBase: githubCopilotDefaultAPIBase, sourceHash: sourceHash, refreshSeq: 1}
	service.storeGitHubCopilotSession(42, fresh)
	require.False(t, service.invalidateGitHubCopilotSession(42, stale))
	raw, ok := service.githubCopilotSessions.Load(int64(42))
	require.True(t, ok)
	cached := raw.(githubCopilotSession)
	require.Empty(t, cached.token)
	require.Equal(t, fresh.apiBase, cached.apiBase)
	require.Equal(t, fresh.refreshSeq, cached.refreshSeq)
	require.True(t, service.invalidateGitHubCopilotSession(42, fresh))
	_, ok = service.githubCopilotSessions.Load(int64(42))
	require.False(t, ok)
}

func TestGitHubCopilotFailoverErrorDistinguishesRequestAndDiscoveryDeadlines(t *testing.T) {
	service := &OpenAIGatewayService{}
	internalTimeout := newGitHubCopilotSessionError(http.StatusBadGateway, "discover GitHub Copilot access", context.DeadlineExceeded)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, service.githubCopilotFailoverError(context.Background(), newGitHubCopilotTestAccount("source"), internalTimeout), &failoverErr)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)

	requestCtx, cancel := context.WithCancel(context.Background())
	cancel()
	requestErr := service.githubCopilotFailoverError(requestCtx, newGitHubCopilotTestAccount("source"), context.Canceled)
	require.ErrorIs(t, requestErr, context.Canceled)
	require.NotErrorAs(t, requestErr, &failoverErr)
}

func newGitHubCopilotTestAccount(sourceToken string) *Account {
	return &Account{
		ID:          42,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": sourceToken},
		Extra:       map[string]any{UpstreamProviderExtraKey: UpstreamProviderGitHubCopilot},
		Concurrency: 1,
	}
}
