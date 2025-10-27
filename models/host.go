package models

import (
	"time"

	"github.com/beego/beego/v2/client/orm"
)

type Host struct {
	Id         int64  `orm:"auto"`
	Name       string `orm:"size(100);unique"`
	ServerAddr string `orm:"size(100);null"`

	User *User `orm:"rel(one);unique;column(user_id);null"`

	MinLimit int  `orm:"default(0)"`
	MaxLimit int  `orm:"default(0)"`
	Priority bool `orm:"default(false)"`
	Timeout  int  `orm:"default(0)"`

	Active          bool      `orm:"default(false)"`
	SocketURL       string    `orm:"size(255);null"`
	AccessToken     string    `orm:"size(255);null"`
	StopTime        time.Time `orm:"type(datetime);null"`
	PaymentMethodID string    `orm:"size(100);null"` // ID выбранного метода оплаты
	WorkerRunning   bool      `orm:"default(false)"` // флаг: запущен ли воркер
	CreatedAt       time.Time `orm:"auto_now_add;type(datetime)"`
	UpdatedAt       time.Time `orm:"auto_now;type(datetime)"`
}

func init() {
	orm.RegisterModel(new(Host))
}
