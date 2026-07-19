package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	githubCopilotUserDiscoveryURL = "https://api.github.com/copilot_internal/user"
	githubCopilotDefaultAPIBase   = "https://api.githubcopilot.com"
	githubCopilotIntegrationID    = "sub2api"
	githubCopilotEditorVersion    = "sub2api/1.0"
	githubCopilotUserAgent        = "sub2api/1.0"
	githubCopilotAPIVersion       = "2026-07-01"
	githubCopilotOpenAIIntent     = "conversation-agent"
	githubCopilotInitiator        = "user"
	githubCopilotDefaultModel     = "gpt-5.4"
	githubCopilotDiscoveryTTL     = 10 * time.Minute
	githubCopilotBodyLimit        = 64 * 1024
	githubCopilotDiscoveryTimeout = 30 * time.Second
)

// githubCopilotMachineID identifies this sub2api process, not the host. It is
// intentionally random and process-local so no hardware identifier is exposed.
var githubCopilotMachineID = uuid.NewString()

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

type githubCopilotUserDiscoveryResponse struct {
	AccessTypeSKU       string `json:"access_type_sku"`
	CanSignupForLimited bool   `json:"can_signup_for_limited"`
	ChatEnabled         *bool  `json:"chat_enabled"`
	CopilotPlan         string `json:"copilot_plan"`
	Endpoints           struct {
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
		cached.token = sourceToken
		return cached, nil
	}

	refreshSeq := s.githubCopilotRefreshSeq.Add(1)
	flightKey := fmt.Sprintf("%d:%x", account.ID, sourceHash)
	baseCtx := context.Background()
	if ctx != nil {
		baseCtx = context.WithoutCancel(ctx)
	}
	resultCh := s.githubCopilotDiscoverySF.DoChan(flightKey, func() (any, error) {
		if cached, ok := s.loadValidGitHubCopilotSession(account.ID, sourceHash); ok {
			cached.token = sourceToken
			return cached, nil
		}
		discoveryCtx, cancel := context.WithTimeout(baseCtx, githubCopilotDiscoveryTimeout)
		defer cancel()
		session, err := discoverGitHubCopilotSessionWithSource(
			discoveryCtx,
			s.httpUpstream,
			account,
			sourceToken,
			githubCopilotUserDiscoveryURL,
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
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "initialize GitHub Copilot session", errors.New("invalid shared access discovery result"))
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
	if session.apiBase == "" || !time.Now().Before(session.expiresAt.Add(-time.Minute)) {
		s.githubCopilotSessions.CompareAndDelete(accountID, raw)
		return githubCopilotSession{}, false
	}
	return session, true
}

func (s *OpenAIGatewayService) storeGitHubCopilotSession(accountID int64, session githubCopilotSession) {
	// The original GitHub token already lives in the encrypted account
	// credentials. Cache only discovery metadata so the long-lived token is not
	// duplicated in process-wide memory.
	session.token = ""
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
	if !ok || session.sourceHash != sha256.Sum256([]byte(token)) {
		return githubCopilotSession{}, false
	}
	sessionTargetURL, err := githubCopilotAPIEndpoint(session.apiBase, "/chat/completions")
	if err != nil || sessionTargetURL != targetURL {
		return githubCopilotSession{}, false
	}
	session.token = token
	return session, true
}

func (s *OpenAIGatewayService) invalidateGitHubCopilotSession(accountID int64, staleSession githubCopilotSession) bool {
	if s == nil || strings.TrimSpace(staleSession.token) == "" {
		return false
	}
	staleSession.token = ""
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
		if statusCode == http.StatusForbidden && isGitHubRateLimitResponse(sessionErr.responseHeaders, sessionErr.responseBody) {
			statusCode = http.StatusTooManyRequests
		} else if statusCode == http.StatusNotFound {
			// GitHub deliberately uses 404 for resources the token cannot access.
			// Do not surface it as a missing downstream model or endpoint.
			statusCode = http.StatusForbidden
		} else if statusCode == http.StatusBadRequest && strings.Contains(strings.ToLower(sessionErr.operation), "classic personal access token") {
			// Existing accounts created before validation was added should leave the
			// scheduler after their unsupported classic PAT is discovered.
			statusCode = http.StatusUnauthorized
		}
		responseBody = githubCopilotSafeFailoverBody(statusCode)
		responseHeaders = githubCopilotSafeResponseHeaders(sessionErr.responseHeaders)
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

func githubCopilotSafeFailoverBody(statusCode int) []byte {
	message := "GitHub Copilot access discovery failed"
	errorType := "upstream_error"
	switch statusCode {
	case http.StatusUnauthorized:
		message = "GitHub Copilot authentication failed"
		errorType = "authentication_error"
	case http.StatusForbidden:
		message = "GitHub Copilot access is unavailable for this account"
		errorType = "permission_error"
	case http.StatusTooManyRequests:
		message = "GitHub Copilot rate limit exceeded"
		errorType = "rate_limit_error"
	}
	body, _ := json.Marshal(map[string]any{
		"error": map[string]string{
			"message": message,
			"type":    errorType,
		},
	})
	return body
}

func githubCopilotSafeResponseHeaders(upstream http.Header) http.Header {
	filtered := make(http.Header)
	filtered.Set("Content-Type", "application/json")
	if upstream == nil {
		return filtered
	}

	if value := strings.TrimSpace(upstream.Get("Retry-After")); value != "" {
		if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
			filtered.Set("Retry-After", strconv.FormatInt(seconds, 10))
		} else if retryAt, err := http.ParseTime(value); err == nil {
			filtered.Set("Retry-After", retryAt.UTC().Format(http.TimeFormat))
		}
	}
	for _, name := range []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"} {
		value := strings.TrimSpace(upstream.Get(name))
		parsed, err := strconv.ParseInt(value, 10, 64)
		if value != "" && err == nil && parsed >= 0 {
			filtered.Set(name, strconv.FormatInt(parsed, 10))
		}
	}
	return filtered
}

func discoverGitHubCopilotSessionAt(
	ctx context.Context,
	upstream HTTPUpstream,
	account *Account,
	discoveryURL string,
) (githubCopilotSession, error) {
	if account == nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "discover GitHub Copilot access", errors.New("GitHub Copilot account is required"))
	}
	return discoverGitHubCopilotSessionWithSource(ctx, upstream, account, strings.TrimSpace(account.GetCredential("api_key")), discoveryURL)
}

