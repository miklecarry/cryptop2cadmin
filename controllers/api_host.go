package controllers

import (
	"encoding/json"
	"hostmanager/models"
	"hostmanager/services"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type APIHostController struct {
	web.Controller
}

type HostStateRequest struct {
	Name     string `json:"name"`
	Ip       string `json:"ip"`
	Enabled  bool   `json:"enabled"`
	MinLimit int    `json:"min_limit,omitempty"`
	MaxLimit int    `json:"max_limit,omitempty"`
}

func (c *APIHostController) Post() {
	c.Ctx.Input.CopyBody(1 << 20)
	//rawBody := c.Ctx.Input.RequestBody
	//rawBody = utils.RemoveBOM(rawBody) // ← удаляем BOM

	if len(c.Ctx.Input.RequestBody) == 0 {
		c.Data["json"] = map[string]interface{}{"status": "empty_body", "timeout": 0}
		c.ServeJSON()
		return
	}
	var hreq HostStateRequest
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &hreq); err != nil || hreq.Name == "" {
		c.Data["json"] = map[string]interface{}{"status": err.Error(), "timeout": 0}
		c.ServeJSON()
		return
	}

	// Обновляем состояние в памяти по NAME
	services.UpdateHostState(hreq.Name, hreq.Enabled)

	o := orm.NewOrm()
	host := models.Host{Name: hreq.Name}
	err := o.Read(&host, "Name") // ← читаем по Name

	if err == orm.ErrNoRows {
		host.Name = hreq.Name
		host.Ip = hreq.Ip
		host.MinLimit = hreq.MinLimit
		host.MaxLimit = hreq.MaxLimit
		_, err = o.Insert(&host)
		if err != nil {
			c.Data["json"] = map[string]interface{}{"status": "error_insert", "timeout": 0}
			c.ServeJSON()
			return
		}
	} else {
		// Обновляем IP и лимиты (IP может меняться!)
		updated := false
		if hreq.Ip != "" && hreq.Ip != host.Ip {
			host.Ip = hreq.Ip
			updated = true
		}
		if hreq.MinLimit != host.MinLimit {
			host.MinLimit = hreq.MinLimit
			updated = true
		}
		if hreq.MaxLimit != host.MaxLimit {
			host.MaxLimit = hreq.MaxLimit
			updated = true
		}
		if updated {
			o.Update(&host, "Ip", "MinLimit", "MaxLimit")
		}
	}

	c.Data["json"] = map[string]interface{}{"status": "ok", "timeout": 0}
	c.ServeJSON()
}
