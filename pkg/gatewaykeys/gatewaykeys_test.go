package gatewaykeys

import "testing"

func TestHashAPIKeyMatchesGatewayHMACSHA256(t *testing.T) {
	got := HashAPIKey("tl_live_example", "pepper")
	want := "06bfbed9282f1dcb96bd25c7bef96d9b49de0be5f3777b44f4f71cfcca8821b1"
	if got != want {
		t.Fatalf("HashAPIKey() = %q, want %q", got, want)
	}
}

func TestRedisKeyAPIKeyHash(t *testing.T) {
	got := RedisKeyAPIKeyHash("hash123")
	want := "aigw:apikey_hash:hash123"
	if got != want {
		t.Fatalf("RedisKeyAPIKeyHash() = %q, want %q", got, want)
	}
}
