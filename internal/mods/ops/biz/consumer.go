package biz

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"go.uber.org/zap"
)

// WSBroadcaster is an interface for broadcasting events via WebSocket.
type WSBroadcaster interface {
	Broadcast(data []byte)
}

// Consumer reads events from a message transport and persists them to MySQL.
type Consumer struct {
	Subscriber EventSubscriber
	EventDAL   *dal.EventLog
	hub        WSBroadcaster
}

// SetHub sets the WebSocket broadcaster for the consumer.
func (c *Consumer) SetHub(hub WSBroadcaster) {
	c.hub = hub
}

// Start begins the consumer goroutine.
func (c *Consumer) Start(ctx context.Context) {
	go c.run(ctx)
}

func (c *Consumer) run(ctx context.Context) {
	logging.Context(ctx).Info("event consumer starting")

	err := c.Subscriber.Subscribe(ctx, func(ctx context.Context, msg *Message) error {
		event := parseMessage(msg)
		if event == nil {
			return nil
		}

		// Persist to MySQL
		if err := c.EventDAL.Create(ctx, event); err != nil {
			logging.Context(ctx).Error("event consumer insert failed",
				zap.String("id", msg.ID),
				zap.Error(err),
			)
			return err // Signal transport to redeliver
		}

		// Broadcast via WebSocket
		if c.hub != nil {
			c.broadcastEvent(event)
		}

		return nil
	})

	if err != nil {
		logging.Context(ctx).Error("event subscriber exited with error", zap.Error(err))
	}
}

func parseMessage(msg *Message) *schema.EventLog {
	fields := msg.Fields

	eventType, _ := fields["event_type"].(string)
	if eventType == "" {
		return nil
	}

	event := &schema.EventLog{
		ID:           util.NewXID(),
		EventType:    eventType,
		TenantCode:   strField(fields, "tenant_code"),
		ModelCode:    strField(fields, "model_code"),
		EndpointID:   strField(fields, "endpoint_id"),
		ProviderName: strField(fields, "provider_name"),
		PolicyID:     strField(fields, "policy_id"),
		PolicyName:   strField(fields, "policy_name"),
		RequestID:    strField(fields, "request_id"),
		TraceID:      strField(fields, "trace_id"),
		Message:      strField(fields, "message"),
	}

	// Parse numeric fields
	if v := strField(fields, "threshold"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			event.Threshold = &f
		}
	}
	if v := strField(fields, "current_value"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			event.CurrentValue = &f
		}
	}

	// Parse timestamp
	if v := strField(fields, "ts"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			event.EventTime = time.Unix(ts, 0)
		}
	}
	if event.EventTime.IsZero() {
		event.EventTime = time.Now()
	}

	return event
}

func (c *Consumer) broadcastEvent(event *schema.EventLog) {
	defer func() {
		if r := recover(); r != nil {
			logging.Context(context.Background()).Warn("event broadcast panic", zap.Any("recover", r))
		}
	}()

	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	c.hub.Broadcast(data)
}

func strField(fields map[string]interface{}, key string) string {
	v, _ := fields[key].(string)
	return v
}
