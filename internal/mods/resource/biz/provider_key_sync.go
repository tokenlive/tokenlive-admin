package biz

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

type endpointCascadeReason string

const (
	cascadeReasonUpdated endpointCascadeReason = "updated"
	cascadeReasonDisable endpointCascadeReason = "disabled"
	cascadeReasonClaimed endpointCascadeReason = "claimed"
)

type endpointCascadeMutation struct {
	before schema.Endpoint
	after  schema.Endpoint
	reason endpointCascadeReason
}

type providerKeyCascadePlan struct {
	resolvedNewKeys []schema.ApiKeyItem
	mutations       []endpointCascadeMutation
	impact          schema.ProviderKeyImpactResult
}

type endpointRouteKey struct {
	ModelID    string
	ProviderID string
	URL        string
	ApiKey     string
	Protocol   string
	RealModel  string
}

func makeEndpointRouteKey(ep *schema.Endpoint, apiKey string) endpointRouteKey {
	return endpointRouteKey{
		ModelID:    ep.ModelID,
		ProviderID: ep.ProviderID,
		URL:        ep.URL,
		ApiKey:     apiKey,
		Protocol:   ep.Protocol,
		RealModel:  ep.RealModel,
	}
}

func normalizeAuthType(authType string) string {
	trimmed := strings.TrimSpace(authType)
	if trimmed == "" {
		return "api_key"
	}
	return trimmed
}

func validateUniqueProviderAPIKeys(authType string, keys []schema.ApiKeyItem) error {
	if normalizeAuthType(authType) == "oauth_token" {
		return nil
	}
	seen := make(map[string]struct{}, len(keys))
	for _, item := range keys {
		val := strings.TrimSpace(item.Value)
		if val == "" {
			continue
		}
		if _, exists := seen[val]; exists {
			return errors.BadRequest("", "同一供应商下不能存在重复的 API Key")
		}
		seen[val] = struct{}{}
	}
	return nil
}

// resolveFormAPIKeyIDs filters blank values (for api_key auth), preserves existing key IDs
// by explicit ID first then exact value match, and assigns fresh XIDs to newly added keys.
func resolveFormAPIKeyIDs(authType string, oldKeys, newKeys []schema.ApiKeyItem) []schema.ApiKeyItem {
	if len(newKeys) == 0 {
		return nil
	}

	isOAuth := normalizeAuthType(authType) == "oauth_token"
	filtered := make([]schema.ApiKeyItem, 0, len(newKeys))
	for _, item := range newKeys {
		val := strings.TrimSpace(item.Value)
		if !isOAuth && val == "" {
			continue
		}
		filtered = append(filtered, schema.ApiKeyItem{
			ID:          strings.TrimSpace(item.ID),
			Value:       val,
			Description: item.Description,
		})
	}
	if len(filtered) == 0 {
		return nil
	}

	oldByID := make(map[string]schema.ApiKeyItem, len(oldKeys))
	for _, k := range oldKeys {
		if k.ID != "" {
			oldByID[k.ID] = k
		}
	}

	usedOldIDs := make(map[string]bool, len(oldKeys))
	usedNewIDs := make(map[string]bool, len(filtered))
	resolved := make([]schema.ApiKeyItem, len(filtered))

	// Pass 1: preserve explicit non-empty IDs from the request.
	for i, item := range filtered {
		id := item.ID
		if id != "" && !usedNewIDs[id] {
			usedNewIDs[id] = true
			if _, ok := oldByID[id]; ok {
				usedOldIDs[id] = true
			}
		} else {
			id = ""
		}
		resolved[i] = schema.ApiKeyItem{
			ID:          id,
			Value:       item.Value,
			Description: item.Description,
		}
	}

	// Pass 2: match omitted IDs by exact key value (supports reordering or legacy API callers).
	for i := range resolved {
		if resolved[i].ID != "" || resolved[i].Value == "" {
			continue
		}
		for _, oldItem := range oldKeys {
			if oldItem.ID == "" || usedOldIDs[oldItem.ID] {
				continue
			}
			if oldItem.Value == resolved[i].Value {
				resolved[i].ID = oldItem.ID
				usedOldIDs[oldItem.ID] = true
				usedNewIDs[oldItem.ID] = true
				break
			}
		}
	}

	// Pass 3: assign fresh XIDs to newly added non-empty keys.
	for i := range resolved {
		if resolved[i].ID == "" && resolved[i].Value != "" {
			resolved[i].ID = util.NewXID()
		}
	}

	return resolved
}

