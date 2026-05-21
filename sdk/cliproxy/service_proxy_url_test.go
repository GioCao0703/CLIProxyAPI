package cliproxy

import (
	"testing"

	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

func TestResolveConfigAPIKeysSyncProxyURLToAuth(t *testing.T) {
	tests := []struct {
		name    string
		service *Service
		auth    *coreauth.Auth
		resolve func(*Service, *coreauth.Auth) bool
		want    string
	}{
		{
			name: "claude",
			service: &Service{cfg: &config.Config{ClaudeKey: []config.ClaudeKey{{
				APIKey:   "claude-key",
				BaseURL:  "https://anthropic.example",
				ProxyURL: "http://claude-proxy.local:8080",
			}}}},
			auth: &coreauth.Auth{Attributes: map[string]string{
				"api_key":  "claude-key",
				"base_url": "https://anthropic.example",
			}},
			resolve: func(service *Service, auth *coreauth.Auth) bool {
				return service.resolveConfigClaudeKey(auth) != nil
			},
			want: "http://claude-proxy.local:8080",
		},
		{
			name: "gemini",
			service: &Service{cfg: &config.Config{GeminiKey: []config.GeminiKey{{
				APIKey:   "gemini-key",
				BaseURL:  "https://generativelanguage.example",
				ProxyURL: "socks5://gemini-proxy.local:1080",
			}}}},
			auth: &coreauth.Auth{Attributes: map[string]string{
				"api_key":  "gemini-key",
				"base_url": "https://generativelanguage.example",
			}},
			resolve: func(service *Service, auth *coreauth.Auth) bool {
				return service.resolveConfigGeminiKey(auth) != nil
			},
			want: "socks5://gemini-proxy.local:1080",
		},
		{
			name: "vertex compat",
			service: &Service{cfg: &config.Config{VertexCompatAPIKey: []config.VertexCompatKey{{
				APIKey:   "vertex-key",
				BaseURL:  "https://vertex.example",
				ProxyURL: "http://vertex-proxy.local:8080",
			}}}},
			auth: &coreauth.Auth{Attributes: map[string]string{
				"api_key":  "vertex-key",
				"base_url": "https://vertex.example",
			}},
			resolve: func(service *Service, auth *coreauth.Auth) bool {
				return service.resolveConfigVertexCompatKey(auth) != nil
			},
			want: "http://vertex-proxy.local:8080",
		},
		{
			name: "codex",
			service: &Service{cfg: &config.Config{CodexKey: []config.CodexKey{{
				APIKey:   "codex-key",
				BaseURL:  "https://openai.example",
				ProxyURL: "http://codex-proxy.local:8080",
			}}}},
			auth: &coreauth.Auth{Attributes: map[string]string{
				"api_key":  "codex-key",
				"base_url": "https://openai.example",
			}},
			resolve: func(service *Service, auth *coreauth.Auth) bool {
				return service.resolveConfigCodexKey(auth) != nil
			},
			want: "http://codex-proxy.local:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.resolve(tt.service, tt.auth) {
				t.Fatal("expected config entry to resolve")
			}
			if tt.auth.ProxyURL != tt.want {
				t.Fatalf("auth.ProxyURL = %q, want %q", tt.auth.ProxyURL, tt.want)
			}
		})
	}
}

func TestResolveConfigAPIKeyDoesNotOverwriteProxyURLWithEmptyConfigValue(t *testing.T) {
	service := &Service{cfg: &config.Config{ClaudeKey: []config.ClaudeKey{{
		APIKey: "claude-key",
	}}}}
	auth := &coreauth.Auth{
		ProxyURL: "http://existing-proxy.local:8080",
		Attributes: map[string]string{
			"api_key": "claude-key",
		},
	}

	if service.resolveConfigClaudeKey(auth) == nil {
		t.Fatal("expected config entry to resolve")
	}
	if auth.ProxyURL != "http://existing-proxy.local:8080" {
		t.Fatalf("auth.ProxyURL = %q, want existing proxy to remain", auth.ProxyURL)
	}
}
