package biz

import (
	"testing"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaykeys"
)

func TestAPIKeyRuntimeRedisKeyUsesHashWhenPepperConfigured(t *testing.T) {
	oldPepper := config.C.Gateway.APIKeyPepper
	t.Cleanup(func() { config.C.Gateway.APIKeyPepper = oldPepper })
	config.C.Gateway.APIKeyPepper = "pepper"

	apiKey := "tl_live_example"
	keyHash := gatewaykeys.HashAPIKey(apiKey, "pepper")
	got := apiKeyRuntimeRedisKey(apiKey)
	want := gatewaykeys.RedisKeyAPIKeyHash(keyHash)
	if got != want {
		t.Fatalf("apiKeyRuntimeRedisKey() = %q, want %q", got, want)
	}
}

func TestAPIKeyRuntimeRedisKeyUsesHashWithoutPepper(t *testing.T) {
	oldPepper := config.C.Gateway.APIKeyPepper
	t.Cleanup(func() { config.C.Gateway.APIKeyPepper = oldPepper })
	config.C.Gateway.APIKeyPepper = ""

	apiKey := "tl_live_example"
	got := apiKeyRuntimeRedisKey(apiKey)
	want := gatewaykeys.RedisKeyAPIKeyHash(gatewaykeys.HashAPIKey(apiKey, ""))
	if got != want {
		t.Fatalf("apiKeyRuntimeRedisKey() = %q, want %q", got, want)
	}
}
