package biz

import (
	"context"
	"fmt"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	policyDal "github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	resourceDal "github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	resourceSchema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

const (
	modelPermissionRead  uint = 0b001
	modelPermissionWrite uint = 0b010
)

func requireModelPermission(ctx context.Context, modelDAL *resourceDal.Model, dataPermissionDAL *resourceDal.DataPermission, modelID string, permission uint) (*resourceSchema.Model, error) {
	if modelID == "" {
		return nil, errors.BadRequest("", "model_id is required")
	}
	model, err := modelDAL.Get(ctx, modelID)
	if err != nil {
		return nil, err
	} else if model == nil {
		return nil, errors.NotFound("", "Model not found")
	}
	if util.FromIsRootUser(ctx) {
		return model, nil
	}
	permTable := config.C.FormatTableName("data_permission")
	ok, err := util.Exists(ctx, util.GetDB(ctx, dataPermissionDAL.DB).Table(permTable).
		Where("type = ? AND data_id = ? AND user = ? AND tenant = ? AND permission & ? = ?",
			resourceSchema.DataPermissionTypeModel, modelID, util.FromUsername(ctx), util.FromTenant(ctx), permission, permission))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if !ok {
		return nil, errors.NotFound("", "Model not found")
	}
	return model, nil
}

func requireExistingModelPolicy(ctx context.Context, modelDAL *resourceDal.Model, dataPermissionDAL *resourceDal.DataPermission, modelID string, permission uint) (*resourceSchema.Model, error) {
	if modelID == "" {
		return nil, nil
	}
	return requireModelPermission(ctx, modelDAL, dataPermissionDAL, modelID, permission)
}

func rejectPolicyKindChange(oldModelID, newModelID string) error {
	if oldModelID == "" && newModelID != "" {
		return errors.BadRequest("", "Policy template cannot be converted to model policy; copy it to a model instead")
	}
	if oldModelID != "" && newModelID == "" {
		return errors.BadRequest("", "Model policy cannot be converted to policy template")
	}
	return nil
}

func nextPolicyName(ctx context.Context, baseName, modelID string, exists func(context.Context, string, string) (bool, error)) (string, error) {
	if baseName == "" {
		return "", errors.BadRequest("", "Name is required")
	}
	name := baseName
	for i := 2; ; i++ {
		ok, err := exists(ctx, modelID, name)
		if err != nil {
			return "", err
		}
		if !ok {
			return name, nil
		}
		name = fmt.Sprintf("%s-%d", baseName, i)
	}
}

func createModelPolicyBinding(ctx context.Context, bindingDAL *policyDal.PolicyBinding, policyType, policyID string, model *resourceSchema.Model) error {
	if err := bindingDAL.CleanDeletedConflict(ctx, "", "", model.ModelCode, policyType, policyID); err != nil {
		return err
	}
	binding := &policySchema.PolicyBinding{
		ID:         util.NewXID(),
		ModelCode:  model.ModelCode,
		PolicyType: policyType,
		PolicyID:   policyID,
		Deleted:    "0",
	}
	username := util.FromUsername(ctx)
	if username != "" {
		binding.Creator = &username
	}
	return bindingDAL.Create(ctx, binding)
}

func replaceModelPolicyBinding(ctx context.Context, bindingDAL *policyDal.PolicyBinding, policyType, policyID string, model *resourceSchema.Model) error {
	if err := bindingDAL.DeleteByPolicyID(ctx, policyType, policyID); err != nil {
		return err
	}
	return createModelPolicyBinding(ctx, bindingDAL, policyType, policyID, model)
}
