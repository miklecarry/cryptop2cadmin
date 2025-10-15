package models

import (
	"time"

	"github.com/beego/beego/v2/client/orm"
)

type HostLog struct {
	Id        int64     `orm:"auto"`
	Host      *Host     `orm:"rel(fk)"`
	Level     string    `orm:"size(20)"` // "err", "info", "bounty"
	Message   string    `orm:"type(text)"`
	CreatedAt time.Time `orm:"auto_now_add;type(datetime)"`
}

func init() {
	orm.RegisterModel(new(HostLog))
}
