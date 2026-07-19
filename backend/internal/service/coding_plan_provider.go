package service

import (
	"maps"
	"net/url"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
)

const (
	UpstreamProviderExtraKey = "upstream_provider"

	UpstreamProviderGLMCodingPlan  = "glm_coding_plan"
	UpstreamProviderKimiCodingPlan = "kimi_coding_plan"
	UpstreamProviderGitHubCopilot  = "github_copilot"

	GLMCodingPlanBaseURL  = "https://open.bigmodel.cn/api/coding/paas/v4"
	KimiCodingPlanBaseURL = "https://api.kimi.com/coding/v1"
)

var codingPlanDefaultModelMappings = map[string]map[string]string{
	UpstreamProviderGLMCodingPlan: {
		"glm-5.2":     "GLM-5.2",
		"GLM-5.2":     "GLM-5.2",
		"glm-5-turbo": "GLM-5-Turbo",
		"GLM-5-Turbo": "GLM-5-Turbo",
		"glm-4.7":     "GLM-4.7",
		"GLM-4.7":     "GLM-4.7",
	},
	UpstreamProviderKimiCodingPlan: {
		"k3":                        "k3",
		"kimi-for-coding":           "kimi-for-coding",
		"kimi-for-coding-highspeed": "kimi-for-coding-highspeed",
	},
	UpstreamProviderGitHubCopilot: {
		"claude-sonnet-4.6":      "claude-sonnet-4.6",
		"claude-sonnet-4-6":      "claude-sonnet-4.6",
		"claude-haiku-4.5":       "claude-haiku-4.5",
		"claude-haiku-4-5":       "claude-haiku-4.5",
		"gpt-5.4":                "gpt-5.4",
		"gpt-5.3-codex":          "gpt-5.3-codex",
		"gemini-3.1-pro-preview": "gemini-3.1-pro-preview",
		"gemini-3.5-flash":       "gemini-3.5-flash",
		"mai-code-1-flash":       "mai-code-1-flash",
	},
}

func (a *Account) UpstreamProvider() string {
	if a == nil || a.Platform != PlatformOpenAI || a.Type != AccountTypeAPIKey {
		return ""
	}
	return normalizeUpstreamProvider(a.GetExtraString(UpstreamProviderExtraKey))
}

func (a *Account) IsGLMCodingPlan() bool {
	return a != nil && a.UpstreamProvider() == UpstreamProviderGLMCodingPlan
}

func (a *Account) IsKimiCodingPlan() bool {
	return a != nil && a.UpstreamProvider() == UpstreamProviderKimiCodingPlan
}

func (a *Account) IsGitHubCopilot() bool {
	return a != nil && a.UpstreamProvider() == UpstreamProviderGitHubCopilot
}

func (a *Account) IsCodingPlanProvider() bool {
	return a != nil && isKnownCodingPlanProvider(a.UpstreamProvider())
}

