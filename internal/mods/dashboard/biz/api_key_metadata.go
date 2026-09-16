package biz

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
	rbac "github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaykeys"
)

type localKeyIdentity struct {
	userID string
	hash   string
}

// Only hashes and safe metadata are cached, never the stored plaintext keys.
type localSnapshot struct {
	byHash    map[string]schema.Metadata
	byID      map[string]localKeyIdentity
	tenants   map[string]string
	conflicts map[string]bool
}

func (r *IdentityResolver) localMetadata(ctx context.Context) *localSnapshot {
	r.mu.Lock()
	snapshot, expires := r.local, r.localExpiry
	r.mu.Unlock()
	if snapshot != nil && r.now().Before(expires) {
		return snapshot
	}
	if r.db == nil {
		return nil
	}
	var keys []rbac.UserAPIKey
	var tenants []rbac.Tenant
	var users []rbac.User
	if r.db.WithContext(ctx).Unscoped().Select("id", "user_id", "name", "api_key", "status", "expires_at", "deleted", "deleted_at").Find(&keys).Error != nil ||
		r.db.WithContext(ctx).Unscoped().Select("id", "code", "name", "api_key", "status", "deleted", "deleted_at").Find(&tenants).Error != nil ||
		r.db.WithContext(ctx).Unscoped().Select("id", "name", "username").Find(&users).Error != nil {
		return nil
	}
	names := map[string]string{}
	for _, user := range users {
		names[user.ID] = user.Name
		if names[user.ID] == "" {
			names[user.ID] = user.Username
		}
	}
	names[config.C.General.Root.ID] = config.C.General.Root.Name
	snapshot = &localSnapshot{
		byHash: map[string]schema.Metadata{}, byID: map[string]localKeyIdentity{},
		tenants: map[string]string{}, conflicts: map[string]bool{},
	}
	put := func(hash string, meta schema.Metadata) {
		if _, exists := snapshot.byHash[hash]; exists {
			snapshot.conflicts[hash] = true
		}
		snapshot.byHash[hash] = meta
	}
	for _, key := range keys {
		if r.pepper == "" || key.APIKey == "" {
			continue
		}
		hash := gatewaykeys.HashAPIKey(key.APIKey, r.pepper)
		snapshot.byID[key.ID] = localKeyIdentity{userID: key.UserID, hash: hash}
		status := "unknown"
		switch key.Status {
		case 1:
			status = "enabled"
		case 2:
			status = "disabled"
		}
		put(hash, schema.Metadata{
			KeyName: key.Name, Display: rbac.MaskAPIKey(key.APIKey), Source: "admin_user",
			KeyStatus: currentKeyStatus(status, key.Deleted, key.DeletedAt, key.ExpiresAt, r.now()), MetadataStatus: "ready",
			Owner: schema.Owner{Kind: "user", ID: key.UserID, Name: names[key.UserID]},
		})
	}
	for _, tenant := range tenants {
		snapshot.tenants[tenant.Code] = tenant.Name
		if r.pepper == "" || tenant.APIKey == "" {
			continue
		}
		status := "disabled"
		if tenant.Status == rbac.TenantStatusActivated {
			status = "enabled"
		}
		put(gatewaykeys.HashAPIKey(tenant.APIKey, r.pepper), schema.Metadata{
			KeyName: tenant.Name + " · API Key", Display: rbac.MaskAPIKey(tenant.APIKey), Source: "tenant",
			KeyStatus: currentKeyStatus(status, tenant.Deleted, tenant.DeletedAt, nil, r.now()), MetadataStatus: "ready",
			Owner: schema.Owner{Kind: "tenant", ID: tenant.Code, Name: tenant.Name},
		})
	}
	r.mu.Lock()
	r.local, r.localExpiry = snapshot, r.now().Add(time.Minute)
	r.mu.Unlock()
	return snapshot
}

func currentKeyStatus(status, deleted string, deletedAt, expiresAt *time.Time, now time.Time) string {
	if (deleted != "" && deleted != "0") || deletedAt != nil {
		return "deleted"
	}
	if status == "revoked" {
		return status
	}
	if expiresAt != nil && !expiresAt.After(now) {
		return "expired"
	}
	switch status {
	case "enabled", "active":
		return "enabled"
	case "disabled":
		return "disabled"
	default:
		return "unknown"
	}
}

func unknownMetadata(group schema.Group) schema.Metadata {
	meta := schema.Metadata{Display: group.Display, Source: "unknown", KeyStatus: "unknown", MetadataStatus: "unavailable",
		Owner: schema.Owner{Kind: "unknown"}}
	switch {
	case group.Ref.WorkspaceID != "":
		meta.Owner = schema.Owner{Kind: "workspace", ID: group.Ref.WorkspaceID}
	case group.Ref.UserID != "":
		meta.Owner = schema.Owner{Kind: "user", ID: group.Ref.UserID}
	case group.Ref.TenantID != "":
		meta.Owner = schema.Owner{Kind: "tenant", ID: group.Ref.TenantID}
	}
	return meta
}

func (r *IdentityResolver) Describe(ctx context.Context, groups []schema.Group) map[string]schema.Metadata {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	workspaces := map[string]bool{}
	needLocal := false
	for _, group := range groups {
		if group.Ref.WorkspaceID != "" {
			workspaces[group.Ref.WorkspaceID] = true
		} else {
			needLocal = true
		}
	}
	lists, _ := r.fetchPortals(ctx, workspaces)
	var local *localSnapshot
	if needLocal {
		local = r.localMetadata(ctx)
	}
	result := make(map[string]schema.Metadata, len(groups))
	for _, group := range groups {
		meta := unknownMetadata(group)
		if group.Ref.WorkspaceID != "" {
			for _, key := range lists[group.Ref.WorkspaceID] {
				if key.ID == group.Ref.KeyID {
					meta = schema.Metadata{
						KeyName: key.Name, Display: key.KeyPrefix + "****" + key.SecretLast4, Source: "portal_workspace",
						KeyStatus: currentKeyStatus(key.Status, "", nil, key.ExpiresAt, r.now()), MetadataStatus: "ready",
						Owner: schema.Owner{Kind: "workspace", ID: group.Ref.WorkspaceID},
					}
					break
				}
			}
		} else if local != nil {
			if item, ok := local.byHash[group.Ref.Hash]; ok && !local.conflicts[group.Ref.Hash] {
				meta = item
			} else if group.Ref.UserID == "" && group.Ref.TenantID != "" {
				if name, ok := local.tenants[group.Ref.TenantID]; ok {
					meta.Owner.Name, meta.Source, meta.MetadataStatus = name, "tenant", "partial"
				}
			}
		}
		result[group.Canonical] = meta
	}
	return result
}
