package biz

import (
	"reflect"
	"testing"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaykeys"
)

func TestNormalizeRequestTypesForProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		in       []string
		want     []string
	}{
		{
			name:     "anthropic chat model is exported as messages capability",
			protocol: "anthropic",
			in:       []string{"chat_completion"},
			want:     []string{"messages"},
		},
		{
			name:     "anthropic keeps explicit messages and drops unsupported embedding",
			protocol: "anthropic",
			in:       []string{"messages", "embedding"},
			want:     []string{"messages"},
		},
		{
			name:     "openai preserves declared supported request types",
			protocol: "openai",
			in:       []string{"chat_completion", "messages", "responses", "embedding"},
			want:     []string{"chat_completion", "messages", "responses", "embedding"},
		},
		{
			name:     "unknown protocol preserves unique declared request types",
			protocol: "custom",
			in:       []string{"chat_completion", "chat_completion"},
			want:     []string{"chat_completion"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeRequestTypesForProtocol(tt.protocol, tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeRequestTypesForProtocol(%q, %v) = %v, want %v", tt.protocol, tt.in, got, tt.want)
			}
		})
	}
}

func TestRuntimeAPIKeyRedisKeyUsesHashWhenPepperConfigured(t *testing.T) {
	oldPepper := config.C.Gateway.APIKeyPepper
	t.Cleanup(func() { config.C.Gateway.APIKeyPepper = oldPepper })
	config.C.Gateway.APIKeyPepper = "pepper"

	apiKey := "tl_live_example"
	keyHash := gatewaykeys.HashAPIKey(apiKey, "pepper")
	got := runtimeAPIKeyRedisKey(apiKey)
	want := gatewaykeys.RedisKeyAPIKeyHash(keyHash)
	if got != want {
		t.Fatalf("runtimeAPIKeyRedisKey() = %q, want %q", got, want)
	}
}

func TestRuntimeAPIKeyRedisKeyUsesHashWithoutPepper(t *testing.T) {
	oldPepper := config.C.Gateway.APIKeyPepper
	t.Cleanup(func() { config.C.Gateway.APIKeyPepper = oldPepper })
	config.C.Gateway.APIKeyPepper = ""

	apiKey := "tl_live_example"
	got := runtimeAPIKeyRedisKey(apiKey)
	want := gatewaykeys.RedisKeyAPIKeyHash(gatewaykeys.HashAPIKey(apiKey, ""))
	if got != want {
		t.Fatalf("runtimeAPIKeyRedisKey() = %q, want %q", got, want)
	}
}
