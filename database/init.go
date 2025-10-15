package database

import (
	"hostmanager/models"
	"log"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB() {
	orm.RegisterDriver("sqlite3", orm.DRSqlite)
	orm.RegisterDataBase("default", "sqlite3", "data.db")
	orm.RunSyncdb("default", false, true)

	o := orm.NewOrm()

	count, _ := o.QueryTable(new(models.User)).Filter("Role", "superadmin").Count()
	if count == 0 {
		user := models.User{
			FullName: "Super Admin",
			Username: "superadmin",
			Password: models.HashPassword("superpassword"), // пароль задаём зашитым
			Role:     "superadmin",
		}
		_, err := o.Insert(&user)
		if err != nil {
			log.Fatal("Не удалось создать суперюзера: ", err)
		}
		log.Println("Создан суперюзер: superadmin / superpassword")
	}
}
