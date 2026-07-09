package util

var ClearGatewayConfigCacheFunc func()

// ClearGatewayConfigCache clears the gateway config cache via registered func
func ClearGatewayConfigCache() {
	if ClearGatewayConfigCacheFunc != nil {
		ClearGatewayConfigCacheFunc()
	}
}
