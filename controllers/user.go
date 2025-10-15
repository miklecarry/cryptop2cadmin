package controllers

import (
	"strconv"

	"hostmanager/models"

	"github.com/beego/beego/v2/client/orm"
)

type UserController struct {
	BaseController
}

func (c *UserController) Prepare() {
	c.BaseController.Prepare()
	role := c.GetSession("role")
	if role != "superadmin" {
		c.Ctx.WriteString("Доступ запрещён")
		c.StopRun()
	}
}

func (c *UserController) Get() {
	o := orm.NewOrm()
	var users []models.User
	o.QueryTable(new(models.User)).All(&users)

	c.Data["Users"] = users
	c.Layout = "layout.tpl"
	c.TplName = "users.tpl"
}

func (c *UserController) Post() {
	fullname := c.GetString("fullname")
	username := c.GetString("username")
	password := c.GetString("password")
	role := c.GetString("role")

	if fullname == "" || username == "" || password == "" || role == "" {
		c.Data["Error"] = "Все поля обязательны"
		c.Get() // повторно загружаем данные
		return
	}

	// Проверка уникальности username
	o := orm.NewOrm()
	existing := models.User{Username: username}
	if o.Read(&existing, "Username") == nil {
		c.Data["Error"] = "Пользователь с таким логином уже существует"
		c.Get()
		return
	}

	user := models.User{
		FullName: fullname,
		Username: username,
		Password: models.HashPassword(password),
		Role:     role,
	}
	_, err := o.Insert(&user)
	if err != nil {
		c.Data["Error"] = "Ошибка создания пользователя"
		c.Get()
		return
	}

	c.Redirect("/users", 302)
}

// Новый метод: удаление
func (c *UserController) Delete() {
	idStr := c.Ctx.Input.Param(":id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Abort("400")
		return
	}

	// Защита: нельзя удалить самого себя
	currentUserID := c.GetSession("user_id")
	if currentUserID != nil {
		if id == currentUserID.(int64) {
			c.Data["Error"] = "Нельзя удалить самого себя"
			c.Redirect("/users", 302)
			return
		}
	}

	o := orm.NewOrm()
	user := models.User{Id: int(id)}
	if o.Read(&user) == nil {
		// Удаляем
		o.Delete(&user)
	}

	c.Redirect("/users", 302)
}
