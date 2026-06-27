package ops

import (
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/dal"
)

var Set = wire.NewSet(
	wire.Struct(new(Ops), "*"),
	wire.Struct(new(dal.EventLog), "*"),
	wire.Struct(new(biz.EventBiz), "*"),
	wire.Struct(new(biz.Consumer), "Subscriber", "EventDAL"),
	wire.Struct(new(biz.CleanupTask), "*"),
	wire.Struct(new(api.EventAPI), "EventBIZ", "Hub"),
	api.NewWSHub,
	ProvideEventSubscriber,
	// Audit Log
	wire.Struct(new(dal.AuditLog), "*"),
	wire.Struct(new(biz.AuditLog), "*"),
	wire.Struct(new(api.AuditLog), "*"),
	// Portal User API Proxy
	wire.Struct(new(biz.PortalUser), "*"),
	wire.Struct(new(api.PortalUserAPI), "*"),
)

// ProvideEventSubscriber creates the appropriate EventSubscriber based on config.
func ProvideEventSubscriber(redisClient *redis.Client) biz.EventSubscriber {
	queueCfg := config.C.Storage.EventQueue
	switch queueCfg.Type {
	case "kafka":
		return biz.NewKafkaSubscriber(biz.KafkaConfig{
			Brokers:       queueCfg.Kafka.Brokers,
			Topic:         queueCfg.Topic,
			ConsumerGroup: queueCfg.ConsumerGroup,
		})
	default: // "redis"
		return biz.NewRedisStreamSubscriber(redisClient, biz.RedisStreamConfig{
			StreamKey:     queueCfg.Topic,
			ConsumerGroup: queueCfg.ConsumerGroup,
		})
	}
}