func discoverGitHubCopilotSessionWithSource(
	ctx context.Context,
	upstream HTTPUpstream,
	account *Account,
	sourceToken string,
	discoveryURL string,
) (githubCopilotSession, error) {
	if upstream == nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "discover GitHub Copilot access", errors.New("upstream HTTP client is not configured"))
	}
	if account == nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "discover GitHub Copilot access", errors.New("GitHub Copilot account is required"))
	}
	sourceToken = strings.TrimSpace(sourceToken)
	if sourceToken == "" {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusUnauthorized, "discover GitHub Copilot access", errors.New("GitHub token not found in credentials"))
	}
	if strings.HasPrefix(sourceToken, "ghp_") {
		return githubCopilotSession{}, &githubCopilotSessionError{
			statusCode: http.StatusBadRequest,
			operation:  "GitHub Copilot does not support classic personal access tokens; use GitHub authorization or a fine-grained token",
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "build GitHub Copilot access discovery request", err)
	}
	// Current Copilot CLI and SDK pass the original GitHub user token directly.
	// The discovery endpoint returns entitlement metadata and the account-specific
	// Copilot API base; it does not mint a second short-lived token.
	req.Header.Set("Authorization", "Bearer "+sourceToken)
	req.Header.Set("Accept", "application/json")

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := upstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "discover GitHub Copilot access", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, githubCopilotBodyLimit+1))
	if err != nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "read GitHub Copilot access discovery response", err)
	}
	if len(body) > githubCopilotBodyLimit {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "read GitHub Copilot access discovery response", errors.New("response is too large"))
	}
	if resp.StatusCode != http.StatusOK {
		return githubCopilotSession{}, &githubCopilotSessionError{
			statusCode:      resp.StatusCode,
			responseBody:    append([]byte(nil), body...),
			responseHeaders: resp.Header.Clone(),
			operation:       fmt.Sprintf("GitHub Copilot access discovery returned HTTP %d", resp.StatusCode),
		}
	}
	if bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "parse GitHub Copilot access discovery response", errors.New("response must be a JSON object"))
	}

	var parsed githubCopilotUserDiscoveryResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "parse GitHub Copilot access discovery response", err)
	}
	if strings.EqualFold(strings.TrimSpace(parsed.AccessTypeSKU), "no_access") {
		operation := "GitHub Copilot access is not enabled for this account"
		if parsed.CanSignupForLimited {
			operation = "GitHub Copilot must be enabled for this account before it can be used"
		}
		return githubCopilotSession{}, &githubCopilotSessionError{
			statusCode: http.StatusForbidden,
			operation:  operation,
		}
	}
	apiBase := strings.TrimSpace(parsed.Endpoints.API)
	if apiBase == "" {
		apiBase = githubCopilotDefaultAPIBase
	}
	apiBase, err = validateGitHubCopilotAPIBase(apiBase)
	if err != nil {
		return githubCopilotSession{}, newGitHubCopilotSessionError(http.StatusBadGateway, "validate GitHub Copilot API endpoint", err)
	}
	return githubCopilotSession{
		token:     sourceToken,
		apiBase:   apiBase,
		expiresAt: time.Now().Add(githubCopilotDiscoveryTTL),
	}, nil
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
	header.Set("Content-Type", "application/json")
	header.Set("Editor-Version", githubCopilotEditorVersion)
	header.Set("Copilot-Integration-Id", githubCopilotIntegrationID)
	header.Set("X-GitHub-Api-Version", githubCopilotAPIVersion)
	header.Set("OpenAI-Intent", githubCopilotOpenAIIntent)
	header.Set("X-Initiator", githubCopilotInitiator)
	header.Set("X-Client-Machine-Id", githubCopilotMachineID)
	header.Set("X-Interaction-Id", uuid.NewString())
	header.Set("User-Agent", githubCopilotUserAgent)
}
