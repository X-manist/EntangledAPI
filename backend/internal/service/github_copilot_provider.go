package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	githubCopilotTokenExchangeURL = "https://api.github.com/copilot_internal/v2/token"
	githubCopilotDefaultAPIBase   = "https://api.githubcopilot.com"
	githubCopilotIntegrationID    = "vscode-chat"
	githubCopilotEditorVersion    = "vscode/1.107.0"
	githubCopilotUserAgent        = "GitHubCopilotChat/0.35.0"
	githubCopilotDefaultModel     = "gpt-5.4"
	githubCopilotTokenBodyLimit   = 64 * 1024
	githubCopilotExchangeTimeout  = 30 * time.Second
)

type githubCopilotSession struct {
	token      string
	apiBase    string
	expiresAt  time.Time
	sourceHash [sha256.Size]byte
	refreshSeq uint64
}

type githubCopilotSessionError struct {
	statusCode      int
	responseBody    []byte
	responseHeaders http.Header
	operation       string
	cause           error
}

func (e *githubCopilotSessionError) Error() string {
	if e == nil {
		return "GitHub Copilot session error"
	}
	if e.cause == nil {
		return e.operation
	}
	return fmt.Sprintf("%s: %v", e.operation, e.cause)
}

func (e *githubCopilotSessionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func newGitHubCopilotSessionError(statusCode int, operation string, cause error) error {
	return &githubCopilotSessionError{
		statusCode: statusCode,
		operation:  operation,
		cause:      cause,
	}
}

type githubCopilotTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	Endpoints struct {
		API string `json:"api"`
	} `json:"endpoints"`
}

func (s *OpenAIGatewayService) ensureGitHubCopilotSession(ctx context.Context, account *Account) (githubCopilotSession, error) {
	if s == nil || s.httpUpstream == nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "initialize GitHub Copilot session", errors.New("upstream HTTP client is not configured"))
	}
	if account == nil || !account.IsGitHubCopilot() {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "initialize GitHub Copilot session", errors.New("account is not a GitHub Copilot account"))
	}
	sourceToken := strings.TrimSpace(account.GetCredential("api_key"))
	if sourceToken == "" {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusUnauthorized, "initialize GitHub Copilot session", errors.New("GitHub token not found in credentials"))
	}
	sourceHash := sha256.Sum256([]byte(sourceToken))
	if cached, ok := s.loadValidGitHubCopilotSession(account.ID, sourceHash); ok {
		return cached, nil
	}

	refreshSeq := s.githubCopilotRefreshSeq.Add(1)
	flightKey := fmt.Sprintf("%d:%x", account.ID, sourceHash)
	baseCtx := context.Background()
	if ctx != nil {
		baseCtx = context.WithoutCancel(ctx)
	}
	resultCh := s.githubCopilotTokenSF.DoChan(flightKey, func() (any, error) {
		if cached, ok := s.loadValidGitHubCopilotSession(account.ID, sourceHash); ok {
			return cached, nil
		}
		exchangeCtx, cancel := context.WithTimeout(baseCtx, githubCopilotExchangeTimeout)
		defer cancel()
		session, err := exchangeGitHubCopilotTokenWithSource(
			exchangeCtx,
			s.httpUpstream,
			account,
			sourceToken,
			githubCopilotTokenExchangeURL,
		)
		if err != nil {
			return nil, err
		}
		session.sourceHash = sourceHash
		session.refreshSeq = refreshSeq
		s.storeGitHubCopilotSession(account.ID, session)
		return session, nil
	})

	var result any
	var err error
	if ctx == nil {
		flightResult := <-resultCh
		result, err = flightResult.Val, flightResult.Err
	} else {
		select {
		case <-ctx.Done():
			return githubCopilotSession{}, ctx.Err()
		case flightResult := <-resultCh:
			result, err = flightResult.Val, flightResult.Err
		}
	}
	if err != nil {
		return githubCopilotSession{}, err
	}
	session, ok := result.(githubCopilotSession)
	if !ok {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "initialize GitHub Copilot session", errors.New("invalid shared token exchange result"))
	}
	return session, nil
}

