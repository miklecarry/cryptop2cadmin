// controllers/host_create.go
package controllers

import (
	"hostmanager/models"

	"github.com/beego/beego/v2/client/orm"
)

type HostCreateController struct {
	BaseController
}

func (c *HostCreateController) Get() {
	o := orm.NewOrm()
	var users []models.User
	o.QueryTable("user").All(&users)

	c.Data["Users"] = users
	c.Data["Role"] = c.GetSession("role")
	c.Layout = "layout.tpl"
	c.TplName = "host_create.tpl"
}

func (c *HostCreateController) Create() {
	name := c.GetString("name")
	userID, _ := c.GetInt64("user_id")

	if name == "" || userID <= 0 {
		c.Data["Error"] = "Имя хоста и пользователь обязательны"
		c.Get()
		return
	}

	o := orm.NewOrm()

	if o.QueryTable("host").Filter("Name", name).Exist() {
		c.Data["Error"] = "Хост с таким именем уже существует"
		c.Get()
		return
	}

	// Проверка, что пользователь существует
	var user models.User
	if o.QueryTable("user").Filter("Id", userID).One(&user) != nil {
		c.Data["Error"] = "Пользователь не найден"
		c.Get()
		return
	}

	// Проверка, что у пользователя ещё нет хоста (1:1)
	if o.QueryTable("host").Filter("User__Id", userID).Exist() {
		c.Data["Error"] = "У этого пользователя уже есть хост"
		c.Get()
		return
	}

	host := &models.Host{
		Name:        name,
		User:        &user,
		ServerAddr:  c.GetString("server_addr"),
		Active:      c.GetString("active") == "on",
		SocketURL:   c.GetString("socket_url"),
		AccessToken: c.GetString("access_token"),

		Priority: c.GetString("priority") == "on",
	}

	if min, err := c.GetInt("min_limit"); err == nil {
		host.MinLimit = min
	}
	if max, err := c.GetInt("max_limit"); err == nil {
		host.MaxLimit = max
	}
	if timeout, err := c.GetInt("timeout"); err == nil {
		host.Timeout = timeout
	}

	if _, err := o.Insert(host); err != nil {
		c.Data["Error"] = "Ошибка создания: " + err.Error()
		c.Get()
		return
	}

	c.Redirect("/", 302)
}
