package biz

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	resourceDal "github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	resourceSchema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/compositerouting"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm/clause"
)

// Target rows are locked in the same transaction as a policy write. Model
// deletion takes the same lock before checking policy references, preventing
// a new fallback reference from appearing after the deletion check.
func (a *PolicyInvocation) validateCompositeFallback(ctx context.Context, item *schema.PolicyInvocation) error {
	if item.ModelID == "" {
		return nil
	}
	db := util.GetDB(ctx, a.PolicyInvocationDAL.DB)
	var owner resourceSchema.Model
	if err := resourceDal.GetModelDB(ctx, db).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", item.ModelID).First(&owner).Error; err != nil {
		return err
	}
	return a.ValidateCompositeFallbackForModel(ctx, &owner, item)
}

// ValidateCompositeFallbackForModel also validates a prospective model state
// before converting an existing model with invocation policies to format 2.
func (a *PolicyInvocation) ValidateCompositeFallbackForModel(ctx context.Context, owner *resourceSchema.Model, item *schema.PolicyInvocation) error {
	db := util.GetDB(ctx, a.PolicyInvocationDAL.DB)
	if owner.ModelType != resourceSchema.ModelTypeSmart || owner.SmartRouting == nil || owner.SmartRouting.RoutingMode() == compositerouting.Difficulty {
		return nil
	}
	if item.FallbackPolicy == nil {
		return nil
	}
	var fallback schema.FallbackPolicy
	if err := json.Unmarshal([]byte(*item.FallbackPolicy), &fallback); err != nil {
		return errors.BadRequest("", "备用模型链格式错误")
	}
	if len(fallback.Targets) > 50 {
		return errors.BadRequest("", "备用模型最多 50 个")
	}
	codes := append([]string(nil), fallback.Targets...)
	sort.Strings(codes)
	seen := map[string]bool{}
	for _, code := range codes {
		if code == "" || seen[code] {
			return errors.BadRequest("", "备用模型不能为空或重复")
		}
		seen[code] = true
		var target resourceSchema.Model
		if err := resourceDal.GetModelDB(ctx, db).Clauses(clause.Locking{Strength: "UPDATE"}).Where("model_code = ?", code).First(&target).Error; err != nil {
			return errors.BadRequest("", "备用模型不存在或无权使用")
		}
		if _, err := requireModelPermission(ctx, a.ModelDAL, a.DataPermissionDAL, target.ID, modelPermissionRead); err != nil {
			return errors.BadRequest("", "备用模型不存在或无权使用")
		}
		if code != target.ModelCode {
			return errors.BadRequest("", "请使用备用模型的准确编码（区分大小写）")
		}
		var types []string
		if err := json.Unmarshal([]byte(target.RequestTypes), &types); err != nil {
			return errors.BadRequest("", "备用模型协议配置不合法")
		}
		chat := false
		for _, rt := range types {
			chat = chat || rt == "chat_completion"
		}
		if target.ID == owner.ID || target.ModelType == resourceSchema.ModelTypeSmart || target.SpaceCode != owner.SpaceCode || target.Enabled != 1 || !chat {
			return errors.BadRequest("", "备用模型必须是同空间、已启用且支持对话的普通模型")
		}
	}
	return nil
}
