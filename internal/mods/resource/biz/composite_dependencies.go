package biz

import (
	"context"
	"encoding/json"
	"fmt"

	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Include every scope. Disabled policies retain references for safe re-enabling,
// but only enabled policies contribute endpoints to a runtime export.
func compositeFallbackCodes(ctx context.Context, db *gorm.DB, modelID string, enabledOnly bool) ([]string, error) {
	query := util.GetDB(ctx, db).Where("model_id = ? AND deleted = '0'", modelID)
	if enabledOnly {
		query = query.Where("enabled = 1")
	}
	var policies []policySchema.PolicyInvocation
	if err := query.Find(&policies).Error; err != nil {
		return nil, err
	}
	var codes []string
	seen := map[string]bool{}
	for _, p := range policies {
		if p.FallbackPolicy == nil {
			continue
		}
		var fallback policySchema.FallbackPolicy
		if err := json.Unmarshal([]byte(*p.FallbackPolicy), &fallback); err != nil {
			return nil, fmt.Errorf("invalid composite fallback policy %s: %w", p.ID, err)
		}
		for _, code := range fallback.Targets {
			if code != "" && !seen[code] {
				codes = append(codes, code)
				seen[code] = true
			}
		}
	}
	return codes, nil
}
