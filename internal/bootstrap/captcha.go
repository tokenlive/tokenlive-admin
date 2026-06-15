package bootstrap

import (
	"context"
	"time"

	"github.com/LyricTian/captcha"
	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
)

// CaptchaRedisStore implements captcha.Store interface using redis v9 client
type CaptchaRedisStore struct {
	cli        *redis.Client
	expiration time.Duration
	prefix     string
}

// NewCaptchaRedisStore creates a new CaptchaRedisStore instance
func NewCaptchaRedisStore(cli *redis.Client, expiration time.Duration, prefix string) *CaptchaRedisStore {
	return &CaptchaRedisStore{
		cli:        cli,
		expiration: expiration,
		prefix:     prefix,
	}
}

// Set stores the captcha id and digits in redis
func (s *CaptchaRedisStore) Set(id string, digits []byte) {
	ctx := context.Background()
	_ = s.cli.Set(ctx, s.prefix+id, digits, s.expiration).Err()
}

// Get retrieves the captcha digits from redis and optionally deletes it
func (s *CaptchaRedisStore) Get(id string, clear bool) []byte {
	ctx := context.Background()
	key := s.prefix + id
	val, err := s.cli.Get(ctx, key).Bytes()
	if err != nil {
		return nil
	}
	if clear {
		_ = s.cli.Del(ctx, key).Err()
	}
	return val
}

// initCaptcha initializes captcha custom store based on configuration
func initCaptcha() {
	cfg := config.C.Util.Captcha
	if cfg.CacheType == "redis" {
		redisCfg := cfg.Redis
		if redisCfg.Addr != "" {
			cli := redis.NewClient(&redis.Options{
				Addr:     redisCfg.Addr,
				Username: redisCfg.Username,
				Password: redisCfg.Password,
				DB:       redisCfg.DB,
			})
			store := NewCaptchaRedisStore(cli, captcha.Expiration, redisCfg.KeyPrefix)
			captcha.SetCustomStore(store)
		}
	}
}
