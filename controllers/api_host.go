package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"hostmanager/models"

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

type remoteConfigResponse struct {
	Active      bool      `json:"active"`
	SocketURL   string    `json:"socket"`
	AccessToken string    `json:"access_token"`
	MinAmount   int       `json:"min"`
	MaxAmount   int       `json:"max"`
	StopTime    time.Time `json:"stop_time"`
	IPAddr      string    `json:"ip_addr"`
	Logger      string    `json:"logger"`
}

func (c *APIHostController) Get() {
	username, password, ok := c.Ctx.Request.BasicAuth()
	if !ok {
		c.Ctx.Output.SetStatus(http.StatusUnauthorized)
		c.Data["json"] = map[string]interface{}{"status": "unauthorized"}
		c.ServeJSON()
		return
	}

	o := orm.NewOrm()
	// Найдём пользователя по username
	user := models.User{Username: username}
	err := o.Read(&user, "Username")
	if err == orm.ErrNoRows {
		c.Ctx.Output.SetStatus(http.StatusUnauthorized)
		c.Data["json"] = map[string]interface{}{"status": "unauthorized"}
		c.ServeJSON()
		return
	} else if err != nil {
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]interface{}{"status": "error", "error": err.Error()}
		c.ServeJSON()
		return
	}

	if user.Password != models.HashPassword(password) {
		c.Ctx.Output.SetStatus(http.StatusUnauthorized)
		c.Data["json"] = map[string]interface{}{"status": "unauthorized"}
		c.ServeJSON()
		return
	}

	var host models.Host
	qs := o.QueryTable(new(models.Host))
	err = qs.Filter("User__Id", user.Id).One(&host)
	if err == orm.ErrNoRows {
		c.Ctx.Output.SetStatus(http.StatusNotFound)
		c.Data["json"] = map[string]interface{}{"status": "no_host", "timeout": 0}
		c.ServeJSON()
		return
	} else if err != nil {
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]interface{}{"status": "error", "error": err.Error()}
		c.ServeJSON()
		return
	}

	resp := remoteConfigResponse{
		Active:      host.Active,
		SocketURL:   host.SocketURL,
		AccessToken: host.AccessToken,
		MinAmount:   host.MinLimit,
		MaxAmount:   host.MaxLimit,
		StopTime:    host.StopTime,
		IPAddr:      host.ServerAddr,
	}

	c.Data["json"] = resp
	c.ServeJSON()
}
func (c *APIHostController) GetPaymentMethods() {
	id := c.Ctx.Input.Param(":id")

	o := orm.NewOrm()

	var host models.Host
	err := o.QueryTable("host").Filter("Id", id).One(&host)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"status": "no host with id"}
		c.ServeJSON()
		return
	}

	// Запрос к app.cr.bot
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "https://app.cr.bot/internal/v1/p2c/accounts", nil)
	req.Header.Set("Cookie", "access_token="+host.AccessToken)

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		fmt.Print(resp.StatusCode)
		c.Data["json"] = map[string]interface{}{"error": "Не удалось получить методы оплаты"}
		c.ServeJSON()
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	c.Data["json"] = result
	c.ServeJSON()
}

func (c *APIHostController) SelectPaymentMethod() {
	id := c.Ctx.Input.Param(":id")
	methodID := c.GetString("method_id")

	o := orm.NewOrm()
	var host models.Host
	err := o.QueryTable("host").Filter("Id", id).One(&host)
	if err != nil {
		c.Abort("404")
		return
	}

	host.PaymentMethodID = methodID
	host.WorkerRunning = true
	o.Update(&host, "PaymentMethodID", "WorkerRunning")

	c.Data["json"] = map[string]string{"status": "ok"}
	c.ServeJSON()

	// Лог
	log := models.HostLog{
		Host:    &host,
		Level:   "info",
		Message: fmt.Sprintf("Бот стартанул с id метода %s", methodID),
	}
	o.Insert(&log)
}
func (c *APIHostController) StartMonitoring() {
	id := c.Ctx.Input.Param(":id")
	methodID := c.GetString("method_id")

	if methodID == "" {
		c.Data["json"] = map[string]string{"error": "method_id обязателен"}
		c.ServeJSON()
		return
	}

	o := orm.NewOrm()
	var host models.Host
	err := o.QueryTable("host").Filter("Id", id).One(&host)
	if err != nil {
		c.Abort("404")
		return
	}

	// Проверяем, что AccessToken задан
	if host.AccessToken == "" {
		c.Data["json"] = map[string]string{"error": "Access Token не задан"}
		c.ServeJSON()
		return
	}

	// Пробуем получить методы (валидация токена)
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "https://app.cr.bot/internal/v1/p2c/accounts", nil)
	req.Header.Set("Cookie", "access_token="+host.AccessToken)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		c.Data["json"] = map[string]string{"error": "Неверный Access Token"}
		c.ServeJSON()
		return
	}
	resp.Body.Close()

	// Сохраняем настройки - воркер запустится автоматически через глобальный мониторинг
	host.Active = true
	host.PaymentMethodID = methodID
	host.WorkerRunning = true
	o.Update(&host, "Active", "PaymentMethodID", "WorkerRunning")

	// Лог
	log := models.HostLog{
		Host:    &host,
		Level:   "info",
		Message: fmt.Sprintf("Мониторинг запущен с id метода %s", methodID),
	}
	o.Insert(&log)

	c.Data["json"] = map[string]string{"status": "ok"}
	c.ServeJSON()
}

func (c *APIHostController) StopMonitoring() {
	id := c.Ctx.Input.Param(":id")
	o := orm.NewOrm()
	var host models.Host
	o.QueryTable("host").Filter("Id", id).One(&host)

	// Только меняем флаги - глобальный мониторинг сам остановит воркер
	host.Active = false
	host.WorkerRunning = false
	o.Update(&host, "Active", "WorkerRunning")

	c.Data["json"] = map[string]string{"status": "ok"}
	c.ServeJSON()
}
