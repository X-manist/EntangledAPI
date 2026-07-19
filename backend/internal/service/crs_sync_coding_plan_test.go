//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestCRSSyncRejectsCodingPlanAccountTypeCollisions(t *testing.T) {
	tests := []struct {
		name        string
		collection  string
		credentials map[string]any
		extra       map[string]any
	}{
		{
			name:        "Claude OAuth",
			collection:  "claudeAccounts",
			credentials: map[string]any{"access_token": "claude-token"},
			extra:       map[string]any{UpstreamProviderExtraKey: UpstreamProviderGLMCodingPlan},
		},
		{
			name:        "Claude API key",
			collection:  "claudeConsoleAccounts",
			credentials: map[string]any{"api_key": "claude-key"},
		},
		{
			name:        "OpenAI OAuth",
			collection:  "openaiOAuthAccounts",
			credentials: map[string]any{"access_token": "openai-token"},
			extra:       map[string]any{UpstreamProviderExtraKey: UpstreamProviderGLMCodingPlan},
		},
		{
			name:        "Gemini OAuth",
			collection:  "geminiOAuthAccounts",
			credentials: map[string]any{"refresh_token": "gemini-refresh"},
			extra:       map[string]any{UpstreamProviderExtraKey: UpstreamProviderGLMCodingPlan},
		},
		{
			name:        "Gemini API key",
			collection:  "geminiApiKeyAccounts",
			credentials: map[string]any{"api_key": "gemini-key"},
			extra:       map[string]any{UpstreamProviderExtraKey: UpstreamProviderGLMCodingPlan},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := newCRSCodingPlanAccount()
			repo := newCRSLongContextAccountRepo(existing)

			result := runCRSCodingPlanSync(t, repo, tt.collection, tt.credentials, tt.extra)

			require.Len(t, result.Items, 1)
			require.Equal(t, "failed", result.Items[0].Action)
			require.Contains(t, result.Items[0].Error, "coding plan")
			require.Equal(t, 0, result.Updated)
			require.Equal(t, PlatformOpenAI, existing.Platform)
			require.Equal(t, AccountTypeAPIKey, existing.Type)
			require.Equal(t, UpstreamProviderGLMCodingPlan, existing.Extra[UpstreamProviderExtraKey])
			require.Equal(t, "old-coding-plan-key", existing.Credentials["api_key"])
		})
	}
}

func TestCRSSyncRejectsIncomingCodingPlanProviderOutsideOpenAIAPIKey(t *testing.T) {
	tests := []struct {
		name        string
		collection  string
		credentials map[string]any
	}{
		{
			name:        "Claude OAuth",
			collection:  "claudeAccounts",
			credentials: map[string]any{"access_token": "claude-token"},
		},
		{
			name:        "OpenAI OAuth",
			collection:  "openaiOAuthAccounts",
			credentials: map[string]any{"access_token": "openai-token"},
		},
		{
			name:        "Gemini OAuth",
			collection:  "geminiOAuthAccounts",
			credentials: map[string]any{"refresh_token": "gemini-refresh"},
		},
		{
			name:        "Gemini API key",
			collection:  "geminiApiKeyAccounts",
			credentials: map[string]any{"api_key": "gemini-key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newCRSLongContextAccountRepo()
			result := runCRSCodingPlanSync(t, repo, tt.collection, tt.credentials, map[string]any{
				UpstreamProviderExtraKey: UpstreamProviderKimiCodingPlan,
			})

			require.Len(t, result.Items, 1)
			require.Equal(t, "failed", result.Items[0].Action)
			require.Contains(t, result.Items[0].Error, "coding plan providers require platform=openai and type=apikey")
			require.Equal(t, 0, result.Created)
			require.Empty(t, repo.accounts)
		})
	}
}

func TestCRSSyncOpenAIAPIKeyPreservesAndNormalizesCodingPlan(t *testing.T) {
	existing := newCRSCodingPlanAccount()
	repo := newCRSLongContextAccountRepo(existing)

	result := runCRSCodingPlanSync(t, repo, "openaiResponsesAccounts", map[string]any{
		"api_key":  "new-coding-plan-key",
		"base_url": GLMCodingPlanBaseURL,
	}, map[string]any{
		UpstreamProviderExtraKey: UpstreamProviderGLMCodingPlan,
	})

	require.Len(t, result.Items, 1)
	require.Equal(t, "updated", result.Items[0].Action)
	require.Equal(t, 1, result.Updated)
	require.Equal(t, PlatformOpenAI, existing.Platform)
	require.Equal(t, AccountTypeAPIKey, existing.Type)
	require.Equal(t, UpstreamProviderGLMCodingPlan, existing.Extra[UpstreamProviderExtraKey])
	require.Equal(t, "new-coding-plan-key", existing.Credentials["api_key"])
	require.Equal(t, GLMCodingPlanBaseURL, existing.Credentials["base_url"])
	require.NotEmpty(t, existing.Credentials["model_mapping"])
	require.Equal(t,
		[]string{string(OpenAIEndpointCapabilityChatCompletions)},
		existing.Credentials[openAIEndpointCapabilitiesCredentialKey],
	)
}