func planProviderKeyCascade(beforeProvider *schema.Provider, formItem *schema.ProviderForm, endpoints schema.Endpoints) (*providerKeyCascadePlan, error) {
	oldAuthType := normalizeAuthType(beforeProvider.AuthType)
	newAuthType := normalizeAuthType(formItem.AuthType)

	if err := validateUniqueProviderAPIKeys(newAuthType, formItem.ApiKeys); err != nil {
		return nil, err
	}

	oldKeys, _ := schema.EnsureAPIKeyIDs(beforeProvider.GetApiKeys())
	resolvedNewKeys := resolveFormAPIKeyIDs(newAuthType, oldKeys, formItem.ApiKeys)

	oldValueToID := make(map[string]string, len(oldKeys))
	for _, k := range oldKeys {
		if k.Value != "" && k.ID != "" {
			if _, exists := oldValueToID[k.Value]; !exists {
				oldValueToID[k.Value] = k.ID
			}
		}
	}

	newByID := make(map[string]schema.ApiKeyItem, len(resolvedNewKeys))
	for _, k := range resolvedNewKeys {
		if k.ID != "" {
			newByID[k.ID] = k
		}
	}

	updatedKeys := make(map[string]string)
	deletedKeyIDs := make(map[string]bool)
	if oldAuthType != "oauth_token" {
		for _, oldKey := range oldKeys {
			if oldKey.ID == "" {
				continue
			}
			if newAuthType == "oauth_token" {
				deletedKeyIDs[oldKey.ID] = true
				continue
			}
			newItem, exists := newByID[oldKey.ID]
			if !exists {
				deletedKeyIDs[oldKey.ID] = true
				continue
			}
			if newItem.Value != "" && oldKey.Value != newItem.Value {
				updatedKeys[oldKey.ID] = newItem.Value
			}
		}
	}

	plan := &providerKeyCascadePlan{
		resolvedNewKeys: resolvedNewKeys,
	}

	simulated := make(map[string]*schema.Endpoint, len(endpoints))
	mutatedIDs := make(map[string]bool)

	for _, ep := range endpoints {
		if ep == nil {
			continue
		}
		after := *ep
		if normalizeAuthType(ep.AuthType) == "oauth_token" {
			simulated[ep.ID] = &after
			continue
		}

		boundKeyID := strings.TrimSpace(ep.ProviderKeyID)
		// Fallback claim for legacy endpoints lacking provider_key_id.
		if boundKeyID == "" && oldAuthType != "oauth_token" {
			if trimmedKey := strings.TrimSpace(ep.ApiKey); trimmedKey != "" {
				if claimedID, ok := oldValueToID[trimmedKey]; ok {
					boundKeyID = claimedID
				}
			}
		}

		if boundKeyID == "" {
			// Key Inheritance (api_key == "") or Key Override (custom api_key).
			simulated[ep.ID] = &after
			continue
		}

		isDeleted := deletedKeyIDs[boundKeyID]
		if !isDeleted && newAuthType == "oauth_token" {
			isDeleted = true
		} else if !isDeleted && oldAuthType != "oauth_token" {
			if _, exists := newByID[boundKeyID]; !exists {
				isDeleted = true
			}
		}

		if isDeleted {
			if ep.Enabled == 1 {
				plan.impact.AffectedEndpointCount++
				plan.impact.AffectedEndpoints = append(plan.impact.AffectedEndpoints, schema.AffectedEndpointInfo{
					ID:            ep.ID,
					Code:          ep.Code,
					ModelID:       ep.ModelID,
					ProviderKeyID: boundKeyID,
				})
			}
			if ep.Enabled != 0 || strings.TrimSpace(ep.ProviderKeyID) != "" {
				after.Enabled = 0
				after.ProviderKeyID = ""
				plan.mutations = append(plan.mutations, endpointCascadeMutation{
					before: *ep,
					after:  after,
					reason: cascadeReasonDisable,
				})
			}
			simulated[ep.ID] = &after
			continue
		}

		if newVal, ok := updatedKeys[boundKeyID]; ok {
			after.ProviderKeyID = boundKeyID
			after.ApiKey = newVal
			plan.mutations = append(plan.mutations, endpointCascadeMutation{
				before: *ep,
				after:  after,
				reason: cascadeReasonUpdated,
			})
			mutatedIDs[ep.ID] = true
		} else if strings.TrimSpace(ep.ProviderKeyID) == "" {
			after.ProviderKeyID = boundKeyID
			plan.mutations = append(plan.mutations, endpointCascadeMutation{
				before: *ep,
				after:  after,
				reason: cascadeReasonClaimed,
			})
		}

		simulated[ep.ID] = &after
	}

	if newAuthType != "oauth_token" && len(mutatedIDs) > 0 {
		seenRoutes := make(map[endpointRouteKey]*schema.Endpoint, len(endpoints))
		for _, ep := range endpoints {
			if ep == nil {
				continue
			}
			sim := simulated[ep.ID]
			key := makeEndpointRouteKey(sim, sim.ApiKey)
			if existing, dup := seenRoutes[key]; dup {
				if mutatedIDs[ep.ID] || mutatedIDs[existing.ID] {
					return nil, errors.Conflict("", "级联更新后的 API Key 与模型下已有端点（%s 与 %s）配置重复，请检查后重试", existing.Code, sim.Code)
				}
			} else {
				seenRoutes[key] = sim
			}
		}
	}

	return plan, nil
}

