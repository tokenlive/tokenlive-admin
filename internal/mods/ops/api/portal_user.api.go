package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

type PortalUserAPI struct {
	PortalUserBIZ *biz.PortalUser
}

// @Tags PortalUserAPI
// @Security ApiKeyAuth
// @Summary Search portal users
// @Param keyword query string false "search keyword"
// @Param limit query int false "limit" default(20)
// @Success 200 {object} util.ResponseResult{data=[]biz.PortalUserResult}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/ops/portal/users [get]
func (a *PortalUserAPI) Query(c *gin.Context) {
	ctx := c.Request.Context()
	keyword := c.Query("keyword")
	limitStr := c.Query("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	result, err := a.PortalUserBIZ.Search(ctx, keyword, limit)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}
