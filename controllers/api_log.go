package controllers

import (
	"encoding/json"
	"hostmanager/models"
	"hostmanager/services"
	"hostmanager/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type APILogController struct {
	web.Controller
}

type LogRequest struct {
	Name    string `json:"name"`
	Level   string `json:"level"` // "info", "err", "bounty"
	Message string `json:"message"`
}

func (c *APILogController) Push() {
	c.Ctx.Input.CopyBody(1 << 20)
	rawBody := c.Ctx.Input.RequestBody
	rawBody = utils.RemoveBOM(rawBody)
	if len(rawBody) == 0 {
		c.Data["json"] = map[string]interface{}{"status": "empty_body", "timeout": 0}
		c.ServeJSON()
		return
	}

	var req LogRequest
	if err := json.Unmarshal(rawBody, &req); err != nil || req.Name == "" {
		c.Data["json"] = map[string]interface{}{"status": "error", "timeout": 0}
		c.ServeJSON()
		return
	}

	if !map[string]bool{"info": true, "err": true, "bounty": true}[req.Level] {
		c.Data["json"] = map[string]interface{}{"status": "error", "timeout": 0}
		c.ServeJSON()
		return
	}

	o := orm.NewOrm()
	host := models.Host{Name: req.Name}
	if err := o.Read(&host, "Name"); err != nil {
		c.Data["json"] = map[string]interface{}{"status": "error", "timeout": 0}
		c.ServeJSON()
		return
	}

	// Обновляем состояние
	currentState := services.GetHostState(req.Name)
	services.UpdateHostState(req.Name, currentState.Enabled)

	// Сохраняем лог
	log := models.HostLog{Host: &host, Level: req.Level, Message: req.Message}
	if _, err := o.Insert(&log); err != nil {
		c.Data["json"] = map[string]interface{}{"status": "error", "timeout": 0}
		c.ServeJSON()
		return
	}

	cleanupOldLogs(o, host.Id)

	timeout := 0

	if req.Level == "bounty" {

		if !host.Priority {

			if services.HasActivePriorityHost() {
				timeout = host.Timeout
			}
		}
	}

	c.Data["json"] = map[string]interface{}{
		"status":  "ok",
		"timeout": timeout,
	}
	c.ServeJSON()
}

// Безопасная очистка для SQLite
func cleanupOldLogs(o orm.Ormer, hostId int64) {
	var logs []models.HostLog
	_, err := o.QueryTable("host_log").
		Filter("Host__Id", hostId).
		OrderBy("-Id").
		Limit(200). // берём чуть больше, чтобы не грузить всё
		All(&logs)
	if err != nil || len(logs) <= 100 {
		return
	}

	// Удаляем всё, кроме первых 100 (самых свежих)
	idsToDelete := make([]int64, len(logs)-100)
	for i := 100; i < len(logs); i++ {
		idsToDelete[i-100] = logs[i].Id
	}
	if len(idsToDelete) > 0 {
		o.QueryTable("host_log").Filter("id__in", idsToDelete).Delete()
	}
}