// PreviewKeyImpact inspects which enabled bound endpoints would be auto-disabled by the proposed provider update.
func (p *Provider) PreviewKeyImpact(ctx context.Context, id string, formItem *schema.ProviderForm) (*schema.ProviderKeyImpactResult, error) {
	provider, err := p.ProviderDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if provider == nil {
		return nil, errors.NotFound("", "Provider not found")
	}

	var endpoints schema.Endpoints
	if p.EndpointDAL != nil {
		endpoints, err = p.EndpointDAL.ListByProviderID(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	plan, err := planProviderKeyCascade(provider, formItem, endpoints)
	if err != nil {
		return nil, err
	}
	return &plan.impact, nil
}

// BackfillProviderKeyBindings idempotently assigns XIDs to provider api_keys entries
// missing an ID and claims legacy endpoints whose api_key matches a provider key value.
func BackfillProviderKeyBindings(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if !db.Migrator().HasColumn(&schema.Endpoint{}, "provider_key_id") {
		return nil
	}

	return (&util.Trans{DB: db}).Exec(ctx, func(ctx context.Context) error {
		tx := util.GetDB(ctx, db)
		var providers schema.Providers
		if err := tx.Model(new(schema.Provider)).Where("deleted = '0'").Find(&providers).Error; err != nil {
			return errors.WithStack(err)
		}

		for _, provider := range providers {
			keys, changed := schema.EnsureAPIKeyIDs(provider.GetApiKeys())
			if changed {
				raw, err := json.Marshal(keys)
				if err != nil {
					return errors.WithStack(err)
				}
				if err := tx.Model(new(schema.Provider)).
					Where("id = ? AND deleted = '0'", provider.ID).
					Update("api_keys", util.RawJSON(raw)).Error; err != nil {
					return errors.WithStack(err)
				}
			}

			if normalizeAuthType(provider.AuthType) == "oauth_token" || len(keys) == 0 {
				continue
			}

			valueToKeyID := make(map[string]string, len(keys))
			for _, k := range keys {
				if k.Value != "" && k.ID != "" {
					if _, exists := valueToKeyID[k.Value]; !exists {
						valueToKeyID[k.Value] = k.ID
					}
				}
			}
			if len(valueToKeyID) == 0 {
				continue
			}

			var endpoints schema.Endpoints
			if err := tx.Model(new(schema.Endpoint)).
				Where("provider_id = ? AND deleted = '0' AND (provider_key_id IS NULL OR provider_key_id = '') AND (auth_type IS NULL OR auth_type != 'oauth_token') AND api_key IS NOT NULL AND api_key != ''", provider.ID).
				Find(&endpoints).Error; err != nil {
				return errors.WithStack(err)
			}

			for _, ep := range endpoints {
				keyID, ok := valueToKeyID[strings.TrimSpace(ep.ApiKey)]
				if !ok {
					continue
				}
				if err := tx.Model(new(schema.Endpoint)).
					Where("id = ? AND deleted = '0'", ep.ID).
					Update("provider_key_id", keyID).Error; err != nil {
					return errors.WithStack(err)
				}
			}
		}

		return nil
	})
}
