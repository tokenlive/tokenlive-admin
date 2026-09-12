package versionregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

const redisKeyPrefix = "tokenlive:gateway-versions:"

type redisStore struct {
	client    *redis.Client
	namespace string
	prefix    string
	initErr   error
}

// NewRedisStore borrows client without taking ownership of its lifecycle.
// Empty namespace defaults to "default"; deployments sharing a Redis DB must
// configure distinct namespaces. Invalid configuration fails every operation
// without issuing Redis commands.
func NewRedisStore(client *redis.Client, namespace string) Store {
	namespace, err := configuredNamespace(namespace)
	if err == nil && client == nil {
		err = errors.New("versionregistry: Redis client is nil")
	}
	return &redisStore{
		client:    client,
		namespace: namespace,
		prefix:    redisKeyPrefix + namespace + ":",
		initErr:   err,
	}
}

func (s *redisStore) Upsert(ctx context.Context, node Node) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if err := Validate(node, s.namespace); err != nil {
		return err
	}
	node.InstanceID, _ = canonicalInstanceID(node.InstanceID)
	payload, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("versionregistry: encode record: %w", err)
	}
	if err := s.client.Set(ctx, s.prefix+node.InstanceID, payload, TTL).Err(); err != nil {
		return fmt.Errorf("versionregistry: set record: %w", err)
	}
	return nil
}

func (s *redisStore) List(ctx context.Context) ([]Node, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	nodes := make([]Node, 0)
	seen := make(map[string]struct{})
	iter := s.client.Scan(ctx, 0, s.prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if !strings.HasPrefix(key, s.prefix) {
			return nil, errors.New("versionregistry: scan returned a key outside the configured namespace")
		}
		instanceID, err := canonicalInstanceID(strings.TrimPrefix(key, s.prefix))
		if err != nil {
			return nil, fmt.Errorf("versionregistry: invalid record key %q: %w", key, err)
		}
		if key != s.prefix+instanceID {
			return nil, fmt.Errorf("versionregistry: instance_id in record key %q is not canonical", key)
		}
		payload, err := s.client.Get(ctx, key).Bytes()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("versionregistry: get record %q: %w", key, err)
		}
		remaining, err := s.client.PTTL(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("versionregistry: read TTL for %q: %w", key, err)
		}
		// go-redis preserves Redis's -1/-2 sentinels as raw durations.
		if remaining == -2 || remaining == 0 {
			continue
		}
		if remaining < 0 {
			return nil, fmt.Errorf("versionregistry: record %q has no valid TTL", key)
		}
		var node Node
		if err := json.Unmarshal(payload, &node); err != nil {
			return nil, fmt.Errorf("versionregistry: decode record %q: %w", key, err)
		}
		if err := Validate(node, s.namespace); err != nil {
			return nil, fmt.Errorf("versionregistry: invalid record %q: %w", key, err)
		}
		node.InstanceID, _ = canonicalInstanceID(node.InstanceID)
		if node.InstanceID != instanceID {
			return nil, fmt.Errorf("versionregistry: instance_id does not match record key %q", key)
		}
		nodes = append(nodes, node)
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("versionregistry: scan records: %w", err)
	}
	return nodes, nil
}

func (s *redisStore) Delete(ctx context.Context, instanceID string) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	instanceID, err := canonicalInstanceID(instanceID)
	if err != nil {
		return err
	}
	if err := s.client.Del(ctx, s.prefix+instanceID).Err(); err != nil {
		return fmt.Errorf("versionregistry: delete record: %w", err)
	}
	return nil
}

func (s *redisStore) ready(ctx context.Context) error {
	if s.initErr != nil {
		return s.initErr
	}
	return ctx.Err()
}
