package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	githubCopilotDeviceCodeURL  = "https://github.com/login/device/code"
	githubCopilotAccessTokenURL = "https://github.com/login/oauth/access_token"
	githubCopilotUserURL        = "https://api.github.com/user"

	GitHubCopilotPATURL = "https://github.com/settings/personal-access-tokens/new?name=Sub2API%20Copilot&description=Use%20GitHub%20Copilot%20with%20Sub2API&copilot_requests=write"

	GitHubCopilotOAuthStatusPending    = "pending"
	GitHubCopilotOAuthStatusSlowDown   = "slow_down"
	GitHubCopilotOAuthStatusAuthorized = "authorized"
	GitHubCopilotOAuthStatusExpired    = "expired"
	GitHubCopilotOAuthStatusDenied     = "denied"
	GitHubCopilotOAuthStatusError      = "error"

	githubCopilotDefaultPollInterval = 5 * time.Second
	githubCopilotSlowDownIncrement   = 5 * time.Second
	githubCopilotMaxPollInterval     = 60 * time.Second
	githubCopilotMaxResponseBytes    = 64 << 10
	githubCopilotExpiredRetention    = 5 * time.Minute
)

var (
	ErrGitHubCopilotOAuthNotConfigured = infraerrors.ServiceUnavailable(
		"GITHUB_COPILOT_OAUTH_NOT_CONFIGURED",
		"GitHub authorization is not configured",
	)
	ErrGitHubCopilotOAuthUnavailable = infraerrors.ServiceUnavailable(
		"GITHUB_COPILOT_OAUTH_UNAVAILABLE",
		"GitHub authorization is temporarily unavailable",
	)
	ErrGitHubCopilotOAuthInvalidResponse = infraerrors.ServiceUnavailable(
		"GITHUB_COPILOT_OAUTH_INVALID_RESPONSE",
		"GitHub returned an invalid authorization response",
	)
	ErrGitHubCopilotOAuthInvalidInput = infraerrors.BadRequest(
		"GITHUB_COPILOT_OAUTH_INVALID_INPUT",
		"invalid GitHub authorization request",
	)
	ErrGitHubCopilotOAuthProxyNotFound = infraerrors.BadRequest(
		"GITHUB_COPILOT_OAUTH_PROXY_NOT_FOUND",
		"proxy not found",
	)
	ErrGitHubCopilotOAuthSessionNotFound = infraerrors.NotFound(
		"GITHUB_COPILOT_OAUTH_SESSION_NOT_FOUND",
		"GitHub authorization session not found",
	)
	ErrGitHubCopilotOAuthSessionForbidden = infraerrors.Forbidden(
		"GITHUB_COPILOT_OAUTH_SESSION_FORBIDDEN",
		"GitHub authorization session belongs to another administrator",
	)
	ErrGitHubCopilotOAuthSessionExpired = infraerrors.New(
		http.StatusGone,
		"GITHUB_COPILOT_OAUTH_SESSION_EXPIRED",
		"GitHub authorization session expired",
	)
	ErrGitHubCopilotOAuthNotAuthorized = infraerrors.Conflict(
		"GITHUB_COPILOT_OAUTH_NOT_AUTHORIZED",
		"GitHub authorization has not completed",
	)
	ErrGitHubCopilotOAuthCredentialInUse = infraerrors.Conflict(
		"GITHUB_COPILOT_OAUTH_CREDENTIAL_IN_USE",
		"GitHub authorization credential is already being used",
	)
	ErrGitHubCopilotOAuthEntitlementRequired = infraerrors.Forbidden(
		"GITHUB_COPILOT_OAUTH_ENTITLEMENT_REQUIRED",
		"the authorized GitHub account does not have an available Copilot entitlement",
	)
)

// GitHubCopilotOAuthSettings is the narrow settings port required by device
// flow. Unlike site sign-in, this flow only needs an OAuth App client ID.
type GitHubCopilotOAuthSettings interface {
	GetGitHubCopilotOAuthClientID(ctx context.Context) (string, error)
}

type GitHubCopilotOAuthCapabilities struct {
	Configured bool   `json:"configured"`
	PATURL     string `json:"pat_url"`
}

