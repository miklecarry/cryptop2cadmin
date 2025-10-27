package routers

import (
	"hostmanager/controllers"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/", &controllers.HostController{})
	web.Router("/users", &controllers.UserController{}, "get:Get;post:Post")
	web.Router("/user/:id/delete", &controllers.UserController{}, "get:Delete")

	// Хосты
	web.Router("/host/:id", &controllers.HostDetailController{})
	web.Router("/host/:id/update", &controllers.HostDetailController{}, "post:Update")
	web.Router("/host/:id/delete", &controllers.HostDetailController{}, "get:Delete")
	web.Router("/host/create", &controllers.HostCreateController{}, "get:Get;post:Create")

	// Аутентификация
	web.Router("/login", &controllers.AuthController{})
	web.Router("/login/telegram", &controllers.AuthController{}, "get:TelegramLogin")
	web.Router("/logout", &controllers.AuthController{}, "get:Logout")
	// API
	web.Router("/api/host/:id/payment-methods", &controllers.APIHostController{}, "get:GetPaymentMethods")
	web.Router("/api/host/:id/start-monitoring", &controllers.APIHostController{}, "post:StartMonitoring")
	web.Router("/api/host/:id/stop-monitoring", &controllers.APIHostController{}, "post:StopMonitoring")
	web.Router("/api/host/state", &controllers.APIHostController{})

}
