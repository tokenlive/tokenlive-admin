package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// UserAPIKey 用户 API Key 的 HTTP API 控制器
type UserAPIKey struct {
	UserAPIKeyBIZ *biz.UserAPIKey
}

// @Tags UserAPIKeyAPI
// @Security ApiKeyAuth
// @Summary 查询 API Key 分页列表
// @Param current query int true "页码" default(1)
// @Param pageSize query int true "每页条数" default(10)
// @Param user_id query string false "关联用户ID"
// @Param name query string false "友好名称"
// @Param status query int false "状态: 1-启用, 2-禁用"
// @Success 200 {object} util.ResponseResult{data=[]schema.UserAPIKey}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/user-api-keys [get]
func (a *UserAPIKey) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.UserAPIKeyQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.UserAPIKeyBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags UserAPIKeyAPI
// @Security ApiKeyAuth
// @Summary 获取 API Key 明文（用于复制操作）
// @Description 安全接口，返回未脱敏的 API Key 明文，仅用于前端复制功能
// @Param id path string true "API Key 记录 ID"
// @Success 200 {object} util.ResponseResult{data=string}
// @Failure 401 {object} util.ResponseResult
// @Failure 404 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/user-api-keys/{id}/plaintext [get]
func (a *UserAPIKey) GetPlaintext(c *gin.Context) {
	ctx := c.Request.Context()
	plaintext, err := a.UserAPIKeyBIZ.GetPlaintext(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, plaintext)
}

// @Tags UserAPIKeyAPI
// @Security ApiKeyAuth
// @Summary 根据 ID 查询单条记录
// @Param id path string true "API Key 记录 ID"
// @Success 200 {object} util.ResponseResult{data=schema.UserAPIKey}
// @Failure 401 {object} util.ResponseResult
// @Failure 404 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/user-api-keys/{id} [get]
func (a *UserAPIKey) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.UserAPIKeyBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags UserAPIKeyAPI
// @Security ApiKeyAuth
// @Summary 创建 API Key
// @Param body body schema.UserAPIKeyForm true "请求 Body"
// @Success 200 {object} util.ResponseResult{data=schema.UserAPIKey}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/user-api-keys [post]
func (a *UserAPIKey) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.UserAPIKeyForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.UserAPIKeyBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	// 在此处，返回的结构体中含有刚生成的明文密钥，仅展示一次
	util.ResSuccess(c, result)
}

// @Tags UserAPIKeyAPI
// @Security ApiKeyAuth
// @Summary 根据 ID 修改记录
// @Param id path string true "API Key 记录 ID"
// @Param body body schema.UserAPIKeyForm true "请求 Body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 404 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/user-api-keys/{id} [put]
func (a *UserAPIKey) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.UserAPIKeyForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.UserAPIKeyBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags UserAPIKeyAPI
// @Security ApiKeyAuth
// @Summary 根据 ID 删除记录
// @Param id path string true "API Key 记录 ID"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 404 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/user-api-keys/{id} [delete]
func (a *UserAPIKey) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.UserAPIKeyBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
