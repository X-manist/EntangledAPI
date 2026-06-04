package copilot

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL   = "https://api.githubcopilot.com"
	TokenExchangeURL = "https://api.github.com/copilot_internal/v2/token"
	DefaultUserAgent = "GitHubCopilotChat/0.26.7"
	GitHubAPIVersion = "2022-11-28"
)

type Model struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
}

var DefaultModels = []Model{
	{ID: "gpt-4o", Object: "model", Type: "model", DisplayName: "GPT-4o"},
	{ID: "gpt-4.1", Object: "model", Type: "model", DisplayName: "GPT-4.1"},
	{ID: "gpt-4.1-mini", Object: "model", Type: "model", DisplayName: "GPT-4.1 mini"},
	{ID: "o3", Object: "model", Type: "model", DisplayName: "o3"},
	{ID: "o4-mini", Object: "model", Type: "model", DisplayName: "o4-mini"},
	{ID: "claude-3.7-sonnet", Object: "model", Type: "model", DisplayName: "Claude 3.7 Sonnet"},
	{ID: "claude-sonnet-4", Object: "model", Type: "model", DisplayName: "Claude Sonnet 4"},
	{ID: "gemini-2.5-pro", Object: "model", Type: "model", DisplayName: "Gemini 2.5 Pro"},
}

type TokenResponse struct {
	Token     string       `json:"token"`
	ExpiresAt FlexibleTime `json:"expires_at"`
	RefreshIn int64        `json:"refresh_in"`
	SKU       string       `json:"sku"`
	Endpoints []string     `json:"endpoints"`
}

func (r TokenResponse) ParsedExpiry(now time.Time) *time.Time {
	if t := r.ExpiresAt.Time(); t != nil {
		return t
	}
	if r.RefreshIn > 0 {
		t := now.Add(time.Duration(r.RefreshIn) * time.Second)
		return &t
	}
	return nil
}

type FlexibleTime struct {
	value *time.Time
}

func (t *FlexibleTime) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		t.value = nil
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		return t.setFromString(text)
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return err
	}
	parsed := time.Unix(seconds, 0).UTC()
	t.value = &parsed
	return nil
}

func (t *FlexibleTime) setFromString(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		t.value = nil
		return nil
	}
	if seconds, err := strconv.ParseInt(text, 10, 64); err == nil {
		parsed := time.Unix(seconds, 0).UTC()
		t.value = &parsed
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return err
	}
	parsed = parsed.UTC()
	t.value = &parsed
	return nil
}

func (t FlexibleTime) Time() *time.Time {
	if t.value == nil {
		return nil
	}
	parsed := *t.value
	return &parsed
}

func LooksLikePersonalAccessToken(token string) bool {
	token = strings.TrimSpace(token)
	return strings.HasPrefix(token, "ghp_") || strings.HasPrefix(token, "github_pat_")
}
