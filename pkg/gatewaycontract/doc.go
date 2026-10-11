// Package gatewaycontract is the Redis runtime contract Admin publishes and
// Gateway reads: key shapes plus the documents stored at those keys.
//
// It does not talk to Redis. Callers keep their own client and decide which
// command to run. HTTP pull documents (GatewayConfig, EndpointConfig) are a
// separate contract and do not live here.
package gatewaycontract