func TestCRSSyncOpenAIAPIKeyRejectsImplicitCodingPlanOverwrite(t *testing.T) {
	existing := newCRSCodingPlanAccount()
	repo := newCRSLongContextAccountRepo(existing)

	result := runCRSCodingPlanSync(t, repo, "openaiResponsesAccounts", map[string]any{
		"api_key":  "ordinary-openai-key",
		"base_url": "https://api.openai.com/v1",
	}, nil)

	require.Len(t, result.Items, 1)
	require.Equal(t, "failed", result.Items[0].Action)
	require.Contains(t, result.Items[0].Error, "must explicitly declare")
	require.Equal(t, 0, result.Updated)
	require.Equal(t, "old-coding-plan-key", existing.Credentials["api_key"])
	require.Equal(t, GLMCodingPlanBaseURL, existing.Credentials["base_url"])
	require.Equal(t, UpstreamProviderGLMCodingPlan, existing.Extra[UpstreamProviderExtraKey])
}

func TestCRSSyncOpenAIAPIKeyNormalizesIncomingCodingPlan(t *testing.T) {
	repo := newCRSLongContextAccountRepo()

	result := runCRSCodingPlanSync(t, repo, "openaiResponsesAccounts", map[string]any{
		"api_key": "kimi-coding-plan-key",
	}, map[string]any{
		UpstreamProviderExtraKey: UpstreamProviderKimiCodingPlan,
	})

	require.Len(t, result.Items, 1)
	require.Equal(t, "created", result.Items[0].Action)
	require.Equal(t, 1, result.Created)
	stored := repo.accounts[crsCodingPlanTestID]
	require.NotNil(t, stored)
	require.Equal(t, UpstreamProviderKimiCodingPlan, stored.Extra[UpstreamProviderExtraKey])
	require.Equal(t, KimiCodingPlanBaseURL, stored.Credentials["base_url"])
	require.NotEmpty(t, stored.Credentials["model_mapping"])
}

func TestNormalizeCRSCodingPlanAccountRejectsNilExtraForExistingProvider(t *testing.T) {
	existing := newCRSCodingPlanAccount()

	_, _, err := normalizeCRSCodingPlanAccount(
		existing,
		PlatformAnthropic,
		AccountTypeAPIKey,
		map[string]any{"api_key": "replacement-key"},
		nil,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must explicitly declare")

	_, _, err = normalizeCRSCodingPlanAccount(
		existing,
		PlatformOpenAI,
		AccountTypeAPIKey,
		map[string]any{
			"api_key":  "replacement-key",
			"base_url": GLMCodingPlanBaseURL,
		},
		nil,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must explicitly declare")
}

const crsCodingPlanTestID = "crs-coding-plan-1"

func newCRSCodingPlanAccount() *Account {
	return &Account{
		ID:       51,
		Name:     "Existing Coding Plan",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "old-coding-plan-key",
			"base_url": GLMCodingPlanBaseURL,
		},
		Extra: map[string]any{
			"crs_account_id":         crsCodingPlanTestID,
			UpstreamProviderExtraKey: UpstreamProviderGLMCodingPlan,
		},
	}
}

func runCRSCodingPlanSync(
	t *testing.T,
	repo AccountRepository,
	collection string,
	credentials map[string]any,
	extra map[string]any,
) *SyncFromCRSResult {
	t.Helper()
	account := map[string]any{
		"kind":               collection,
		"id":                 crsCodingPlanTestID,
		"name":               "CRS Coding Plan Test",
		"authType":           "oauth",
		"isActive":           true,
		"schedulable":        true,
		"maxConcurrentTasks": 3,
		"credentials":        credentials,
	}
	if extra != nil {
		account["extra"] = extra
	}

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/web/auth/login" {
			_, _ = response.Write([]byte(`{"success":true,"token":"admin-token"}`))
			return
		}
		require.Equal(t, "/admin/sync/export-accounts", request.URL.Path)
		require.NoError(t, json.NewEncoder(response).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{collection: []any{account}},
		}))
	}))
	t.Cleanup(server.Close)

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	service := NewCRSSyncService(repo, nil, nil, nil, nil, cfg)
	result, err := service.SyncFromCRS(context.Background(), SyncFromCRSInput{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "password",
	})
	require.NoError(t, err)
	return result
}
