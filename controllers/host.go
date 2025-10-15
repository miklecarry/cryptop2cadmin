package controllers

import (
	"hostmanager/models"
	"hostmanager/services"

	"github.com/beego/beego/v2/client/orm"
)

type HostController struct {
	BaseController
}

func (c *HostController) Get() {
	o := orm.NewOrm()
	var hosts []models.Host
	_, err := o.QueryTable(new(models.Host)).All(&hosts)
	if err != nil {
		hosts = []models.Host{}
	}

	hostsWithState := make([]map[string]interface{}, len(hosts))
	for i, h := range hosts {
		state := services.GetHostState(h.Name) // ← по Name!
		hostsWithState[i] = map[string]interface{}{
			"Host":    h,
			"Online":  state.Online,
			"Enabled": state.Enabled,
		}
	}

	c.Data["Hosts"] = hostsWithState
	c.Data["Role"] = c.GetSession("role")
	c.Layout = "layout.tpl"
	c.TplName = "hosts.tpl"
}
