package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Provider management
type Provider struct {
	ProviderBIZ *biz.Provider
}

// @Tags ProviderAPI
// @Security ApiKeyAuth
// @Summary Query provider list
// @Param current query int true "pagination index" default(1)
// @Param pageSize query int true "pagination size" default(10)
// @Param code query string false "Code of provider"
// @Param name query string false "Name of provider"
// @Success 200 {object} util.ResponseResult{data=[]schema.Provider}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/providers [get]
func (p *Provider) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.ProviderQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := p.ProviderBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags ProviderAPI
// @Security ApiKeyAuth
// @Summary Get provider record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.Provider}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/providers/{id} [get]
func (p *Provider) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := p.ProviderBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags ProviderAPI
// @Security ApiKeyAuth
// @Summary Create provider record
// @Param body body schema.ProviderForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.Provider}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/providers [post]
func (p *Provider) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ProviderForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := p.ProviderBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags ProviderAPI
// @Security ApiKeyAuth
// @Summary Update provider record by ID
// @Param id path string true "unique id"
// @Param body body schema.ProviderForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/providers/{id} [put]
func (p *Provider) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ProviderForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := p.ProviderBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ProviderAPI
// @Security ApiKeyAuth
// @Summary Delete provider record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/providers/{id} [delete]
func (p *Provider) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := p.ProviderBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ProviderAPI
// @Security ApiKeyAuth
// @Summary Fetch models from upstream provider
// @Param id path string true "provider id"
// @Param body body schema.FetchModelsForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.FetchModelsResult}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/providers/{id}/fetch-models [post]
func (p *Provider) FetchModels(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.FetchModelsForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := p.ProviderBIZ.FetchModels(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}