func (s *OpenAIGatewayService) loadValidGitHubCopilotSession(accountID int64, sourceHash [sha256.Size]byte) (githubCopilotSession, bool) {
	raw, ok := s.githubCopilotSessions.Load(accountID)
	if !ok {
		return githubCopilotSession{}, false
	}
	session, ok := raw.(githubCopilotSession)
	if !ok {
		s.githubCopilotSessions.CompareAndDelete(accountID, raw)
		return githubCopilotSession{}, false
	}
	if session.sourceHash != sourceHash {
		return githubCopilotSession{}, false
	}
	if session.token == "" || session.apiBase == "" || !time.Now().Before(session.expiresAt.Add(-time.Minute)) {
		s.githubCopilotSessions.CompareAndDelete(accountID, raw)
		return githubCopilotSession{}, false
	}
	return session, true
}

func (s *OpenAIGatewayService) storeGitHubCopilotSession(accountID int64, session githubCopilotSession) {
	for {
		raw, loaded := s.githubCopilotSessions.Load(accountID)
		if !loaded {
			if _, raced := s.githubCopilotSessions.LoadOrStore(accountID, session); !raced {
				return
			}
			continue
		}
		current, ok := raw.(githubCopilotSession)
		if ok && current.refreshSeq > session.refreshSeq {
			return
		}
		if s.githubCopilotSessions.CompareAndSwap(accountID, raw, session) {
			return
		}
	}
}

func (s *OpenAIGatewayService) resolveGitHubCopilotChatSession(ctx context.Context, account *Account) (githubCopilotSession, string, error) {
	session, err := s.ensureGitHubCopilotSession(ctx, account)
	if err != nil {
		return githubCopilotSession{}, "", err
	}
	targetURL, err := githubCopilotAPIEndpoint(session.apiBase, "/chat/completions")
	if err != nil {
		return githubCopilotSession{}, "", newGitHubCopilotSessionError(http.StatusBadGateway, "resolve GitHub Copilot API endpoint", err)
	}
	return session, targetURL, nil
}

func (s *OpenAIGatewayService) observeGitHubCopilotSession(accountID int64, token string, targetURL string) (githubCopilotSession, bool) {
	if s == nil || strings.TrimSpace(token) == "" || strings.TrimSpace(targetURL) == "" {
		return githubCopilotSession{}, false
	}
	raw, ok := s.githubCopilotSessions.Load(accountID)
	if !ok {
		return githubCopilotSession{}, false
	}
	session, ok := raw.(githubCopilotSession)
	if !ok || session.token != token {
		return githubCopilotSession{}, false
	}
	sessionTargetURL, err := githubCopilotAPIEndpoint(session.apiBase, "/chat/completions")
	if err != nil || sessionTargetURL != targetURL {
		return githubCopilotSession{}, false
	}
	return session, true
}

func (s *OpenAIGatewayService) invalidateGitHubCopilotSession(accountID int64, staleSession githubCopilotSession) bool {
	if s == nil || strings.TrimSpace(staleSession.token) == "" {
		return false
	}
	return s.githubCopilotSessions.CompareAndDelete(accountID, staleSession)
}

func (s *OpenAIGatewayService) githubCopilotFailoverError(ctx context.Context, account *Account, err error) error {
	if err == nil {
		return nil
	}
	var failoverErr *UpstreamFailoverError
	if errors.As(err, &failoverErr) {
		return err
	}
	if ctx != nil && ctx.Err() != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		return err
	}

	statusCode := http.StatusBadGateway
	var responseBody []byte
	var responseHeaders http.Header
	var sessionErr *githubCopilotSessionError
	if errors.As(err, &sessionErr) {
		if sessionErr.statusCode > 0 {
			statusCode = sessionErr.statusCode
		}
		responseBody = sessionErr.responseBody
		responseHeaders = sessionErr.responseHeaders
	}
	if s != nil {
		s.handleOpenAIAccountUpstreamError(ctx, account, statusCode, responseHeaders, responseBody)
	}
	return &UpstreamFailoverError{
		StatusCode:      statusCode,
		ResponseBody:    responseBody,
		ResponseHeaders: responseHeaders,
	}
}

