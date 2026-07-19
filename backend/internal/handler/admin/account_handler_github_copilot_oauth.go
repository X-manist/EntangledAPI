package admin

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const githubCopilotOAuthSessionCredentialKey = "github_copilot_oauth_session_id"

type githubCopilotOAuthCredentialService interface {
	Capabilities(ctx context.Context) (*service.GitHubCopilotOAuthCapabilities, error)
	Start(ctx context.Context, adminID int64, proxyID *int64) (*service.GitHubCopilotDeviceStartResult, error)
	Poll(ctx context.Context, adminID int64, sessionID string) (*service.GitHubCopilotDevicePollResult, error)
	CheckoutCredential(adminID int64, sessionID string) (*service.GitHubCopilotOAuthCredential, error)
	ReleaseCredential(adminID int64, sessionID string) error
	FinalizeCredential(adminID int64, sessionID string) error
}

type githubCopilotDeviceStartRequest struct {
	ProxyID *int64 `json:"proxy_id"`
}

type githubCopilotDevicePollRequest struct {
	OAuthSessionID string `json:"oauth_session_id" binding:"required"`
}

// GetGitHubCopilotOAuthCapabilities reports whether the operator configured an
// OAuth App client ID. Manual PAT authorization is always available.
func (h *AccountHandler) GetGitHubCopilotOAuthCapabilities(c *gin.Context) {
	if h.githubCopilotOAuth == nil {
		response.Success(c, &service.GitHubCopilotOAuthCapabilities{
			Configured: false,
			PATURL:     service.GitHubCopilotPATURL,
		})
		return
	}
	capabilities, err := h.githubCopilotOAuth.Capabilities(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, capabilities)
}

// StartGitHubCopilotOAuth starts a GitHub Device Flow bound to the current
// administrator and optional account proxy.
func (h *AccountHandler) StartGitHubCopilotOAuth(c *gin.Context) {
	adminID := getAdminIDFromContext(c)
	if adminID <= 0 {
		response.Unauthorized(c, "Administrator authentication is required")
		return
	}
	if h.githubCopilotOAuth == nil {
		response.ErrorFrom(c, service.ErrGitHubCopilotOAuthNotConfigured)
		return
	}

	var req githubCopilotDeviceStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.githubCopilotOAuth.Start(c.Request.Context(), adminID, req.ProxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// PollGitHubCopilotOAuth advances one Device Flow polling attempt. The access
// token remains server-side even when authorization succeeds.
func (h *AccountHandler) PollGitHubCopilotOAuth(c *gin.Context) {
	adminID := getAdminIDFromContext(c)
	if adminID <= 0 {
		response.Unauthorized(c, "Administrator authentication is required")
		return
	}
	if h.githubCopilotOAuth == nil {
		response.ErrorFrom(c, service.ErrGitHubCopilotOAuthNotConfigured)
		return
	}

	var req githubCopilotDevicePollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.githubCopilotOAuth.Poll(c.Request.Context(), adminID, req.OAuthSessionID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *AccountHandler) resolveGitHubCopilotOAuthCredential(
	c *gin.Context,
	platform, accountType string,
	credentials, extra map[string]any,
) (map[string]any, string, error) {
	rawSessionID, hasSession := credentials[githubCopilotOAuthSessionCredentialKey]
	if !hasSession {
		return credentials, "", nil
	}

	sessionID, ok := rawSessionID.(string)
	sessionID = strings.TrimSpace(sessionID)
	provider, _ := extra[service.UpstreamProviderExtraKey].(string)
	if !ok || sessionID == "" || platform != service.PlatformOpenAI || accountType != service.AccountTypeAPIKey ||
		!strings.EqualFold(strings.TrimSpace(provider), service.UpstreamProviderGitHubCopilot) {
		return nil, "", infraerrors.BadRequest(
			"GITHUB_COPILOT_OAUTH_CREDENTIAL_INVALID",
			"GitHub authorization can only create a GitHub Copilot account",
		)
	}
	if apiKey, exists := credentials["api_key"]; exists && strings.TrimSpace(stringValue(apiKey)) != "" {
		return nil, "", infraerrors.BadRequest(
			"GITHUB_COPILOT_OAUTH_CREDENTIAL_AMBIGUOUS",
			"choose either GitHub authorization or a manual token",
		)
	}
	if h.githubCopilotOAuth == nil {
		return nil, "", service.ErrGitHubCopilotOAuthNotConfigured
	}
	adminID := getAdminIDFromContext(c)
	if adminID <= 0 {
		return nil, "", infraerrors.Unauthorized(
			"ADMIN_AUTH_REQUIRED",
			"administrator authentication is required",
		)
	}

	credential, err := h.githubCopilotOAuth.CheckoutCredential(adminID, sessionID)
	if err != nil {
		return nil, "", err
	}
	resolved := make(map[string]any, len(credentials))
	for key, value := range credentials {
		if key != githubCopilotOAuthSessionCredentialKey && key != "api_key" {
			resolved[key] = value
		}
	}
	resolved["api_key"] = credential.AccessToken
	return resolved, sessionID, nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