func normalizeUpstreamProvider(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isKnownCodingPlanProvider(provider string) bool {
	switch provider {
	case UpstreamProviderGLMCodingPlan, UpstreamProviderKimiCodingPlan, UpstreamProviderGitHubCopilot:
		return true
	default:
		return false
	}
}

// normalizeCodingPlanAccount enforces the transport contract behind the branded
// Coding Plan entries while keeping them in the existing OpenAI protocol family.
func normalizeCodingPlanAccount(
	platform string,
	accountType string,
	credentials map[string]any,
	extra map[string]any,
) (map[string]any, map[string]any, error) {
	rawProvider, exists := extra[UpstreamProviderExtraKey]
	if !exists {
		return credentials, extra, nil
	}
	providerValue, ok := rawProvider.(string)
	if !ok {
		return nil, nil, infraerrors.BadRequest("UPSTREAM_PROVIDER_INVALID", "upstream_provider must be a string")
	}
	provider := normalizeUpstreamProvider(providerValue)
	if provider == "" {
		return credentials, extra, nil
	}
	if !isKnownCodingPlanProvider(provider) {
		return nil, nil, infraerrors.BadRequest("UPSTREAM_PROVIDER_UNSUPPORTED", "unsupported upstream_provider")
	}
	if platform != PlatformOpenAI || accountType != AccountTypeAPIKey {
		return nil, nil, infraerrors.BadRequest(
			"CODING_PLAN_ACCOUNT_TYPE_INVALID",
			"coding plan providers require platform=openai and type=apikey",
		)
	}

	normalizedCredentials := maps.Clone(credentials)
	if normalizedCredentials == nil {
		normalizedCredentials = make(map[string]any)
	}
	apiKey, _ := normalizedCredentials["api_key"].(string)
	if strings.TrimSpace(apiKey) == "" {
		return nil, nil, infraerrors.BadRequest("CODING_PLAN_CREDENTIAL_REQUIRED", "coding plan credential is required")
	}

	switch provider {
	case UpstreamProviderGLMCodingPlan:
		if err := normalizeOfficialCodingPlanBaseURL(normalizedCredentials, GLMCodingPlanBaseURL); err != nil {
			return nil, nil, err
		}
	case UpstreamProviderKimiCodingPlan:
		if err := normalizeOfficialCodingPlanBaseURL(normalizedCredentials, KimiCodingPlanBaseURL); err != nil {
			return nil, nil, err
		}
	case UpstreamProviderGitHubCopilot:
		// The configured secret is a GitHub token, not an OpenAI API key. The
		// runtime exchanges it and obtains the API endpoint from GitHub.
		delete(normalizedCredentials, "base_url")
	}

	if !hasNonEmptyModelMapping(normalizedCredentials["model_mapping"]) {
		normalizedCredentials["model_mapping"] = cloneStringMapAsAny(codingPlanDefaultModelMappings[provider])
	}
	// Coding Plan providers expose Chat Completions, not the full OpenAI API.
	normalizedCredentials[openAIEndpointCapabilitiesCredentialKey] = []string{string(OpenAIEndpointCapabilityChatCompletions)}
	delete(normalizedCredentials, "compact_model_mapping")

	normalizedExtra := maps.Clone(extra)
	if normalizedExtra == nil {
		normalizedExtra = make(map[string]any)
	}
	normalizedExtra[UpstreamProviderExtraKey] = provider
	normalizedExtra[openai_compat.ExtraKeyResponsesMode] = string(openai_compat.ResponsesSupportModeForceChatCompletions)
	normalizedExtra[openai_compat.ExtraKeyResponsesSupported] = false
	normalizedExtra["openai_compact_mode"] = OpenAICompactModeForceOff
	normalizedExtra["openai_compact_supported"] = false
	normalizedExtra["openai_ws_force_http"] = true
	normalizedExtra["openai_apikey_responses_websockets_v2_mode"] = OpenAIWSIngressModeOff
	normalizedExtra["openai_apikey_responses_websockets_v2_enabled"] = false
	normalizedExtra["openai_oauth_responses_websockets_v2_mode"] = OpenAIWSIngressModeOff
	normalizedExtra["openai_oauth_responses_websockets_v2_enabled"] = false
	normalizedExtra["responses_websockets_v2_enabled"] = false
	normalizedExtra["openai_ws_enabled"] = false
	return normalizedCredentials, normalizedExtra, nil
}

func preserveCodingPlanProviderOnUpdate(account *Account, extra map[string]any) error {
	if account == nil || !account.IsCodingPlanProvider() || extra == nil {
		return nil
	}
	current := account.UpstreamProvider()
	if raw, exists := extra[UpstreamProviderExtraKey]; exists {
		incoming, ok := raw.(string)
		if !ok || normalizeUpstreamProvider(incoming) != current {
			return infraerrors.BadRequest(
				"CODING_PLAN_PROVIDER_IMMUTABLE",
				"upstream_provider cannot be changed on an existing coding plan account",
			)
		}
		return nil
	}
	extra[UpstreamProviderExtraKey] = current
	return nil
}

func validateCodingPlanExtraPatch(account *Account, updates map[string]any) error {
	if account == nil || len(updates) == 0 {
		return nil
	}
	if raw, exists := updates[UpstreamProviderExtraKey]; exists {
		provider, ok := raw.(string)
		if !ok || account.UpstreamProvider() == "" || normalizeUpstreamProvider(provider) != account.UpstreamProvider() {
			return infraerrors.BadRequest(
				"CODING_PLAN_PROVIDER_IMMUTABLE",
				"upstream_provider cannot be changed through an extra patch",
			)
		}
	}
	if account.IsCodingPlanProvider() {
		for key := range codingPlanImmutableProtocolExtraKeys {
			if _, exists := updates[key]; exists {
				return infraerrors.BadRequest(
					"CODING_PLAN_PROTOCOL_SETTING_IMMUTABLE",
					"coding plan protocol capabilities cannot be changed through an extra patch",
				)
			}
		}
		if raw, exists := updates[openai_compat.ExtraKeyResponsesMode]; exists {
			mode, ok := raw.(string)
			if !ok || openai_compat.NormalizeResponsesSupportMode(mode) != openai_compat.ResponsesSupportModeForceChatCompletions {
				return infraerrors.BadRequest(
					"CODING_PLAN_RESPONSES_MODE_INVALID",
					"coding plan providers must use force_chat_completions",
				)
			}
		}
	}
	return nil
}

var codingPlanImmutableProtocolExtraKeys = map[string]struct{}{
	openai_compat.ExtraKeyResponsesSupported:        {},
	"openai_compact_mode":                           {},
	"openai_compact_supported":                      {},
	"openai_ws_force_http":                          {},
	"openai_apikey_responses_websockets_v2_mode":    {},
	"openai_apikey_responses_websockets_v2_enabled": {},
	"openai_oauth_responses_websockets_v2_mode":     {},
	"openai_oauth_responses_websockets_v2_enabled":  {},
	"responses_websockets_v2_enabled":               {},
	"openai_ws_enabled":                             {},
}

func codingPlanExtraPatchRequiresValidation(updates map[string]any) bool {
	if len(updates) == 0 {
		return false
	}
	if _, exists := updates[UpstreamProviderExtraKey]; exists {
		return true
	}
	if _, exists := updates[openai_compat.ExtraKeyResponsesMode]; exists {
		return true
	}
	for key := range codingPlanImmutableProtocolExtraKeys {
		if _, exists := updates[key]; exists {
			return true
		}
	}
	return false
}

func normalizeOfficialCodingPlanBaseURL(credentials map[string]any, expected string) error {
	raw, _ := credentials["base_url"].(string)
	if strings.TrimSpace(raw) == "" {
		credentials["base_url"] = expected
		return nil
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.Opaque != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return infraerrors.BadRequest("CODING_PLAN_BASE_URL_INVALID", "invalid coding plan base_url")
	}
	expectedURL, _ := url.Parse(expected)
	if !strings.EqualFold(parsed.Scheme, "https") ||
		!strings.EqualFold(parsed.Hostname(), expectedURL.Hostname()) ||
		(parsed.Port() != "" && parsed.Port() != "443") ||
		strings.TrimRight(parsed.Path, "/") != strings.TrimRight(expectedURL.Path, "/") {
		return infraerrors.BadRequest("CODING_PLAN_BASE_URL_INVALID", "coding plan base_url must use the official coding endpoint")
	}
	credentials["base_url"] = expected
	return nil
}

func hasNonEmptyModelMapping(raw any) bool {
	switch mapping := raw.(type) {
	case map[string]any:
		return len(mapping) > 0
	case map[string]string:
		return len(mapping) > 0
	default:
		return false
	}
}

func cloneStringMapAsAny(source map[string]string) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
