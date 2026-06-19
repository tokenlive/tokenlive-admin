package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"go.uber.org/zap"
)

// KafkaConfig holds Kafka consumer configuration.
type KafkaConfig struct {
	Brokers       []string // e.g. ["localhost:9092"]
	Topic         string   // e.g. "aigw-events-policy"
	ConsumerGroup string   // e.g. "admin-consumer"
	MinBytes      int      // minimum bytes to fetch (default 1)
	MaxBytes      int      // maximum bytes to fetch (default 10MB)
	MaxWait       time.Duration // max wait time for new data (default 1s)
}

// KafkaSubscriber implements EventSubscriber using Kafka consumer groups.
type KafkaSubscriber struct {
	reader *kafka.Reader
	config KafkaConfig
}

// NewKafkaSubscriber creates a new Kafka subscriber.
func NewKafkaSubscriber(cfg KafkaConfig) *KafkaSubscriber {
	if cfg.MinBytes <= 0 {
		cfg.MinBytes = 1
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 10e6 // 10MB
	}
	if cfg.MaxWait <= 0 {
		cfg.MaxWait = 1 * time.Second
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	groupID := cfg.ConsumerGroup
	if groupID == "" {
		groupID = fmt.Sprintf("admin-%s-%d", hostname, os.Getpid())
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  groupID,
		MinBytes: cfg.MinBytes,
		MaxBytes: cfg.MaxBytes,
		MaxWait:  cfg.MaxWait,
		StartOffset: kafka.LastOffset,
	})

	return &KafkaSubscriber{
		reader: reader,
		config: cfg,
	}
}

// Subscribe starts consuming messages from Kafka.
func (s *KafkaSubscriber) Subscribe(ctx context.Context, handler EventHandler) error {
	logging.Context(ctx).Info("kafka subscriber starting",
		zap.String("topic", s.config.Topic),
		zap.String("group", s.config.ConsumerGroup),
		zap.Strings("brokers", s.config.Brokers),
	)

	for {
		select {
		case <-ctx.Done():
			logging.Context(ctx).Info("kafka subscriber stopped")
			return nil
		default:
		}

		msg, err := s.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				// Context cancelled, exit gracefully
				logging.Context(ctx).Info("kafka subscriber stopped")
				return nil
			}
			logging.Context(ctx).Warn("kafka fetch error", zap.Error(err))
			time.Sleep(defaultInitialBackoff)
			continue
		}

		// Normalize Kafka message to our Message format
		normalized := kafkaMessageToMessage(msg)

		if err := handler(ctx, normalized); err != nil {
			logging.Context(ctx).Warn("event handler error, message will be redelivered",
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.Error(err),
			)
			// Do NOT commit — message will be redelivered
			continue
		}

		// Commit on success
		if err := s.reader.CommitMessages(ctx, msg); err != nil {
			logging.Context(ctx).Warn("kafka commit failed",
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.Error(err),
			)
		}
	}
}

// Close closes the Kafka reader.
func (s *KafkaSubscriber) Close() error {
	return s.reader.Close()
}

// kafkaMessageToMessage converts a Kafka message to our normalized Message format.
// The Kafka message value is expected to be a JSON object with string key-value pairs,
// matching the same field structure as Redis Stream XADD.
func kafkaMessageToMessage(msg kafka.Message) *Message {
	m := &Message{
		ID:     fmt.Sprintf("%d-%d", msg.Partition, msg.Offset),
		Fields: make(map[string]interface{}),
	}

	// Try to parse the message value as a JSON object (key-value pairs)
	var fields map[string]interface{}
	if err := json.Unmarshal(msg.Value, &fields); err == nil {
		m.Fields = fields
	} else {
		// If not JSON, treat the entire value as a single "message" field
		m.Fields["message"] = string(msg.Value)
	}

	// Also extract fields from Kafka message headers if present
	for _, h := range msg.Headers {
		if _, exists := m.Fields[h.Key]; !exists {
			m.Fields[h.Key] = string(h.Value)
		}
	}

	return m
}
