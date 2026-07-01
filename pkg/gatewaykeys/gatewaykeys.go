package gatewaykeys

import "github.com/tokenlive/tokenlive-admin/pkg/crypto/hash"

func HashAPIKey(apiKey string, pepper string) string {
	return hash.HMACSHA256String(apiKey, pepper)
}

func RedisKeyAPIKeyHash(keyHash string) string {
	return "aigw:apikey_hash:" + keyHash
}
