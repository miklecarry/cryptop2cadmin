package main

import (
	"hostmanager/database"
	_ "hostmanager/routers"
	"hostmanager/services"

	"github.com/beego/beego/v2/server/web"
)

func main() {
	database.InitDB()
	services.StartStateCleanup() // ← добавить
	web.Run()
}