func exchangeGitHubCopilotTokenAt(
	ctx context.Context,
	upstream HTTPUpstream,
	account *Account,
	exchangeURL string,
) (githubCopilotSession, error) {
	if account == nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "exchange GitHub Copilot token", errors.New("GitHub Copilot account is required"))
	}
	return exchangeGitHubCopilotTokenWithSource(ctx, upstream, account, strings.TrimSpace(account.GetCredential("api_key")), exchangeURL)
}

func exchangeGitHubCopilotTokenWithSource(
	ctx context.Context,
	upstream HTTPUpstream,
	account *Account,
	sourceToken string,
	exchangeURL string,
) (githubCopilotSession, error) {
	if upstream == nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "exchange GitHub Copilot token", errors.New("upstream HTTP client is not configured"))
	}
	if account == nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "exchange GitHub Copilot token", errors.New("GitHub Copilot account is required"))
	}
	sourceToken = strings.TrimSpace(sourceToken)
	if sourceToken == "" {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusUnauthorized, "exchange GitHub Copilot token", errors.New("GitHub token not found in credentials"))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exchangeURL, nil)
	if err != nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "build GitHub Copilot token request", err)
	}
	req.Header.Set("Authorization", "token "+sourceToken)
	req.Header.Set("Accept", "application/json")
	applyGitHubCopilotHeaders(req.Header)

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := upstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "exchange GitHub Copilot token", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, githubCopilotTokenBodyLimit+1))
	if err != nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "read GitHub Copilot token response", err)
	}
	if len(body) > githubCopilotTokenBodyLimit {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "read GitHub Copilot token response", errors.New("response is too large"))
	}
	if resp.StatusCode != http.StatusOK {
		return githubCopilotSession{}, &githubCopilotSessionError{
			statusCode:      resp.StatusCode,
			responseBody:    append([]byte(nil), body...),
			responseHeaders: resp.Header.Clone(),
			operation:       fmt.Sprintf("GitHub Copilot token exchange returned HTTP %d", resp.StatusCode),
		}
	}

	var parsed githubCopilotTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "parse GitHub Copilot token response", err)
	}
	parsed.Token = strings.TrimSpace(parsed.Token)
	if parsed.Token == "" {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "parse GitHub Copilot token response", errors.New("response did not contain a token"))
	}
	expiresAt := time.Unix(parsed.ExpiresAt, 0)
	if parsed.ExpiresAt <= 0 || !expiresAt.After(time.Now()) {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "parse GitHub Copilot token response", errors.New("response contained an invalid expiry"))
	}
	apiBase := strings.TrimSpace(parsed.Endpoints.API)
	if apiBase == "" {
		apiBase = githubCopilotDefaultAPIBase
	}
	apiBase, err = validateGitHubCopilotAPIBase(apiBase)
	if err != nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "validate GitHub Copilot API endpoint", err)
	}
	return githubCopilotSession{token: parsed.Token, apiBase: apiBase, expiresAt: expiresAt}, nil
}

func validateGitHubCopilotAPIBase(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.Opaque != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("invalid GitHub Copilot API endpoint")
	}
	hostname := strings.ToLower(parsed.Hostname())
	if parsed.Scheme != "https" ||
		(hostname != "api.githubcopilot.com" && !strings.HasSuffix(hostname, ".githubcopilot.com")) ||
		(parsed.Port() != "" && parsed.Port() != "443") {
		return "", errors.New("untrusted GitHub Copilot API endpoint")
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func githubCopilotAPIEndpoint(apiBase, endpoint string) (string, error) {
	validated, err := validateGitHubCopilotAPIBase(apiBase)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(validated, "/") + "/" + strings.TrimLeft(endpoint, "/"), nil
}

func applyGitHubCopilotHeaders(header http.Header) {
	header.Set("Editor-Version", githubCopilotEditorVersion)
	header.Set("Copilot-Integration-Id", githubCopilotIntegrationID)
	header.Set("X-GitHub-Api-Version", "2022-11-28")
	header.Set("User-Agent", githubCopilotUserAgent)
}
