package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func (a *Dashboard) SyncRedis(c *gin.Context) {
	if !util.FromIsRootUser(c.Request.Context()) {
		util.ResError(c, fmt.Errorf("only admin is allowed to perform sync operation"))
		return
	}

	if a.RedisSync == nil {
		util.ResError(c, fmt.Errorf("redis sync service is not initialized"))
		return
	}

	err := a.RedisSync.SyncAllToRedis(c.Request.Context())
	if err != nil {
		util.ResError(c, err)
		return
	}

	util.ResSuccess(c, nil)
}
