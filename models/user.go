package models

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/beego/beego/v2/client/orm"
)

type User struct {
	Id       int    `orm:"auto"`
	FullName string `orm:"size(100)"`
	Username string `orm:"unique"`
	Password string `orm:"size(100)"`
	Role     string `orm:"size(20)"` // "superadmin", "admin", "user"
}

func init() {
	orm.RegisterModel(new(User))
}
func HashPassword(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}
