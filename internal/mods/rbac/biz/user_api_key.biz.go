package biz

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ProvideRedisClient 提供 Redis 客户端单例以供同步使用
func ProvideRedisClient() *redis.Client {
	cfg := config.C.Storage.Cache.Redis
	if cfg.Addr == "" {
		return nil
	}
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}

// UserAPIKey 业务逻辑处理结构体
type UserAPIKey struct {
	Trans         *util.Trans
	UserAPIKeyDAL *dal.UserAPIKey
	UserDAL       *dal.User
	RedisClient   *redis.Client
	AuditLogBIZ   *opsBiz.AuditLog
}

// checkOwnership 校验当前用户是否有权操作该 API Key。
// root 用户（超级管理员）跳过校验，普通用户只能操作自己名下的 Key。
func (a *UserAPIKey) checkOwnership(ctx context.Context, item *schema.UserAPIKey) error {
	if util.FromIsRootUser(ctx) {
		return nil
	}
	if item.UserID != util.FromUserID(ctx) {
		return errors.NotFound("", "API key not found")
	}
	return nil
}

// Query 分页查询用户 API Key 列表（已掩码脱敏）
func (a *UserAPIKey) Query(ctx context.Context, params schema.UserAPIKeyQueryParam) (*schema.UserAPIKeyQueryResult, error) {
	params.Pagination = true

	// 非 root 用户只能查看自己的 API Key
	if !util.FromIsRootUser(ctx) {
		params.UserID = util.FromUserID(ctx)
	}

	result, err := a.UserAPIKeyDAL.Query(ctx, params, schema.UserAPIKeyQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	// 安全脱敏，列表不显示完整明文 API Key
	result.Data.Mask()
	return result, nil
}

// GetPlaintext 获取单条 API Key 的明文（用于复制操作）
// 安全考虑：仅允许管理员或密钥所属用户获取明文
func (a *UserAPIKey) GetPlaintext(ctx context.Context, id string) (string, error) {
	item, err := a.UserAPIKeyDAL.Get(ctx, id)
	if err != nil {
		return "", err
	} else if item == nil {
		return "", errors.NotFound("", "API key not found")
	}

	if err := a.checkOwnership(ctx, item); err != nil {
		return "", err
	}

	// 返回明文 API Key
	return item.APIKey, nil
}

// Get 获取单条 API Key 记录（已掩码脱敏）
func (a *UserAPIKey) Get(ctx context.Context, id string) (*schema.UserAPIKey, error) {
	item, err := a.UserAPIKeyDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if item == nil {
		return nil, errors.NotFound("", "API key not found")
	}

	if err := a.checkOwnership(ctx, item); err != nil {
		return nil, err
	}

	// 详情查询也进行掩码处理，仅在 Create 接口回显明文
	item.Mask()
	return item, nil
}

// Create 创建用户 API Key（仅在此接口返回新生成的明文 API Key 供前台复制）
func (a *UserAPIKey) Create(ctx context.Context, formItem *schema.UserAPIKeyForm) (*schema.UserAPIKey, error) {
	// 非 root 用户只能给自己创建 API Key
	if !util.FromIsRootUser(ctx) {
		formItem.UserID = util.FromUserID(ctx)
	}

	// 1. 自动生成密码学安全的高熵 API Key
	rawKey, err := a.generateRandomAPIKey(ctx)
	if err != nil {
		return nil, err
	}

	apiKey := &schema.UserAPIKey{
		ID:        util.NewXID(),
		APIKey:    rawKey,
		CreatedAt: time.Now(),
	}
	formItem.FillTo(apiKey)

	username := util.FromUsername(ctx)
	if username != "" {
		apiKey.Creator = username
	}

	// 2. 数据库事务落库
	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.UserAPIKeyDAL.Create(ctx, apiKey)
	})
	if err != nil {
		return nil, err
	}

	// 3. 事务提交成功后同步更新 Redis 缓存（异步防挂，或同步确保一致性。这里使用同步，并容忍 Redis 的网络错误打印日志）
	if err := a.syncToRedis(ctx, apiKey); err != nil {
		// Redis 同步失败不回滚 DB，但打印 Warn 日志
		// 避免影响核心业务
	}

	// 注意：此处返回的 apiKey 包含 rawKey (明文)，供前端一次性展示
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypeAPIKey, apiKey.ID, apiKey.Name, nil, apiKey)
	return apiKey, nil
}

