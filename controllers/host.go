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

	// Загружаем User для каждого хоста
	for i := range hosts {
		o.LoadRelated(&hosts[i], "User")
	}

	hostsWithState := make([]map[string]interface{}, len(hosts))
	for i, h := range hosts {
		state := services.GetHostState(h.Name)

		// Извлекаем UserID (если есть)
		var userID int
		if h.User != nil {
			userID = h.User.Id
		}

		hostsWithState[i] = map[string]interface{}{
			"Host":    h,
			"Online":  state.Online,
			"Enabled": state.Enabled,
			"UserID":  userID, // ← добавили!
		}
	}

	c.Data["Hosts"] = hostsWithState
	c.Data["Role"] = c.GetSession("role")

	// Текущий пользователь
	currentUserID := int(0)
	if uid := c.GetSession("user_id"); uid != nil {
		if id, ok := uid.(int); ok {
			currentUserID = id
		}
	}
	c.Data["CurrentUserID"] = currentUserID

	c.Layout = "layout.tpl"
	c.TplName = "hosts.tpl"
}
