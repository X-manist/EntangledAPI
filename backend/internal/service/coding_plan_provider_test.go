package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/stretchr/testify/require"
)

func TestNormalizeCodingPlanAccountAppliesGLMContract(t *testing.T) {
	credentials, extra, err := normalizeCodingPlanAccount(
		PlatformOpenAI,
		AccountTypeAPIKey,
		map[string]any{"api_key": "glm-secret"},
		map[string]any{UpstreamProviderExtraKey: " GLM_CODING_PLAN "},
	)
	require.NoError(t, err)
	require.Equal(t, GLMCodingPlanBaseURL, credentials["base_url"])
	require.Equal(t, []string{string(OpenAIEndpointCapabilityChatCompletions)}, credentials[openAIEndpointCapabilitiesCredentialKey])
	require.NotEmpty(t, credentials["model_mapping"])
	require.Equal(t, UpstreamProviderGLMCodingPlan, extra[UpstreamProviderExtraKey])
	require.Equal(t, string(openai_compat.ResponsesSupportModeForceChatCompletions), extra[openai_compat.ExtraKeyResponsesMode])
	require.Equal(t, false, extra[openai_compat.ExtraKeyResponsesSupported])
	require.Equal(t, OpenAICompactModeForceOff, extra["openai_compact_mode"])
	require.Equal(t, false, extra["openai_compact_supported"])
	require.Equal(t, true, extra["openai_ws_force_http"])
	require.Equal(t, OpenAIWSIngressModeOff, extra["openai_apikey_responses_websockets_v2_mode"])
	require.Equal(t, false, extra["openai_apikey_responses_websockets_v2_enabled"])
	require.Equal(t, false, extra["responses_websockets_v2_enabled"])
}

func TestNormalizeCodingPlanAccountRejectsUnofficialBaseURL(t *testing.T) {
	_, _, err := normalizeCodingPlanAccount(
		PlatformOpenAI,
		AccountTypeAPIKey,
		map[string]any{"api_key": "secret", "base_url": "https://example.com/v1"},
		map[string]any{UpstreamProviderExtraKey: UpstreamProviderKimiCodingPlan},
	)
	require.Error(t, err)
}

func TestNormalizeCodingPlanAccountKeepsCustomModelMapping(t *testing.T) {
	customMapping := map[string]any{"my-model": "upstream-model"}
	credentials, _, err := normalizeCodingPlanAccount(
		PlatformOpenAI,
		AccountTypeAPIKey,
		map[string]any{
			"api_key":       "secret",
			"model_mapping": customMapping,
		},
		map[string]any{UpstreamProviderExtraKey: UpstreamProviderKimiCodingPlan},
	)
	require.NoError(t, err)
	require.Equal(t, customMapping, credentials["model_mapping"])
}

func TestNormalizeCopilotAccountDoesNotExposeGitHubTokenAsOpenAIAPIKey(t *testing.T) {
	credentials, extra, err := normalizeCodingPlanAccount(
		PlatformOpenAI,
		AccountTypeAPIKey,
		map[string]any{"api_key": "github-token", "base_url": "https://api.openai.com"},
		map[string]any{UpstreamProviderExtraKey: UpstreamProviderGitHubCopilot},
	)
	require.NoError(t, err)
	require.NotContains(t, credentials, "base_url")

	account := &Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: credentials,
		Extra:       extra,
	}
	require.True(t, account.IsGitHubCopilot())
	require.Empty(t, account.GetOpenAIApiKey())
	require.Equal(t, "github-token", account.GetCredential("api_key"))
}

func TestPreserveCodingPlanProviderOnUpdate(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra:    map[string]any{UpstreamProviderExtraKey: UpstreamProviderGLMCodingPlan},
	}
	extra := map[string]any{}
	require.NoError(t, preserveCodingPlanProviderOnUpdate(account, extra))
	require.Equal(t, UpstreamProviderGLMCodingPlan, extra[UpstreamProviderExtraKey])

	err := preserveCodingPlanProviderOnUpdate(account, map[string]any{
		UpstreamProviderExtraKey: UpstreamProviderKimiCodingPlan,
	})
	require.Error(t, err)
}

func TestCodingPlanProviderExcludesNativeOpenAIOnlyEndpoints(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra:    map[string]any{UpstreamProviderExtraKey: UpstreamProviderGitHubCopilot},
	}

	require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions))
	require.False(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityAlphaSearch))
	require.False(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityResponsesWebSocket))
	_, err := (&OpenAIGatewayService{}).buildOpenAIResponsesWSURL(account)
	require.ErrorContains(t, err, "not supported by coding plan providers")
}

func TestUpstreamProviderRequiresOpenAIAPIKeySubtype(t *testing.T) {
	for _, account := range []*Account{
		{Platform: PlatformOpenAI, Type: AccountTypeOAuth},
		{Platform: PlatformAnthropic, Type: AccountTypeAPIKey},
	} {
		account.Extra = map[string]any{UpstreamProviderExtraKey: UpstreamProviderGLMCodingPlan}
		require.Empty(t, account.UpstreamProvider())
		require.False(t, account.IsCodingPlanProvider())
	}
}

func TestCodingPlanChatTargetIgnoresStoredBaseURL(t *testing.T) {
	service := &OpenAIGatewayService{}
	for _, tc := range []struct {
		provider string
		baseURL  string
	}{
		{provider: UpstreamProviderGLMCodingPlan, baseURL: GLMCodingPlanBaseURL},
		{provider: UpstreamProviderKimiCodingPlan, baseURL: KimiCodingPlanBaseURL},
	} {
		account := &Account{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "secret", "base_url": "https://attacker.invalid/v1"},
			Extra:       map[string]any{UpstreamProviderExtraKey: tc.provider},
		}
		require.Equal(t, tc.baseURL, account.GetOpenAIBaseURL())
		targetURL, err := service.openAIChatCompletionsTargetURL(account)
		require.NoError(t, err)
		require.Equal(t, buildOpenAIChatCompletionsURL(tc.baseURL), targetURL)
		require.NotContains(t, targetURL, "attacker.invalid")
	}
}

func TestCodingPlanCompactSupportIsAlwaysKnownUnsupported(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			UpstreamProviderExtraKey:   UpstreamProviderKimiCodingPlan,
			"openai_compact_mode":      OpenAICompactModeForceOn,
			"openai_compact_supported": true,
		},
	}

	supported, known := account.OpenAICompactSupportKnown()
	require.False(t, supported)
	require.True(t, known)
	require.False(t, account.AllowsOpenAICompact())
}

func TestValidateCodingPlanExtraPatchRejectsCompactAndWebSocketSettings(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra:    map[string]any{UpstreamProviderExtraKey: UpstreamProviderGitHubCopilot},
	}
	for _, key := range []string{
		"openai_compact_mode",
		"openai_compact_supported",
		"openai_ws_force_http",
		"openai_apikey_responses_websockets_v2_mode",
		"openai_apikey_responses_websockets_v2_enabled",
		"responses_websockets_v2_enabled",
	} {
		err := validateCodingPlanExtraPatch(account, map[string]any{key: true})
		require.Error(t, err, key)
	}
}
