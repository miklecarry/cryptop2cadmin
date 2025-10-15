package controllers

import (
	"hostmanager/models"
	"hostmanager/services"

	"github.com/beego/beego/v2/client/orm"
)

type HostDetailController struct {
	BaseController
}

func (c *HostDetailController) Get() {
	id := c.Ctx.Input.Param(":id")
	o := orm.NewOrm()
	var host models.Host
	err := o.QueryTable("host").Filter("Id", id).One(&host)
	if err != nil {
		c.Abort("404")
		return
	}

	// Получаем логи
	var logs []models.HostLog
	o.QueryTable("host_log").Filter("Host__Id", host.Id).OrderBy("-Id").Limit(100).All(&logs)

	c.Data["Host"] = host
	c.Data["Logs"] = logs
	c.Layout = "layout.tpl"
	c.TplName = "host_detail.tpl"
}

func (c *HostDetailController) Update() {
	id := c.Ctx.Input.Param(":id")

	priority := c.GetString("priority") == "on"
	timeout, _ := c.GetInt("timeout")

	o := orm.NewOrm()
	var host models.Host
	err := o.QueryTable("host").Filter("Id", id).One(&host)
	if err != nil {
		c.Abort("404")
		return
	}

	host.Priority = priority
	host.Timeout = timeout

	o.Update(&host)

	c.Redirect("/host/"+id, 302)
}

func (c *HostDetailController) Delete() {
	id := c.Ctx.Input.Param(":id")
	o := orm.NewOrm()

	var host models.Host
	err := o.QueryTable("host").Filter("Id", id).One(&host)
	if err != nil {
		c.Abort("404")
		return
	}

	o.QueryTable("host_log").Filter("Host__Id", host.Id).Delete()
	o.QueryTable("host").Filter("Id", id).Delete()

	services.DeleteHostState(host.Name)

	c.Redirect("/", 302)
}
