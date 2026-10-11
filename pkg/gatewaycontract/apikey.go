package gatewaycontract

import "github.com/tokenlive/tokenlive-admin/pkg/crypto/hash"

// HashAPIKey is the HMAC Gateway uses to look up a runtime API key.
// An empty pepper is a valid HMAC key here; Gateway itself rejects an empty
// pepper when it reads, and that mismatch is not resolved in this package.
func HashAPIKey(apiKey string, pepper string) string {
	return hash.HMACSHA256String(apiKey, pepper)
}
