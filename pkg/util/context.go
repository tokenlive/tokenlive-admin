package util

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/pkg/encoding/json"
	"gorm.io/gorm"
)

type (
	traceIDCtx    struct{}
	transCtx      struct{}
	rowLockCtx    struct{}
	userIDCtx     struct{}
	usernameCtx   struct{}
	tenantCtx     struct{}
	userTokenCtx  struct{}
	isRootUserCtx struct{}
	userCacheCtx  struct{}
	clientIPCtx   struct{}
	userAgentCtx  struct{}
)

func NewTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDCtx{}, traceID)
}

func FromTraceID(ctx context.Context) string {
	v := ctx.Value(traceIDCtx{})
	if v != nil {
		return v.(string)
	}
	return ""
}

func NewTrans(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, transCtx{}, db)
}

func FromTrans(ctx context.Context) (*gorm.DB, bool) {
	v := ctx.Value(transCtx{})
	if v != nil {
		return v.(*gorm.DB), true
	}
	return nil, false
}

func NewRowLock(ctx context.Context) context.Context {
	return context.WithValue(ctx, rowLockCtx{}, true)
}

func FromRowLock(ctx context.Context) bool {
	v := ctx.Value(rowLockCtx{})
	return v != nil && v.(bool)
}

func NewUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDCtx{}, userID)
}

func FromUserID(ctx context.Context) string {
	v := ctx.Value(userIDCtx{})
	if v != nil {
		return v.(string)
	}
	return ""
}

func NewUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameCtx{}, username)
}

func FromUsername(ctx context.Context) string {
	v := ctx.Value(usernameCtx{})
	if v != nil {
		return v.(string)
	}
	return ""
}

func NewTenant(ctx context.Context, tenant string) context.Context {
	return context.WithValue(ctx, tenantCtx{}, tenant)
}

func FromTenant(ctx context.Context) string {
	v := ctx.Value(tenantCtx{})
	if v != nil {
		return v.(string)
	}
	return ""
}

func NewUserToken(ctx context.Context, userToken string) context.Context {
	return context.WithValue(ctx, userTokenCtx{}, userToken)
}

func FromUserToken(ctx context.Context) string {
	v := ctx.Value(userTokenCtx{})
	if v != nil {
		return v.(string)
	}
	return ""
}

func NewIsRootUser(ctx context.Context) context.Context {
	return context.WithValue(ctx, isRootUserCtx{}, true)
}

func FromIsRootUser(ctx context.Context) bool {
	v := ctx.Value(isRootUserCtx{})
	return v != nil && v.(bool)
}

// Set user cache object
type UserCache struct {
	RoleIDs  []string `json:"rids"`
	Username string   `json:"username"`
	Tenant   string   `json:"tenant"`
}

func ParseUserCache(s string) UserCache {
	var a UserCache
	if s == "" {
		return a
	}

	_ = json.Unmarshal([]byte(s), &a)
	return a
}

func (a UserCache) String() string {
	s := json.MarshalToString(a)
	if s == nil {
		return ""
	}
	return *s
}

func NewUserCache(ctx context.Context, userCache UserCache) context.Context {
	return context.WithValue(ctx, userCacheCtx{}, userCache)
}

func FromUserCache(ctx context.Context) UserCache {
	v := ctx.Value(userCacheCtx{})
	if v != nil {
		return v.(UserCache)
	}
	return UserCache{}
}

func NewClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPCtx{}, ip)
}

func FromClientIP(ctx context.Context) string {
	v := ctx.Value(clientIPCtx{})
	if v != nil {
		return v.(string)
	}
	return ""
}

func NewUserAgent(ctx context.Context, userAgent string) context.Context {
	return context.WithValue(ctx, userAgentCtx{}, userAgent)
}

func FromUserAgent(ctx context.Context) string {
	v := ctx.Value(userAgentCtx{})
	if v != nil {
		return v.(string)
	}
	return ""
}
