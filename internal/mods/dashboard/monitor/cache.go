package monitor

import (
	"context"
	"time"
)

type cacheVal struct {
	data      interface{}
	expiredAt time.Time
	source    string
	failed    map[string]bool
	seen      map[int64]bool
}

func (a *Reader) getCache(ctx context.Context, key string) (interface{}, bool) {
	if s := scope(ctx); s != nil {
		key += ":" + s.plan.identity()
	}
	a.cacheMu.RLock()
	defer a.cacheMu.RUnlock()
	if a.cacheMap == nil {
		return nil, false
	}
	val, ok := a.cacheMap[key]
	if !ok || time.Now().After(val.expiredAt) {
		return nil, false
	}
	if s := scope(ctx); s != nil {
		s.audit.source = val.source
		for k, v := range val.failed {
			s.audit.failed[k] = v
		}
		for k, v := range val.seen {
			s.audit.seen[k] = v
		}
	}
	return val.data, true
}

func (a *Reader) setCache(ctx context.Context, key string, data interface{}, ttl time.Duration) {
	var audit *readAudit
	if s := scope(ctx); s != nil {
		audit = s.audit
		if len(audit.failed) > 0 || (audit.source == "prometheus" && len(audit.seen) == 0) {
			return
		}
		key += ":" + s.plan.identity()
	}
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()
	if a.cacheMap == nil {
		a.cacheMap = make(map[string]cacheVal)
	}
	entry := cacheVal{data: data, expiredAt: time.Now().Add(ttl)}
	if audit != nil {
		entry.source = audit.source
		entry.failed = map[string]bool{}
		entry.seen = map[int64]bool{}
		for k, v := range audit.failed {
			entry.failed[k] = v
		}
		for k, v := range audit.seen {
			entry.seen[k] = v
		}
	}
	// Expired window identities must not accumulate in a long-lived Reader.
	for k, v := range a.cacheMap {
		if time.Now().After(v.expiredAt) {
			delete(a.cacheMap, k)
		}
	}
	a.cacheMap[key] = entry
}
