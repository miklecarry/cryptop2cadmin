package controllers

import (
	"hostmanager/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type AuthController struct {
	web.Controller
}

func (c *AuthController) Get() {

	c.TplName = "login.tpl"
}

func (c *AuthController) Post() {
	username := c.GetString("username")
	password := c.GetString("password")

	if username == "" || password == "" {
		c.Data["Error"] = "Все поля обязательны"

		c.TplName = "login.tpl"
		return
	}

	o := orm.NewOrm()
	user := models.User{Username: username}
	err := o.Read(&user, "Username")

	if err != nil || user.Password != models.HashPassword(password) {
		c.Data["Error"] = "Неверный логин или пароль"
		c.TplName = "login.tpl"
		return
	}

	// сохраняем пользователя в сессии
	c.SetSession("user_id", user.Id)
	c.SetSession("role", user.Role)

	c.Redirect("/", 302)
}

func (c *AuthController) Logout() {
	c.DestroySession()
	c.Redirect("/login", 302)
}

func (c *AuthController) TelegramLogin() {
	token := c.GetString("token")
	if token == "" {
		c.Redirect("/login", 302)
		return
	}

	o := orm.NewOrm()
	var user models.User
	err := o.QueryTable("user").Filter("WebAppToken", token).One(&user)
	if err != nil {
		c.Data["Error"] = "Ссылка недействительна"
		c.TplName = "login.tpl"
		return
	}

	// Авторизуем
	c.SetSession("user_id", user.Id)
	c.SetSession("role", user.Role)
	c.SetSession("username", user.Username)

	// Опционально: обнулить токен (если одноразовый)
	// user.WebAppToken = ""
	// o.Update(&user, "WebAppToken")

	c.Redirect("/", 302)
}
