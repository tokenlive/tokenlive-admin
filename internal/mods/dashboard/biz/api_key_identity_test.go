package biz

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
	opsbiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
)

type fixturePortal struct {
	mu    sync.Mutex
	calls int
	list  func(context.Context, string) ([]opsbiz.PortalWorkspaceAPIKey, error)
}

func (p *fixturePortal) ListWorkspaceAPIKeys(ctx context.Context, workspace string) ([]opsbiz.PortalWorkspaceAPIKey, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	return p.list(ctx, workspace)
}

func TestCanonicalKeysNeverUsesDisplayOrOverridesHash(t *testing.T) {
	a, b, unknown := schema.KeyRef{Hash: "a"}, schema.KeyRef{Hash: "b"}, schema.KeyRef{}
	result := CanonicalKeys([]schema.Candidate{{Ref: a, Display: "same"}, {Ref: b, Display: "same"}, {Ref: unknown}},
		map[schema.KeyRef]string{a: "wrong"})
	require.Equal(t, "h:a", result.Keys[a])
	require.Equal(t, "h:b", result.Keys[b])
	require.Empty(t, result.Keys[unknown])
}

func TestIdentityPortalAliasRequiresVerifiedWorkspaceAndUniqueHash(t *testing.T) {
	portal := &fixturePortal{list: func(_ context.Context, workspace string) ([]opsbiz.PortalWorkspaceAPIKey, error) {
		if workspace == "ws" {
			return []opsbiz.PortalWorkspaceAPIKey{{ID: "id", Name: "App", Status: "enabled"}}, nil
		}
		return nil, nil
	}}
	resolver := NewIdentityResolver(nil, portal, "", time.Now)
	withHash := schema.KeyRef{Hash: "a", KeyID: "id", WorkspaceID: "ws"}
	withoutHash := schema.KeyRef{KeyID: "id", WorkspaceID: "ws"}
	foreign := schema.KeyRef{KeyID: "id", WorkspaceID: "other"}
	rows := []schema.Candidate{{Ref: withHash}, {Ref: withoutHash}, {Ref: foreign}}
	result := resolver.Resolve(context.Background(), rows)
	require.Equal(t, "h:a", result.Keys[withoutHash])
	require.Empty(t, result.Keys[foreign])
	rows = append(rows, schema.Candidate{Ref: schema.KeyRef{Hash: "b", KeyID: "id", WorkspaceID: "ws"}})
	result = resolver.Resolve(context.Background(), rows)
	require.Empty(t, result.Keys[withoutHash])
	require.Equal(t, "h:a", result.Keys[withHash])
}

func TestIdentityValidatedPortalIDWithoutHashIsScopedAndStable(t *testing.T) {
	portal := &fixturePortal{list: func(context.Context, string) ([]opsbiz.PortalWorkspaceAPIKey, error) {
		return []opsbiz.PortalWorkspaceAPIKey{{ID: "same", Status: "revoked"}}, nil
	}}
	resolver := NewIdentityResolver(nil, portal, "", time.Now)
	a := schema.KeyRef{WorkspaceID: "a", KeyID: "same"}
	b := schema.KeyRef{WorkspaceID: "b", KeyID: "same"}
	result := resolver.Resolve(context.Background(), []schema.Candidate{{Ref: a}, {Ref: b}})
	require.NotEmpty(t, result.Keys[a])
	require.NotEqual(t, result.Keys[a], result.Keys[b])
}

func TestIdentityPortalFailurePreservesHashAndCanRecover(t *testing.T) {
	fail := true
	portal := &fixturePortal{list: func(context.Context, string) ([]opsbiz.PortalWorkspaceAPIKey, error) {
		if fail {
			return nil, errors.New("offline")
		}
		return []opsbiz.PortalWorkspaceAPIKey{{ID: "id", Status: "enabled"}}, nil
	}}
	resolver := NewIdentityResolver(nil, portal, "", time.Now)
	ref := schema.KeyRef{WorkspaceID: "ws", KeyID: "id"}
	hashRef := schema.KeyRef{WorkspaceID: "ws", KeyID: "id", Hash: "a"}
	rows := []schema.Candidate{{Ref: ref}, {Ref: hashRef}}
	result := resolver.Resolve(context.Background(), rows)
	require.Empty(t, result.Keys[ref])
	require.Equal(t, "h:a", result.Keys[hashRef])
	require.Contains(t, result.Warnings, "metadata_unavailable")
	fail = false
	result = resolver.Resolve(context.Background(), rows)
	require.Equal(t, "h:a", result.Keys[ref])
}
