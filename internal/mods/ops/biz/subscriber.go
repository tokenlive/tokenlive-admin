package biz

import "context"

// Message is a normalized message from any transport layer.
type Message struct {
	ID     string                 // Message ID (Redis Stream ID or Kafka offset+partition)
	Fields map[string]interface{} // Message fields as key-value pairs
}

// EventHandler is the callback invoked for each consumed message.
// Return a non-nil error to signal processing failure — the transport should redeliver.
type EventHandler func(ctx context.Context, msg *Message) error

// EventSubscriber abstracts the message consumption transport.
// Implementations: RedisStreamSubscriber, KafkaSubscriber (future).
type EventSubscriber interface {
	// Subscribe starts consuming messages and calls handler for each one.
	// Blocks until ctx is cancelled or a fatal error occurs.
	Subscribe(ctx context.Context, handler EventHandler) error

	// Close releases transport resources (connections, consumer groups, etc.).
	Close() error
}
