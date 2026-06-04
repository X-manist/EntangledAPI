package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/copilot"
)

type openAICompatiblePlatformContextKey struct{}

func WithOpenAICompatiblePlatform(ctx context.Context, platform string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openAICompatiblePlatformContextKey{}, normalizeOpenAICompatiblePlatform(platform))
}

func normalizeOpenAICompatiblePlatform(platform string) string {
	switch strings.TrimSpace(platform) {
	case PlatformGitHubCopilot:
		return PlatformGitHubCopilot
	default:
		return PlatformOpenAI
	}
}

func OpenAICompatiblePlatformFromContext(ctx context.Context) string {
	if ctx == nil {
		return PlatformOpenAI
	}
	if v, ok := ctx.Value(openAICompatiblePlatformContextKey{}).(string); ok {
		return normalizeOpenAICompatiblePlatform(v)
	}
	return PlatformOpenAI
}

func IsOpenAICompatiblePlatform(platform string) bool {
	switch strings.TrimSpace(platform) {
	case PlatformOpenAI, PlatformGitHubCopilot:
		return true
	default:
		return false
	}
}

func defaultOpenAICompatibleBaseURL(account *Account) string {
	if account == nil {
		return "https://api.openai.com"
	}
	if account.IsGitHubCopilot() {
		return copilot.DefaultBaseURL
	}
	return "https://api.openai.com"
}

func applyGitHubCopilotHeaders(req *http.Request, account *Account, accept string) {
	if req == nil {
		return
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", copilot.DefaultUserAgent)
	}
	req.Header.Set("Editor-Version", "vscode/1.99.3")
	req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.26.7")
	req.Header.Set("Copilot-Integration-Id", "vscode-chat")
	if account != nil {
		if sku := strings.TrimSpace(account.GetCredential("copilot_sku")); sku != "" {
			req.Header.Set("X-Copilot-Sku", sku)
		}
	}
}

func (s *OpenAIGatewayService) ensureGitHubCopilotAccessToken(ctx context.Context, account *Account) (string, error) {
	if account == nil || !account.IsGitHubCopilot() {
		return "", errors.New("account is not github copilot")
	}
	switch account.Type {
	case AccountTypeAPIKey:
		token := strings.TrimSpace(account.GetGitHubCopilotAPIKey())
		if token == "" {
			return "", errors.New("api_key not found in credentials")
		}
		return token, nil
	case AccountTypeOAuth:
		if token := strings.TrimSpace(account.GetGitHubCopilotAccessToken()); token != "" && !account.IsGitHubCopilotTokenExpired() {
			return token, nil
		}
		githubToken := strings.TrimSpace(account.GetGitHubCopilotGitHubToken())
		if githubToken == "" {
			if token := strings.TrimSpace(account.GetGitHubCopilotAccessToken()); token != "" {
				return token, nil
			}
			return "", errors.New("github_token not found in credentials")
		}
		tokenResp, err := s.fetchGitHubCopilotToken(ctx, account, githubToken)
		if err != nil {
			return "", err
		}
		if tokenResp.Token == "" {
			return "", errors.New("copilot token exchange returned empty token")
		}
		if s.accountRepo != nil {
			if account.Credentials == nil {
				account.Credentials = map[string]any{}
			}
			account.Credentials["access_token"] = tokenResp.Token
			if tokenResp.SKU != "" {
				account.Credentials["copilot_sku"] = tokenResp.SKU
			}
			if expiresAt := tokenResp.ParsedExpiry(time.Now()); expiresAt != nil {
				account.Credentials["expires_at"] = expiresAt.UTC().Format(time.RFC3339)
			}
			if err := s.accountRepo.Update(ctx, account); err != nil {
				return "", fmt.Errorf("persist copilot access token: %w", err)
			}
		}
		return tokenResp.Token, nil
	default:
		return "", fmt.Errorf("unsupported github copilot account type: %s", account.Type)
	}
}

func (s *OpenAIGatewayService) fetchGitHubCopilotToken(ctx context.Context, account *Account, githubToken string) (*copilot.TokenResponse, error) {
	if s == nil || s.httpUpstream == nil {
		return nil, errors.New("upstream http client is not configured")
	}
	if copilot.LooksLikePersonalAccessToken(githubToken) {
		return nil, errors.New("GitHub Copilot token exchange does not accept GitHub personal access tokens (ghp_/github_pat_); use a GitHub OAuth token from Copilot sign-in (usually ghu_/gho_) or store a Copilot session token as API Key")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, copilot.TokenExchangeURL, bytes.NewReader(nil))
	if err != nil {
		return nil, fmt.Errorf("build copilot token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("X-GitHub-Api-Version", copilot.GitHubAPIVersion)
	req.Header.Set("User-Agent", copilot.DefaultUserAgent)
	applyGitHubCopilotHeaders(req, account, "application/json")
	proxyURL := ""
	if account != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, fmt.Errorf("request copilot token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("copilot token exchange failed with HTTP %d", resp.StatusCode)
	}
	var tokenResp copilot.TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode copilot token response: %w", err)
	}
	return &tokenResp, nil
}

func (s *AccountTestService) ensureGitHubCopilotAccessToken(ctx context.Context, account *Account) (string, error) {
	gateway := &OpenAIGatewayService{
		accountRepo:  accountRepoOrNil(s),
		httpUpstream: s.httpUpstream,
	}
	return gateway.ensureGitHubCopilotAccessToken(ctx, account)
}

func accountRepoOrNil(s *AccountTestService) AccountRepository {
	if s == nil {
		return nil
	}
	return s.accountRepo
}
