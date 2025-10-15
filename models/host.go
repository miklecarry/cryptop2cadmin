package models

import (
	"time"

	"github.com/beego/beego/v2/client/orm"
)

type Host struct {
	Id        int64     `orm:"auto"`
	Name      string    `orm:"size(100);unique"` // ← UNIQUE!
	Ip        string    `orm:"size(50)"`         // может повторяться
	MinLimit  int       `orm:"default(0)"`
	MaxLimit  int       `orm:"default(0)"`
	Priority  bool      `orm:"default(false)"`
	Timeout   int       `orm:"default(0)"`
	CreatedAt time.Time `orm:"auto_now_add;type(datetime)"`
	UpdatedAt time.Time `orm:"auto_now;type(datetime)"`
}
type HostState struct {
	Online  bool // "в сети"
	Enabled bool // "включен"
}

func init() {
	orm.RegisterModel(new(Host))
}
