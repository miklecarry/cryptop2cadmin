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
	role := c.GetSession("role")
	if role == nil {
		c.Redirect("/login", 302)
		return
	}

	o := orm.NewOrm()
	var host models.Host
	err := o.QueryTable("host").Filter("Id", id).One(&host)
	if err != nil {
		c.Abort("404")
		return
	}

	// Загружаем связанного пользователя
	o.LoadRelated(&host, "User")

	// Для обычного пользователя — только свой хост
	if role == "user" {
		currentUserID := c.GetSession("user_id")
		if currentUserID == nil {
			c.Abort("403")
			return
		}

		// Приведение типа: сессия может хранить int64 или int
		var currentID int
		switch v := currentUserID.(type) {
		case int64:
			currentID = int(v)
		case int:
			currentID = v
		default:
			c.Abort("403")
			return
		}

		if host.User == nil || host.User.Id != currentID {
			c.Abort("403")
			return
		}
	}

	// Логи
	var logs []models.HostLog
	o.QueryTable("host_log").Filter("Host__Id", host.Id).OrderBy("-Id").Limit(100).All(&logs)

	c.Data["Host"] = host
	c.Data["Role"] = role
	c.Data["Logs"] = logs

	// Только админы видят список пользователей
	if role == "admin" || role == "superadmin" {
		var users []models.User
		o.QueryTable("user").All(&users)
		c.Data["Users"] = users
	}

	c.Layout = "layout.tpl"
	c.TplName = "host_detail.tpl"
}

func (c *HostDetailController) Update() {
	id := c.Ctx.Input.Param(":id")
	role := c.GetSession("role")
	if role == nil {
		c.Abort("403")
		return
	}

	o := orm.NewOrm()
	var host models.Host
	err := o.QueryTable("host").Filter("Id", id).One(&host)
	if err != nil {
		c.Abort("404")
		return
	}

	// Для обычного пользователя — только свой хост и только разрешённые поля
	if role == "user" {
		currentUserID := c.GetSession("user_id")
		if currentUserID == nil {
			c.Abort("403")
			return
		}

		var currentID int
		switch v := currentUserID.(type) {
		case int64:
			currentID = int(v)
		case int:
			currentID = v
		default:
			c.Abort("403")
			return
		}

		o.LoadRelated(&host, "User")
		if host.User == nil || host.User.Id != currentID {
			c.Abort("403")
			return
		}

		// Обновляем ТОЛЬКО разрешённые поля

		host.AccessToken = c.GetString("access_token")
		if min, err := c.GetInt("min_limit"); err == nil {
			host.MinLimit = min
		}
		if max, err := c.GetInt("max_limit"); err == nil {
			host.MaxLimit = max
		}

		o.Update(&host)
		c.Redirect("/host/"+id, 302)
		return
	}

	// === Админы: полный доступ ===
	newUserID, _ := c.GetInt("user_id")
	if newUserID > 0 && (host.User == nil || host.User.Id != newUserID) {
		var newUser models.User
		if err := o.QueryTable("user").Filter("Id", newUserID).One(&newUser); err != nil {
			c.Data["Error"] = "Пользователь не найден"
			c.Get()
			return
		}

		count, _ := o.QueryTable("host").Filter("User__Id", newUserID).Count()
		if count > 0 {
			c.Data["Error"] = "У выбранного пользователя уже есть хост"
			c.Get()
			return
		}

		host.User = &newUser
	}

	// Все поля доступны админам

	host.ServerAddr = c.GetString("server_addr")
	host.SocketURL = c.GetString("socket_url")
	host.AccessToken = c.GetString("access_token")

	host.Priority = c.GetString("priority") == "on"

	if min, err := c.GetInt("min_limit"); err == nil {
		host.MinLimit = min
	}
	if max, err := c.GetInt("max_limit"); err == nil {
		host.MaxLimit = max
	}
	if timeout, err := c.GetInt("timeout"); err == nil {
		host.Timeout = timeout
	}

	if _, err := o.Update(&host); err != nil {
		c.Data["Error"] = "Ошибка обновления: " + err.Error()
		c.Get()
		return
	}

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
