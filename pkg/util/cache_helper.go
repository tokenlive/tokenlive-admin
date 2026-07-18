package util

import (
	"context"
	"sync"
)

// ClearGatewayConfigCacheFunc clears in-process gateway config cache (set by resource/biz).
var ClearGatewayConfigCacheFunc func()

// ClearGatewayConfigCache clears the gateway config cache via registered func.
func ClearGatewayConfigCache() {
	if ClearGatewayConfigCacheFunc != nil {
		ClearGatewayConfigCacheFunc()
	}
}

// ConfigChangeKind identifies which gateway runtime data changed.
// Values: "endpoints" | "policies" | "apikeys" | "all"
type ConfigChangeKind = string

const (
	ConfigChangeEndpoints ConfigChangeKind = "endpoints"
	ConfigChangePolicies  ConfigChangeKind = "policies"
	ConfigChangeAPIKeys   ConfigChangeKind = "apikeys"
	ConfigChangeAll       ConfigChangeKind = "all"
)

// ConfigChangeListener is invoked after admin mutations that affect gateway runtime.
// Admin never imports gateway; hosts (e.g. tokenlive-standalone) register listeners.
type ConfigChangeListener func(ctx context.Context, kind ConfigChangeKind, keys ...string)

var (
	configChangeMu        sync.RWMutex
	configChangeListeners []ConfigChangeListener
)

// OnConfigChanged registers a listener. Safe for concurrent use.
func OnConfigChanged(fn ConfigChangeListener) {
	if fn == nil {
		return
	}
	configChangeMu.Lock()
	configChangeListeners = append(configChangeListeners, fn)
	configChangeMu.Unlock()
}

// ResetConfigChangeListeners clears all listeners (tests / shutdown).
func ResetConfigChangeListeners() {
	configChangeMu.Lock()
	configChangeListeners = nil
	configChangeMu.Unlock()
}

// NotifyConfigChanged invokes all listeners. Never panics on listener failure.
func NotifyConfigChanged(ctx context.Context, kind ConfigChangeKind, keys ...string) {
	configChangeMu.RLock()
	listeners := append([]ConfigChangeListener(nil), configChangeListeners...)
	configChangeMu.RUnlock()
	for _, fn := range listeners {
		func() {
			defer func() { _ = recover() }()
			fn(ctx, kind, keys...)
		}()
	}
}
