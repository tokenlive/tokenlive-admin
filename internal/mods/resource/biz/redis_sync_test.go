package biz

import (
	"context"
	"reflect"
	"testing"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaykeys"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestNormalizeRequestTypesForProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		in       []string
		want     []string
	}{
		{
			name:     "anthropic preserves declared request types",
			protocol: "anthropic",
			in:       []string{"chat_completion", "messages"},
			want:     []string{"chat_completion", "messages"},
		},
		{
			name:     "deduplicates unique declared request types",
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

func TestConfigRedisSync_NilRedis_NotifiesConfigChanged(t *testing.T) {
	util.ResetConfigChangeListeners()
	t.Cleanup(util.ResetConfigChangeListeners)

	var notifiedKind string
	var notifiedKey string
	util.OnConfigChanged(func(ctx context.Context, kind util.ConfigChangeKind, keys ...string) {
		notifiedKind = kind
		if len(keys) > 0 {
			notifiedKey = keys[0]
		}
	})

	sync := &ConfigRedisSync{RedisClient: nil}
	ctx := context.Background()

	// 1. SyncModelByCode
	notifiedKind = ""
	notifiedKey = ""
	if err := sync.SyncModelByCode(ctx, "test-model"); err != nil {
		t.Fatalf("SyncModelByCode error: %v", err)
	}
	if notifiedKind != util.ConfigChangeAll || notifiedKey != "test-model" {
		t.Errorf("expected ConfigChangeAll with test-model, got %s, %s", notifiedKind, notifiedKey)
	}

	// 2. SyncProviderID
	notifiedKind = ""
	if err := sync.SyncProviderID(ctx, "p1"); err != nil {
		t.Fatalf("SyncProviderID error: %v", err)
	}
	if notifiedKind != util.ConfigChangeAll {
		t.Errorf("expected ConfigChangeAll on SyncProviderID, got %s", notifiedKind)
	}

	// 3. SyncModelDisable
	notifiedKind = ""
	notifiedKey = ""
	if err := sync.SyncModelDisable(ctx, "m1", "test-model"); err != nil {
		t.Fatalf("SyncModelDisable error: %v", err)
	}
	if notifiedKind != util.ConfigChangeAll || notifiedKey != "test-model" {
		t.Errorf("expected ConfigChangeAll on SyncModelDisable, got %s, %s", notifiedKind, notifiedKey)
	}

	// 4. SyncModelEnable
	notifiedKind = ""
	notifiedKey = ""
	if err := sync.SyncModelEnable(ctx, "m1", "test-model"); err != nil {
		t.Fatalf("SyncModelEnable error: %v", err)
	}
	if notifiedKind != util.ConfigChangeAll || notifiedKey != "test-model" {
		t.Errorf("expected ConfigChangeAll on SyncModelEnable, got %s, %s", notifiedKind, notifiedKey)
	}
}
