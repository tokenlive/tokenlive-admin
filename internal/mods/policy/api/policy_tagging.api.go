package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Tagging policy management
type PolicyTagging struct {
	PolicyTaggingBIZ *biz.PolicyTagging
}

// @Tags PolicyTaggingAPI
// @Security ApiKeyAuth
// @Summary Query policy tagging list
// @Success 200 {object} util.ResponseResult{data=[]schema.PolicyTagging}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-taggings [get]
func (a *PolicyTagging) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.PolicyTaggingQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyTaggingBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags PolicyTaggingAPI
// @Security ApiKeyAuth
// @Summary Get policy tagging record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyTaggingForm}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-taggings/{id} [get]
func (a *PolicyTagging) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.PolicyTaggingBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags PolicyTaggingAPI
// @Security ApiKeyAuth
// @Summary Create policy tagging record
// @Param body body schema.PolicyTaggingForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyTagging}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-taggings [post]
func (a *PolicyTagging) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyTaggingForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyTaggingBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags PolicyTaggingAPI
// @Security ApiKeyAuth
// @Summary Update policy tagging record by ID
// @Param id path string true "unique id"
// @Param body body schema.PolicyTaggingForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-taggings/{id} [put]
func (a *PolicyTagging) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyTaggingForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.PolicyTaggingBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PolicyTaggingAPI
// @Security ApiKeyAuth
// @Summary Delete policy tagging record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-taggings/{id} [delete]
func (a *PolicyTagging) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.PolicyTaggingBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
