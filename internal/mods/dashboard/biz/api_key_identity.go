package biz

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
	opsbiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	"gorm.io/gorm"
)

type PortalKeys interface {
	ListWorkspaceAPIKeys(context.Context, string) ([]opsbiz.PortalWorkspaceAPIKey, error)
}

type portalCacheEntry struct {
	keys    []opsbiz.PortalWorkspaceAPIKey
	expires time.Time
}

type IdentityResolver struct {
	db          *gorm.DB
	portal      PortalKeys
	pepper      string
	now         func() time.Time
	mu          sync.Mutex
	local       *localSnapshot
	localExpiry time.Time
	portals     map[string]portalCacheEntry
	portalSlots chan struct{}
}

func NewIdentityResolver(db *gorm.DB, portal PortalKeys, pepper string, now func() time.Time) *IdentityResolver {
	if now == nil {
		now = time.Now
	}
	return &IdentityResolver{
		db: db, portal: portal, pepper: pepper, now: now,
		portals: make(map[string]portalCacheEntry), portalSlots: make(chan struct{}, 4),
	}
}

func CanonicalKeys(rows []schema.Candidate, verified map[schema.KeyRef]string) schema.Resolution {
	result := schema.Resolution{Keys: make(map[schema.KeyRef]string), Warnings: []string{}}
	for _, row := range rows {
		if row.Ref.Hash != "" {
			result.Keys[row.Ref] = "h:" + row.Ref.Hash
		} else if key := verified[row.Ref]; key != "" {
			result.Keys[row.Ref] = key
		}
	}
	return result
}

func portalIdentity(workspace, id string) string {
	encoded, _ := json.Marshal([]string{"portal_workspace", workspace, id})
	return "id:" + string(encoded)
}

func (r *IdentityResolver) Resolve(ctx context.Context, rows []schema.Candidate) schema.Resolution {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	workspaces := map[string]bool{}
	aliases := map[string]map[string]bool{}
	needLocal := false
	for _, row := range rows {
		ref := row.Ref
		if ref.WorkspaceID != "" && ref.KeyID != "" {
			id := portalIdentity(ref.WorkspaceID, ref.KeyID)
			if ref.Hash != "" {
				if aliases[id] == nil {
					aliases[id] = map[string]bool{}
				}
				aliases[id][ref.Hash] = true
			} else {
				workspaces[ref.WorkspaceID] = true
			}
		} else if ref.Hash == "" && ref.KeyID != "" && ref.UserID != "" {
			needLocal = true
		}
	}
	portalLists, unavailable := r.fetchPortals(ctx, workspaces)
	var local *localSnapshot
	if needLocal {
		local = r.localMetadata(ctx)
		unavailable = unavailable || local == nil
	}
	verified := map[schema.KeyRef]string{}
	for _, row := range rows {
		ref := row.Ref
		if ref.Hash != "" || ref.KeyID == "" {
			continue
		}
		if ref.WorkspaceID != "" {
			for _, key := range portalLists[ref.WorkspaceID] {
				if key.ID != ref.KeyID {
					continue
				}
				id := portalIdentity(ref.WorkspaceID, ref.KeyID)
				switch len(aliases[id]) {
				case 0:
					verified[ref] = id
				case 1:
					for hash := range aliases[id] {
						verified[ref] = "h:" + hash
					}
				}
			}
		} else if local != nil {
			if key, ok := local.byID[ref.KeyID]; ok && key.userID == ref.UserID && key.hash != "" {
				verified[ref] = "h:" + key.hash
			}
		}
	}
	result := CanonicalKeys(rows, verified)
	if unavailable {
		result.Warnings = append(result.Warnings, "metadata_unavailable")
	}
	return result
}

func (r *IdentityResolver) portalList(ctx context.Context, workspace string) ([]opsbiz.PortalWorkspaceAPIKey, bool) {
	r.mu.Lock()
	entry, ok := r.portals[workspace]
	r.mu.Unlock()
	if ok && r.now().Before(entry.expires) {
		return entry.keys, true
	}
	if r.portal == nil {
		return nil, false
	}
	select {
	case r.portalSlots <- struct{}{}:
		defer func() { <-r.portalSlots }()
	case <-ctx.Done():
		return nil, false
	}
	keys, err := r.portal.ListWorkspaceAPIKeys(ctx, workspace)
	if err != nil {
		return nil, false
	}
	r.mu.Lock()
	// Expired workspace entries are removed so historical traffic cannot grow
	// this cache without bound. Empty/error lookups are intentionally not cached.
	for id, cached := range r.portals {
		if !r.now().Before(cached.expires) {
			delete(r.portals, id)
		}
	}
	if len(keys) > 0 {
		r.portals[workspace] = portalCacheEntry{keys: keys, expires: r.now().Add(time.Minute)}
	}
	r.mu.Unlock()
	return keys, true
}

func (r *IdentityResolver) fetchPortals(ctx context.Context, workspaces map[string]bool) (map[string][]opsbiz.PortalWorkspaceAPIKey, bool) {
	result := make(map[string][]opsbiz.PortalWorkspaceAPIKey)
	ids := make([]string, 0, len(workspaces))
	for id := range workspaces {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	jobs := make(chan string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	unavailable := false
	for i := 0; i < min(4, len(ids)); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				keys, ok := r.portalList(ctx, id)
				mu.Lock()
				if ok {
					result[id] = keys
				} else {
					unavailable = true
				}
				mu.Unlock()
			}
		}()
	}
	for _, id := range ids {
		select {
		case jobs <- id:
		case <-ctx.Done():
			mu.Lock()
			unavailable = true
			mu.Unlock()
		}
		if ctx.Err() != nil {
			break
		}
	}
	close(jobs)
	wg.Wait()
	return result, unavailable
}
