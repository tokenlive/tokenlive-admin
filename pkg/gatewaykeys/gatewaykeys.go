// Package gatewaykeys is the previous home of the API-key half of the Gateway
// runtime contract. New code should use pkg/gatewaycontract.
package gatewaykeys

import "github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"

// HashAPIKey hashes a runtime API key the way Gateway looks it up.
func HashAPIKey(apiKey string, pepper string) string {
	return gatewaycontract.HashAPIKey(apiKey, pepper)
}

// RedisKeyAPIKeyHash is the Redis HASH key for a hashed runtime API key.
func RedisKeyAPIKeyHash(keyHash string) string {
	return gatewaycontract.Keys.APIKey.Hash(keyHash)
}
