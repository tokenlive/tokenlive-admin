package biz

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
)

func ensurePolicyUnbound(ctx context.Context, bindingDAL *dal.PolicyBinding, policyType, policyID string) error {
	isBound, err := bindingDAL.ExistsByPolicyID(ctx, policyType, policyID)
	if err != nil {
		return err
	}
	if isBound {
		return errors.BadRequest("", "策略已被绑定，请先到对应模型下解绑后再执行删除操作")
	}
	return nil
}
