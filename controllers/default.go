package controllers

import "github.com/beego/beego/v2/server/web"

type BaseController struct {
	web.Controller
}

func (c *BaseController) Prepare() {
	c.Data["CurrentPath"] = c.Ctx.Request.URL.Path

	uid := c.GetSession("user_id")

	if uid == nil {
		c.Redirect("/login", 302)
		c.StopRun()
		return
	}
}
