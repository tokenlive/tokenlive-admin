package biz

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"go.uber.org/zap"
)

const (
	defaultBatchSize      = 100
	defaultBlockTimeout   = 5 * time.Second
	defaultMaxBackoff     = 30 * time.Second
	defaultInitialBackoff = 1 * time.Second
)

// RedisStreamConfig holds Redis Stream consumer configuration.
type RedisStreamConfig struct {
	StreamKey     string // e.g. "aigw:events:policy"
	ConsumerGroup string // e.g. "admin-consumer"
	BatchSize     int64
	BlockTimeout  time.Duration
}

// RedisStreamSubscriber implements EventSubscriber using Redis Streams (XREADGROUP).
type RedisStreamSubscriber struct {
	client       *redis.Client
	config       RedisStreamConfig
	consumerName string
}

// NewRedisStreamSubscriber creates a new Redis Stream subscriber.
func NewRedisStreamSubscriber(client *redis.Client, cfg RedisStreamConfig) *RedisStreamSubscriber {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultBatchSize
	}
	if cfg.BlockTimeout <= 0 {
		cfg.BlockTimeout = defaultBlockTimeout
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	consumerName := fmt.Sprintf("admin-%s-%d", hostname, os.Getpid())

	return &RedisStreamSubscriber{
		client:       client,
		config:       cfg,
		consumerName: consumerName,
	}
}

// Subscribe starts consuming messages from the Redis Stream.
func (s *RedisStreamSubscriber) Subscribe(ctx context.Context, handler EventHandler) error {
	logging.Context(ctx).Info("redis stream subscriber starting",
		zap.String("stream", s.config.StreamKey),
		zap.String("group", s.config.ConsumerGroup),
		zap.String("consumer", s.consumerName),
	)

	// Create consumer group idempotently
	if err := s.ensureConsumerGroup(ctx); err != nil {
		return err
	}

	backoff := defaultInitialBackoff
	for {
		select {
		case <-ctx.Done():
			logging.Context(ctx).Info("redis stream subscriber stopped")
			return nil
		default:
		}

		streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    s.config.ConsumerGroup,
			Consumer: s.consumerName,
			Streams:  []string{s.config.StreamKey, ">"},
			Count:    s.config.BatchSize,
			Block:    s.config.BlockTimeout,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			logging.Context(ctx).Warn("redis stream subscriber read error, retrying...",
				zap.Error(err),
				zap.Duration("backoff", backoff),
			)
			time.Sleep(backoff)
			backoff = minDuration(backoff*2, defaultMaxBackoff)
			continue
		}

		backoff = defaultInitialBackoff

		for _, stream := range streams {
			if len(stream.Messages) == 0 {
				continue
			}

			for _, msg := range stream.Messages {
				normalized := &Message{
					ID:     msg.ID,
					Fields: normalizeFields(msg.Values),
				}

				if err := handler(ctx, normalized); err != nil {
					logging.Context(ctx).Warn("event handler error, message will be redelivered",
						zap.String("id", msg.ID),
						zap.Error(err),
					)
					// Do NOT ACK — message will be redelivered
					continue
				}

				// ACK on success
				if err := s.client.XAck(ctx, s.config.StreamKey, s.config.ConsumerGroup, msg.ID).Err(); err != nil {
					logging.Context(ctx).Warn("redis stream ack failed",
						zap.String("id", msg.ID),
						zap.Error(err),
					)
				}
			}
		}
	}
}

// Close releases the Redis client resources.
func (s *RedisStreamSubscriber) Close() error {
	// The Redis client is shared and managed externally, so we don't close it here.
	return nil
}

func (s *RedisStreamSubscriber) ensureConsumerGroup(ctx context.Context) error {
	err := s.client.XGroupCreateMkStream(ctx, s.config.StreamKey, s.config.ConsumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		logging.Context(ctx).Error("failed to create consumer group", zap.Error(err))
		return err
	}
	return nil
}

func normalizeFields(fields map[string]interface{}) map[string]interface{} {
	if fields == nil {
		return make(map[string]interface{})
	}
	return fields
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// NoOpSubscriber implements EventSubscriber and does nothing.
type NoOpSubscriber struct{}

func NewNoOpSubscriber() *NoOpSubscriber {
	return &NoOpSubscriber{}
}

func (s *NoOpSubscriber) Subscribe(ctx context.Context, handler EventHandler) error {
	zap.L().Warn("No-Op event queue subscriber is active. Event processing is disabled because Redis is not configured.")
	<-ctx.Done()
	return nil
}

func (s *NoOpSubscriber) Close() error {
	return nil
}
