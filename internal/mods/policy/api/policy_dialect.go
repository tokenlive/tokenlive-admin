package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

type PolicyDialect struct {
	PolicyDialectBIZ *biz.PolicyDialect
}

// @Tags PolicyDialectAPI
// @Security ApiKeyAuth
// @Summary List dialect adaptations for an ordinary model
// @Param model_id query string true "Model ID"
// @Param current query int false "Page number"
// @Param pageSize query int false "Page size"
// @Success 200 {object} util.ResponseResult{data=[]schema.PolicyDialect}
// @Router /api/v1/policy/policy-dialects [get]
func (a *PolicyDialect) Query(c *gin.Context) {
	var query schema.PolicyDialectQuery
	if err := util.ParseQuery(c, &query); err != nil {
		util.ResError(c, err)
		return
	}
	items, page, err := a.PolicyDialectBIZ.Query(c.Request.Context(), query)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, items, page)
}

// @Tags PolicyDialectAPI
// @Security ApiKeyAuth
// @Summary Create a dialect adaptation with an optional initial enabled state
// @Param body body schema.PolicyDialectForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=biz.PolicyDialectCreateResult}
// @Router /api/v1/policy/policy-dialects [post]
func (a *PolicyDialect) Create(c *gin.Context) {
	var form schema.PolicyDialectForm
	if err := util.ParseJSON(c, &form); err != nil {
		util.ResError(c, err)
		return
	}
	item, err := a.PolicyDialectBIZ.Create(c.Request.Context(), form)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags PolicyDialectAPI
// @Security ApiKeyAuth
// @Summary Enable or disable a dialect adaptation
// @Param id path string true "Policy ID"
// @Param body body schema.PolicyDialectEnabledForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.ModelMutationResult}
// @Router /api/v1/policy/policy-dialects/{id}/enabled [put]
func (a *PolicyDialect) UpdateEnabled(c *gin.Context) {
	var form schema.PolicyDialectEnabledForm
	if err := util.ParseJSON(c, &form); err != nil {
		util.ResError(c, err)
		return
	}
	result, err := a.PolicyDialectBIZ.ToggleEnabled(c.Request.Context(), c.Param("id"), *form.Enabled)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags PolicyDialectAPI
// @Security ApiKeyAuth
// @Summary Delete a dialect adaptation
// @Param id path string true "Policy ID"
// @Success 200 {object} util.ResponseResult{data=schema.ModelMutationResult}
// @Router /api/v1/policy/policy-dialects/{id} [delete]
func (a *PolicyDialect) Delete(c *gin.Context) {
	result, err := a.PolicyDialectBIZ.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}
