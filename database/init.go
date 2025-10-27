package database

import (
	"hostmanager/models"
	"log"
	"os"
	"sync"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var once sync.Once

func InitDB() {
	once.Do(func() {
		if _, err := os.Stat("data"); os.IsNotExist(err) {
			err := os.Mkdir("data", 0755)
			if err != nil {
				log.Fatalf("Не удалось создать папку для базы: %v", err)
			}
		}

		err := orm.RegisterDriver("sqlite3", orm.DRSqlite)
		if err != nil {
			log.Printf("Драйвер sqlite3 уже зарегистрирован: %v", err)
		}

		err = orm.RegisterDataBase("default", "sqlite3", "data/data.db")
		if err != nil {
			log.Fatalf("Ошибка регистрации базы данных: %v", err)
		}

		// ✅ Регистрируем модели один раз
		orm.RegisterModel(
			new(models.User),
			new(models.Host),
			new(models.HostLog),
		)

		if err := orm.RunSyncdb("default", false, true); err != nil {
			log.Fatalf("Ошибка при создании таблиц: %v", err)
		}

		o := orm.NewOrm()
		count, err := o.QueryTable(new(models.User)).Filter("Role", "superadmin").Count()
		if err != nil {
			log.Fatalf("Ошибка при проверке суперюзера: %v", err)
		}

		if count == 0 {
			user := models.User{
				FullName: "Super Admin",
				Username: "superadmin",
				Password: models.HashPassword("superpassword"),
				Role:     "superadmin",
			}
			if _, err := o.Insert(&user); err != nil {
				log.Fatalf("Не удалось создать суперюзера: %v", err)
			}
			log.Println("✅ Создан суперюзер: superadmin / superpassword")
		}

		log.Println("✅ База данных успешно инициализирована")
	})
}
