package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Data permission management
type DataPermission struct {
	DataPermissionBIZ *biz.DataPermission
}

// @Tags DataPermissionAPI
// @Security ApiKeyAuth
// @Summary Query data permission list
// @Success 200 {object} util.ResponseResult{data=[]schema.DataPermission}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/resource/data-permissions [get]
func (a *DataPermission) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.DataPermissionQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.DataPermissionBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags DataPermissionAPI
// @Security ApiKeyAuth
// @Summary Get data permission record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.DataPermission}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/resource/data-permissions/{id} [get]
func (a *DataPermission) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.DataPermissionBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags DataPermissionAPI
// @Security ApiKeyAuth
// @Summary Create data permission record
// @Param body body schema.DataPermissionForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.DataPermission}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/resource/data-permissions [post]
func (a *DataPermission) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.DataPermissionForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.DataPermissionBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags DataPermissionAPI
// @Security ApiKeyAuth
// @Summary Update data permission record by ID
// @Param id path string true "unique id"
// @Param body body schema.DataPermissionForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/resource/data-permissions/{id} [put]
func (a *DataPermission) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.DataPermissionForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.DataPermissionBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags DataPermissionAPI
// @Security ApiKeyAuth
// @Summary Delete data permission record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/resource/data-permissions/{id} [delete]
func (a *DataPermission) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.DataPermissionBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