// Update 更新用户 API Key（支持更新 Name, Status, Quota, ExpiresAt, Description）
func (a *UserAPIKey) Update(ctx context.Context, id string, formItem *schema.UserAPIKeyForm) error {
	apiKey, err := a.UserAPIKeyDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if apiKey == nil {
		return errors.NotFound("", "API key not found")
	}

	if err := a.checkOwnership(ctx, apiKey); err != nil {
		return err
	}

	beforeAPIKey := *apiKey

	formItem.FillTo(apiKey)
	apiKey.UpdatedAt = time.Now()

	username := util.FromUsername(ctx)
	if username != "" {
		apiKey.Modifier = username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.UserAPIKeyDAL.Update(ctx, apiKey)
	})
	if err != nil {
		return err
	}

	// 同步 Redis
	_ = a.syncToRedis(ctx, apiKey)
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypeAPIKey, apiKey.ID, apiKey.Name, beforeAPIKey, apiKey)
	return nil
}

// Delete 逻辑删除用户 API Key
func (a *UserAPIKey) Delete(ctx context.Context, id string) error {
	apiKey, err := a.UserAPIKeyDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if apiKey == nil {
		return errors.NotFound("", "API key not found")
	}

	if err := a.checkOwnership(ctx, apiKey); err != nil {
		return err
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.UserAPIKeyDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 逻辑删除时直接从 Redis 清除 Key
	if a.RedisClient != nil {
		redisKey := "aigw:apikey:" + apiKey.APIKey
		_ = a.RedisClient.Del(ctx, redisKey).Err()
	}
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypeAPIKey, apiKey.ID, apiKey.Name, apiKey, nil)
	return nil
}

// generateRandomAPIKey 生成高熵值随机 API Key
func (a *UserAPIKey) generateRandomAPIKey(ctx context.Context) (string, error) {
	for i := 0; i < 3; i++ { // 重试 3 次防碰撞
		bytes := make([]byte, 16) // 16 字节 = 32位 16进制字符
		if _, err := rand.Read(bytes); err != nil {
			return "", errors.WithStack(err)
		}
		key := "sk-" + hex.EncodeToString(bytes)

		exists, err := a.UserAPIKeyDAL.ExistsAPIKey(ctx, key)
		if err != nil {
			return "", err
		}
		if !exists {
			return key, nil
		}
	}
	return "", errors.InternalServerError("", "Failed to generate unique API key")
}

// syncToRedis 同步 API Key 数据到 Redis
func (a *UserAPIKey) syncToRedis(ctx context.Context, apiKey *schema.UserAPIKey) error {
	if a.RedisClient == nil {
		return nil
	}

	redisKey := "aigw:apikey:" + apiKey.APIKey

	// 逻辑删除或者已删除，直接清除 Redis 缓存
	if apiKey.Deleted != "0" {
		return a.RedisClient.Del(ctx, redisKey).Err()
	}

	// 转换 ExpiresAt 转换为 Unix 时间戳 (秒)
	var expiresAtVal int64 = 0
	if apiKey.ExpiresAt != nil {
		expiresAtVal = apiKey.ExpiresAt.Unix()
	}

	// 查询用户所属租户
	userTenant := ""
	if user, err := a.UserDAL.Get(ctx, apiKey.UserID); err == nil && user != nil {
		userTenant = user.Tenant
	}

	// 准备 Hash 字段
	fields := map[string]interface{}{
		"user_id":     apiKey.UserID,
		"user_tenant": userTenant,
		"status":      apiKey.Status,
		"quota":       apiKey.Quota,
		"expires_at":  expiresAtVal,
	}

	// 执行 HSET 写入 Redis
	return a.RedisClient.HSet(ctx, redisKey, fields).Err()
}
