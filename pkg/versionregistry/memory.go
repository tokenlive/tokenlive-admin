package versionregistry

import (
	"context"
	"sync"
	"time"
)

type memoryRecord struct {
	node      Node
	expiresAt time.Time
}

type memoryStore struct {
	mu        sync.Mutex
	namespace string
	initErr   error
	now       func() time.Time
	records   map[string]memoryRecord
}

// NewMemoryStore creates an isolated, in-memory report store. An empty namespace
// defaults to "default", and a nil clock uses time.Now. Invalid configuration is
// reported by every operation.
func NewMemoryStore(namespace string, now func() time.Time) Store {
	namespace, err := configuredNamespace(namespace)
	if now == nil {
		now = time.Now
	}
	return &memoryStore{
		namespace: namespace,
		initErr:   err,
		now:       now,
		records:   make(map[string]memoryRecord),
	}
}

func (s *memoryStore) Upsert(ctx context.Context, node Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.initErr != nil {
		return s.initErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := Validate(node, s.namespace); err != nil {
		return err
	}
	node.InstanceID, _ = canonicalInstanceID(node.InstanceID)
	s.records[node.InstanceID] = memoryRecord{node: node, expiresAt: s.now().Add(TTL)}
	return nil
}

func (s *memoryStore) List(ctx context.Context) ([]Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.initErr != nil {
		return nil, s.initErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	nodes := make([]Node, 0, len(s.records))
	now := s.now()
	for instanceID, record := range s.records {
		if !now.Before(record.expiresAt) {
			delete(s.records, instanceID)
			continue
		}
		nodes = append(nodes, record.node)
	}
	return nodes, nil
}

func (s *memoryStore) Delete(ctx context.Context, instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.initErr != nil {
		return s.initErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	instanceID, err := canonicalInstanceID(instanceID)
	if err != nil {
		return err
	}
	delete(s.records, instanceID)
	return nil
}
