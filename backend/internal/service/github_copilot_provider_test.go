package service

import (
	"bytes"
	"context"
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

func TestExchangeGitHubCopilotToken(t *testing.T) {
	expiresAt := time.Now().Add(10 * time.Minute).Unix()
	upstream := &githubCopilotTestHTTPUpstream{}
	upstream.do = func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, req.Method)
		require.Equal(t, "token github-secret", req.Header.Get("Authorization"))
		require.Equal(t, githubCopilotIntegrationID, req.Header.Get("Copilot-Integration-Id"))
		require.NotEmpty(t, req.Header.Get("Editor-Version"))
		body := fmt.Sprintf(`{"token":"short-lived","expires_at":%d,"endpoints":{"api":"https://api.individual.githubcopilot.com"}}`, expiresAt)
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	account := newGitHubCopilotTestAccount("github-secret")

	session, err := exchangeGitHubCopilotTokenAt(context.Background(), upstream, account, githubCopilotTokenExchangeURL)
	require.NoError(t, err)
	require.Equal(t, "short-lived", session.token)
	require.Equal(t, "https://api.individual.githubcopilot.com", session.apiBase)
	require.Equal(t, expiresAt, session.expiresAt.Unix())
}

func TestEnsureGitHubCopilotSessionCachesAndRotatesCredential(t *testing.T) {
	expiresAt := time.Now().Add(10 * time.Minute).Unix()
	upstream := &githubCopilotTestHTTPUpstream{}
	upstream.do = func(req *http.Request) (*http.Response, error) {
		source := strings.TrimPrefix(req.Header.Get("Authorization"), "token ")
		body := fmt.Sprintf(`{"token":"session-for-%s","expires_at":%d,"endpoints":{"api":"https://api.githubcopilot.com"}}`, source, expiresAt)
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
	require.Equal(t, "session-for-rotated", rotated.token)
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

func TestForwardAsRawChatCompletionsUsesExchangedCopilotToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expiresAt := time.Now().Add(10 * time.Minute).Unix()
	upstream := &githubCopilotTestHTTPUpstream{}
	upstream.do = func(req *http.Request) (*http.Response, error) {
		switch upstream.calls {
		case 1:
			require.Equal(t, githubCopilotTokenExchangeURL, req.URL.String())
			require.Equal(t, "token github-source-token", req.Header.Get("Authorization"))
			body := fmt.Sprintf(`{"token":"copilot-session-token","expires_at":%d,"endpoints":{"api":"https://api.individual.githubcopilot.com"}}`, expiresAt)
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		case 2:
			require.Equal(t, "https://api.individual.githubcopilot.com/chat/completions", req.URL.String())
			require.Equal(t, "Bearer copilot-session-token", req.Header.Get("Authorization"))
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

func TestForwardAsRawChatCompletionsRefreshesCopilotTokenOnceAfter401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expiresAt := time.Now().Add(10 * time.Minute).Unix()
	upstream := &githubCopilotTestHTTPUpstream{}
	upstream.do = func(req *http.Request) (*http.Response, error) {
		switch upstream.calls {
		case 1, 3:
			token := "stale-session"
			apiBase := "https://api.githubcopilot.com"
			if upstream.calls == 3 {
				token = "fresh-session"
				apiBase = "https://api.individual.githubcopilot.com"
			}
			body := fmt.Sprintf(`{"token":"%s","expires_at":%d,"endpoints":{"api":"%s"}}`, token, expiresAt, apiBase)
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		case 2:
			require.Equal(t, "Bearer stale-session", req.Header.Get("Authorization"))
			return &http.Response{StatusCode: http.StatusUnauthorized, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"expired"}`))}, nil
		case 4:
			require.Equal(t, "Bearer fresh-session", req.Header.Get("Authorization"))
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

func TestForwardAsRawChatCompletionsCopilotExchangeFailureTriggersFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &githubCopilotTestHTTPUpstream{}
	upstream.do = func(req *http.Request) (*http.Response, error) {
		require.Equal(t, githubCopilotTokenExchangeURL, req.URL.String())
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"X-Request-Id": []string{"copilot-exchange-denied"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":"forbidden"}`)),
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
	require.JSONEq(t, `{"error":"forbidden"}`, string(failoverErr.ResponseBody))
	require.Equal(t, "copilot-exchange-denied", failoverErr.ResponseHeaders.Get("X-Request-Id"))
}

func TestForwardAsRawChatCompletionsCopilotRefreshFailureTriggersFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expiresAt := time.Now().Add(10 * time.Minute).Unix()
	upstream := &githubCopilotTestHTTPUpstream{}
	upstream.do = func(req *http.Request) (*http.Response, error) {
		switch upstream.calls {
		case 1:
			body := fmt.Sprintf(`{"token":"stale-session","expires_at":%d,"endpoints":{"api":"https://api.githubcopilot.com"}}`, expiresAt)
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
	body := fmt.Sprintf(`{"token":"session-%d","expires_at":%d,"endpoints":{"api":"https://api.githubcopilot.com"}}`, accountID, time.Now().Add(10*time.Minute).Unix())
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
			t.Fatal("token exchanges for different accounts were serialized")
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
	fresh := githubCopilotSession{token: "same-token", apiBase: githubCopilotDefaultAPIBase, refreshSeq: 2}
	stale := githubCopilotSession{token: "same-token", apiBase: githubCopilotDefaultAPIBase, refreshSeq: 1}
	service.githubCopilotSessions.Store(int64(42), fresh)
	require.False(t, service.invalidateGitHubCopilotSession(42, stale))
	raw, ok := service.githubCopilotSessions.Load(int64(42))
	require.True(t, ok)
	require.Equal(t, fresh, raw.(githubCopilotSession))
	require.True(t, service.invalidateGitHubCopilotSession(42, fresh))
	_, ok = service.githubCopilotSessions.Load(int64(42))
	require.False(t, ok)
}

func TestGitHubCopilotFailoverErrorDistinguishesRequestAndExchangeDeadlines(t *testing.T) {
	service := &OpenAIGatewayService{}
	internalTimeout := newGitHubCopilotSessionError(http.StatusBadGateway, "exchange GitHub Copilot token", context.DeadlineExceeded)
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
