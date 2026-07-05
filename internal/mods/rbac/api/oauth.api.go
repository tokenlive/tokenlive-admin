package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

type OAuth struct {
	OAuthBIZ *biz.OAuth
}

func (a *OAuth) Providers(c *gin.Context) {
	ctx := c.Request.Context()
	util.ResSuccess(c, a.OAuthBIZ.GetEnabledProviders(ctx))
}

func (a *OAuth) Login(c *gin.Context) {
	ctx := c.Request.Context()
	loginURL, err := a.OAuthBIZ.BuildLoginURL(ctx, c.Param("provider"), c.Query("redirect"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	c.Redirect(http.StatusFound, loginURL)
}

func (a *OAuth) Callback(c *gin.Context) {
	ctx := c.Request.Context()
	redirectURL, err := a.OAuthBIZ.HandleCallback(ctx, c.Param("provider"), c.Query("code"), c.Query("state"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func (a *OAuth) Exchange(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.OAuthExchangeForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	}
	token, err := a.OAuthBIZ.ExchangeTicket(ctx, item.Ticket)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, token)
}