type GitHubCopilotDeviceStartResult struct {
	OAuthSessionID          string `json:"oauth_session_id"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
}

type GitHubCopilotDevicePollResult struct {
	Status       string `json:"status"`
	GitHubLogin  string `json:"github_login,omitempty"`
	GitHubUserID int64  `json:"github_user_id,omitempty"`
	Interval     int64  `json:"interval,omitempty"`
}

// GitHubCopilotOAuthCredential is an internal hand-off value. Its access token
// is explicitly excluded from JSON so a handler cannot accidentally serialize
// it back to the browser.
type GitHubCopilotOAuthCredential struct {
	AccessToken  string `json:"-"`
	GitHubLogin  string `json:"github_login"`
	GitHubUserID int64  `json:"github_user_id"`
}

type githubCopilotOAuthSession struct {
	adminID  int64
	proxyID  *int64
	proxy    *Proxy
	proxyURL string
	clientID string

	deviceCode   string
	accessToken  string
	githubLogin  string
	githubUserID int64

	createdAt  time.Time
	expiresAt  time.Time
	interval   time.Duration
	nextPollAt time.Time
	status     string
	polling    bool
	checkedOut bool
}

func (s *githubCopilotOAuthSession) clearSecrets() {
	s.deviceCode = ""
	s.accessToken = ""
}

type GitHubCopilotOAuthService struct {
	settings     GitHubCopilotOAuthSettings
	proxyRepo    ProxyRepository
	httpUpstream HTTPUpstream

	mu       sync.Mutex
	sessions map[string]*githubCopilotOAuthSession
	now      func() time.Time
	randomID func() (string, error)
}

func NewGitHubCopilotOAuthService(
	settings GitHubCopilotOAuthSettings,
	proxyRepo ProxyRepository,
	httpUpstream HTTPUpstream,
) *GitHubCopilotOAuthService {
	return &GitHubCopilotOAuthService{
		settings:     settings,
		proxyRepo:    proxyRepo,
		httpUpstream: httpUpstream,
		sessions:     make(map[string]*githubCopilotOAuthSession),
		now:          time.Now,
		randomID:     newGitHubCopilotOAuthSessionID,
	}
}

func (s *GitHubCopilotOAuthService) Capabilities(ctx context.Context) (*GitHubCopilotOAuthCapabilities, error) {
	clientID := ""
	if s != nil && s.settings != nil {
		var err error
		clientID, err = s.settings.GetGitHubCopilotOAuthClientID(ctx)
		if err != nil {
			return nil, infraerrors.ServiceUnavailable(
				"GITHUB_COPILOT_OAUTH_SETTINGS_UNAVAILABLE",
				"GitHub authorization settings are temporarily unavailable",
			)
		}
	}
	return &GitHubCopilotOAuthCapabilities{
		Configured: strings.TrimSpace(clientID) != "",
		PATURL:     GitHubCopilotPATURL,
	}, nil
}

func (s *GitHubCopilotOAuthService) Start(
	ctx context.Context,
	adminID int64,
	proxyID *int64,
) (*GitHubCopilotDeviceStartResult, error) {
	if adminID <= 0 {
		return nil, ErrGitHubCopilotOAuthInvalidInput
	}
	clientID, err := s.clientID(ctx)
	if err != nil {
		return nil, err
	}
	proxyURL, boundProxyID, boundProxy, err := s.resolveProxy(ctx, proxyID)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", "read:user")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubCopilotDeviceCodeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, ErrGitHubCopilotOAuthUnavailable
	}
	setGitHubCopilotOAuthHeaders(req, true)

	var upstream githubDeviceCodeResponse
	if err := s.doJSON(req, proxyURL, &upstream); err != nil {
		return nil, err
	}
	if err := validateGitHubDeviceCodeResponse(&upstream); err != nil {
		return nil, err
	}

	interval := time.Duration(upstream.Interval) * time.Second
	if interval <= 0 {
		interval = githubCopilotDefaultPollInterval
	}
	if interval > githubCopilotMaxPollInterval {
		interval = githubCopilotMaxPollInterval
	}
	sessionID, err := s.randomID()
	if err != nil {
		return nil, infraerrors.InternalServer(
			"GITHUB_COPILOT_OAUTH_SESSION_FAILED",
			"failed to create GitHub authorization session",
		)
	}

	now := s.now()
	expiresIn := time.Duration(upstream.ExpiresIn) * time.Second
	session := &githubCopilotOAuthSession{
		adminID:    adminID,
		proxyID:    boundProxyID,
		proxy:      boundProxy,
		proxyURL:   proxyURL,
		clientID:   clientID,
		deviceCode: upstream.DeviceCode,
		createdAt:  now,
		expiresAt:  now.Add(expiresIn),
		interval:   interval,
		nextPollAt: now.Add(interval),
		status:     GitHubCopilotOAuthStatusPending,
	}

	s.mu.Lock()
	s.cleanupLocked(now)
	s.sessions[sessionID] = session
	s.mu.Unlock()
	s.scheduleCleanup(sessionID, session.expiresAt.Add(githubCopilotExpiredRetention))

	return &GitHubCopilotDeviceStartResult{
		OAuthSessionID:          sessionID,
		UserCode:                upstream.UserCode,
		VerificationURI:         upstream.VerificationURI,
		VerificationURIComplete: upstream.VerificationURIComplete,
		ExpiresIn:               upstream.ExpiresIn,
		Interval:                int64(interval / time.Second),
	}, nil
}

func (s *GitHubCopilotOAuthService) Poll(
	ctx context.Context,
	adminID int64,
	sessionID string,
) (*GitHubCopilotDevicePollResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	if adminID <= 0 || sessionID == "" {
		return nil, ErrGitHubCopilotOAuthInvalidInput
	}

	now := s.now()
	s.mu.Lock()
	s.cleanupLocked(now)
	session, err := s.sessionForAdminLocked(adminID, sessionID)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if !now.Before(session.expiresAt) {
		session.status = GitHubCopilotOAuthStatusExpired
		session.polling = false
		session.clearSecrets()
		result := pollResultFromSession(session)
		s.mu.Unlock()
		return result, nil
	}
	if session.status != GitHubCopilotOAuthStatusPending {
		result := pollResultFromSession(session)
		s.mu.Unlock()
		return result, nil
	}
	if session.polling || now.Before(session.nextPollAt) {
		result := pollResultFromSession(session)
		result.Interval = ceilDurationSeconds(session.nextPollAt.Sub(now), session.interval)
		s.mu.Unlock()
		return result, nil
	}

	session.polling = true
	session.nextPollAt = now.Add(session.interval)
	deviceCode := session.deviceCode
	accessToken := session.accessToken
	clientID := session.clientID
	proxyURL := session.proxyURL
	proxy := session.proxy
	s.mu.Unlock()

	if accessToken != "" {
		return s.finishGitHubUserLookup(ctx, adminID, sessionID, proxyURL, proxy, accessToken)
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("device_code", deviceCode)
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubCopilotAccessTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		s.clearPolling(adminID, sessionID)
		return nil, ErrGitHubCopilotOAuthUnavailable
	}
	setGitHubCopilotOAuthHeaders(req, true)

	var tokenResponse githubAccessTokenResponse
	if err := s.doJSON(req, proxyURL, &tokenResponse); err != nil {
		s.clearPolling(adminID, sessionID)
		return nil, err
	}
	if tokenResponse.Error != "" {
		return s.finishOAuthError(adminID, sessionID, tokenResponse.Error)
	}
	accessToken = strings.TrimSpace(tokenResponse.AccessToken)
	if accessToken == "" {
		s.clearPolling(adminID, sessionID)
		return nil, ErrGitHubCopilotOAuthInvalidResponse
	}

	s.mu.Lock()
	session, err = s.sessionForAdminLocked(adminID, sessionID)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if !s.now().Before(session.expiresAt) {
		session.status = GitHubCopilotOAuthStatusExpired
		session.polling = false
		session.clearSecrets()
		result := pollResultFromSession(session)
		s.mu.Unlock()
		return result, nil
	}
	session.accessToken = accessToken
	session.deviceCode = ""
	s.mu.Unlock()

	return s.finishGitHubUserLookup(ctx, adminID, sessionID, proxyURL, proxy, accessToken)
}

// CheckoutCredential atomically reserves an authorized token for one account
// creation attempt. Call ReleaseCredential if creation fails, or
// FinalizeCredential after a successful create.
func (s *GitHubCopilotOAuthService) CheckoutCredential(adminID int64, sessionID string) (*GitHubCopilotOAuthCredential, error) {
	if adminID <= 0 || strings.TrimSpace(sessionID) == "" {
		return nil, ErrGitHubCopilotOAuthInvalidInput
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	session, err := s.sessionForAdminLocked(adminID, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, err
	}
	if !now.Before(session.expiresAt) {
		session.status = GitHubCopilotOAuthStatusExpired
		session.clearSecrets()
		return nil, ErrGitHubCopilotOAuthSessionExpired
	}
	if session.status != GitHubCopilotOAuthStatusAuthorized || session.accessToken == "" {
		return nil, ErrGitHubCopilotOAuthNotAuthorized
	}
	if session.checkedOut {
		return nil, ErrGitHubCopilotOAuthCredentialInUse
	}
	session.checkedOut = true
	return &GitHubCopilotOAuthCredential{
		AccessToken:  session.accessToken,
		GitHubLogin:  session.githubLogin,
		GitHubUserID: session.githubUserID,
	}, nil
}

func (s *GitHubCopilotOAuthService) ReleaseCredential(adminID int64, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.sessionForAdminLocked(adminID, strings.TrimSpace(sessionID))
	if err != nil {
		return err
	}
	if session.status != GitHubCopilotOAuthStatusAuthorized {
		return ErrGitHubCopilotOAuthNotAuthorized
	}
	session.checkedOut = false
	return nil
}

func (s *GitHubCopilotOAuthService) FinalizeCredential(adminID int64, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionID = strings.TrimSpace(sessionID)
	session, err := s.sessionForAdminLocked(adminID, sessionID)
	if err != nil {
		return err
	}
	if session.status != GitHubCopilotOAuthStatusAuthorized || !session.checkedOut {
		return ErrGitHubCopilotOAuthNotAuthorized
	}
	session.clearSecrets()
	delete(s.sessions, sessionID)
	return nil
}

func (s *GitHubCopilotOAuthService) clientID(ctx context.Context) (string, error) {
	if s == nil || s.settings == nil {
		return "", ErrGitHubCopilotOAuthNotConfigured
	}
	clientID, err := s.settings.GetGitHubCopilotOAuthClientID(ctx)
	if err != nil {
		return "", infraerrors.ServiceUnavailable(
			"GITHUB_COPILOT_OAUTH_SETTINGS_UNAVAILABLE",
			"GitHub authorization settings are temporarily unavailable",
		)
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return "", ErrGitHubCopilotOAuthNotConfigured
	}
	return clientID, nil
}

func (s *GitHubCopilotOAuthService) resolveProxy(ctx context.Context, proxyID *int64) (string, *int64, *Proxy, error) {
	if proxyID == nil {
		return "", nil, nil, nil
	}
	if *proxyID <= 0 || s.proxyRepo == nil {
		return "", nil, nil, ErrGitHubCopilotOAuthProxyNotFound
	}
	proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
	if err != nil || proxy == nil {
		return "", nil, nil, ErrGitHubCopilotOAuthProxyNotFound
	}
	boundID := *proxyID
	boundProxy := *proxy
	return boundProxy.URL(), &boundID, &boundProxy, nil
}

func (s *GitHubCopilotOAuthService) doJSON(req *http.Request, proxyURL string, target any) error {
	if s == nil || s.httpUpstream == nil {
		return ErrGitHubCopilotOAuthUnavailable
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, 0, 1)
	if err != nil {
		// Deliberately do not wrap transport errors: an implementation may include
		// the request body, which contains device_code, in its error text.
		return ErrGitHubCopilotOAuthUnavailable
	}
	if resp == nil || resp.Body == nil {
		return ErrGitHubCopilotOAuthInvalidResponse
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ErrGitHubCopilotOAuthUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, githubCopilotMaxResponseBytes+1))
	if err != nil || len(body) > githubCopilotMaxResponseBytes {
		return ErrGitHubCopilotOAuthInvalidResponse
	}
	if err := json.Unmarshal(body, target); err != nil {
		return ErrGitHubCopilotOAuthInvalidResponse
	}
	return nil
}

func (s *GitHubCopilotOAuthService) finishOAuthError(adminID int64, sessionID, oauthError string) (*GitHubCopilotDevicePollResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.sessionForAdminLocked(adminID, sessionID)
	if err != nil {
		return nil, err
	}
	session.polling = false
	switch strings.TrimSpace(oauthError) {
	case "authorization_pending":
		session.status = GitHubCopilotOAuthStatusPending
	case "slow_down":
		session.interval += githubCopilotSlowDownIncrement
		if session.interval > githubCopilotMaxPollInterval {
			session.interval = githubCopilotMaxPollInterval
		}
		session.nextPollAt = s.now().Add(session.interval)
		result := pollResultFromSession(session)
		result.Status = GitHubCopilotOAuthStatusSlowDown
		return result, nil
	case "access_denied":
		session.status = GitHubCopilotOAuthStatusDenied
		session.clearSecrets()
	case "expired_token":
		session.status = GitHubCopilotOAuthStatusExpired
		session.clearSecrets()
	default:
		session.status = GitHubCopilotOAuthStatusError
		session.clearSecrets()
	}
	return pollResultFromSession(session), nil
}

func (s *GitHubCopilotOAuthService) finishGitHubUserLookup(
	ctx context.Context,
	adminID int64,
	sessionID, proxyURL string,
	proxy *Proxy,
	accessToken string,
) (*GitHubCopilotDevicePollResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubCopilotUserURL, nil)
	if err != nil {
		s.clearPolling(adminID, sessionID)
		return nil, ErrGitHubCopilotOAuthUnavailable
	}
	setGitHubCopilotOAuthHeaders(req, false)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	var user githubUserResponse
	if err := s.doJSON(req, proxyURL, &user); err != nil {
		s.clearPolling(adminID, sessionID)
		return nil, err
	}
	user.Login = strings.TrimSpace(user.Login)
	if user.Login == "" || user.ID <= 0 {
		s.clearPolling(adminID, sessionID)
		return nil, ErrGitHubCopilotOAuthInvalidResponse
	}
	if err := s.validateCopilotEntitlement(ctx, proxy, accessToken); err != nil {
		if errors.Is(err, ErrGitHubCopilotOAuthEntitlementRequired) {
			s.markSessionError(adminID, sessionID)
		} else {
			// Keep the GitHub access token for a later retry when the entitlement
			// check failed because of a transient network or upstream condition.
			s.clearPolling(adminID, sessionID)
		}
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.sessionForAdminLocked(adminID, sessionID)
	if err != nil {
		return nil, err
	}
	if !s.now().Before(session.expiresAt) {
		session.status = GitHubCopilotOAuthStatusExpired
		session.polling = false
		session.clearSecrets()
		return pollResultFromSession(session), nil
	}
	session.githubLogin = user.Login
	session.githubUserID = user.ID
	session.status = GitHubCopilotOAuthStatusAuthorized
	session.polling = false
	return pollResultFromSession(session), nil
}

func (s *GitHubCopilotOAuthService) validateCopilotEntitlement(ctx context.Context, proxy *Proxy, accessToken string) error {
	account := &Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Extra: map[string]any{
			UpstreamProviderExtraKey: UpstreamProviderGitHubCopilot,
		},
		Proxy: proxy,
	}
	if _, err := exchangeGitHubCopilotTokenWithSource(
		ctx,
		s.httpUpstream,
		account,
		accessToken,
		githubCopilotTokenExchangeURL,
	); err != nil {
		// The lower-level exchange error can contain an upstream response body.
		// Never wrap it because that body must not reach the browser or logs.
		var sessionErr *githubCopilotSessionError
		if errors.As(err, &sessionErr) {
			if sessionErr.statusCode == http.StatusUnauthorized {
				return ErrGitHubCopilotOAuthEntitlementRequired
			}
			if sessionErr.statusCode == http.StatusForbidden && !isGitHubRateLimitResponse(sessionErr.responseHeaders, sessionErr.responseBody) {
				return ErrGitHubCopilotOAuthEntitlementRequired
			}
		}
		return ErrGitHubCopilotOAuthUnavailable
	}
	return nil
}

func isGitHubRateLimitResponse(headers http.Header, body []byte) bool {
	if headers != nil && (strings.TrimSpace(headers.Get("Retry-After")) != "" ||
		strings.TrimSpace(headers.Get("X-RateLimit-Remaining")) == "0") {
		return true
	}

	// GitHub's secondary rate limit may return 403 without either header. The
	// response is already size-bounded by the token exchange, and is inspected
	// only in memory so upstream details never reach the browser or logs.
	message := strings.ToLower(string(body))
	return strings.Contains(message, "rate limit") ||
		strings.Contains(message, "rate-limit") ||
		strings.Contains(message, "abuse detection")
}

func (s *GitHubCopilotOAuthService) scheduleCleanup(sessionID string, cleanupAt time.Time) {
	delay := cleanupAt.Sub(s.now())
	if delay < 0 {
		delay = 0
	}
	time.AfterFunc(delay, func() {
		now := s.now()
		s.mu.Lock()
		defer s.mu.Unlock()
		session, ok := s.sessions[sessionID]
		if !ok || now.Before(session.expiresAt.Add(githubCopilotExpiredRetention)) {
			return
		}
		session.clearSecrets()
		delete(s.sessions, sessionID)
	})
}

func (s *GitHubCopilotOAuthService) markSessionError(adminID int64, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session, ok := s.sessions[sessionID]; ok && session.adminID == adminID {
		session.status = GitHubCopilotOAuthStatusError
		session.polling = false
		session.clearSecrets()
	}
}

func (s *GitHubCopilotOAuthService) clearPolling(adminID int64, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session, ok := s.sessions[sessionID]; ok && session.adminID == adminID {
		session.polling = false
	}
}

func (s *GitHubCopilotOAuthService) sessionForAdminLocked(adminID int64, sessionID string) (*githubCopilotOAuthSession, error) {
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrGitHubCopilotOAuthSessionNotFound
	}
	if session.adminID != adminID {
		return nil, ErrGitHubCopilotOAuthSessionForbidden
	}
	return session, nil
}

func (s *GitHubCopilotOAuthService) cleanupLocked(now time.Time) {
	for sessionID, session := range s.sessions {
		if now.Before(session.expiresAt.Add(githubCopilotExpiredRetention)) {
			continue
		}
		session.clearSecrets()
		delete(s.sessions, sessionID)
	}
}

func pollResultFromSession(session *githubCopilotOAuthSession) *GitHubCopilotDevicePollResult {
	result := &GitHubCopilotDevicePollResult{
		Status:   session.status,
		Interval: int64(session.interval / time.Second),
	}
	if session.status == GitHubCopilotOAuthStatusAuthorized {
		result.GitHubLogin = session.githubLogin
		result.GitHubUserID = session.githubUserID
	}
	return result
}

func ceilDurationSeconds(remaining, fallback time.Duration) int64 {
	if remaining <= 0 {
		remaining = fallback
	}
	seconds := (remaining + time.Second - 1) / time.Second
	if seconds < 1 {
		return 1
	}
	return int64(seconds)
}

func setGitHubCopilotOAuthHeaders(req *http.Request, form bool) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sub2api-github-copilot-oauth")
	if form {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if req.URL.Host == "api.github.com" {
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
}

func validateGitHubDeviceCodeResponse(response *githubDeviceCodeResponse) error {
	if response == nil || strings.TrimSpace(response.DeviceCode) == "" || strings.TrimSpace(response.UserCode) == "" || response.ExpiresIn <= 0 {
		return ErrGitHubCopilotOAuthInvalidResponse
	}
	if !isOfficialGitHubVerificationURL(response.VerificationURI) {
		return ErrGitHubCopilotOAuthInvalidResponse
	}
	if response.VerificationURIComplete != "" && !isOfficialGitHubVerificationURL(response.VerificationURIComplete) {
		return ErrGitHubCopilotOAuthInvalidResponse
	}
	return nil
}

func isOfficialGitHubVerificationURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && parsed.Scheme == "https" && strings.EqualFold(parsed.Hostname(), "github.com")
}

func newGitHubCopilotOAuthSessionID() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

type githubDeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
}

type githubAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
}

type githubUserResponse struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
}
