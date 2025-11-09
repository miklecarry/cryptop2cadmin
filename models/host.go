package models

import (
	"encoding/json"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

type Host struct {
	Id         int64  `orm:"auto"`
	Name       string `orm:"size(100);unique"`
	ServerAddr string `orm:"size(100);null"`

	User *User `orm:"rel(one);unique;column(user_id);null"`

	MinLimit           int       `orm:"default(0)"`
	MaxLimit           int       `orm:"default(0)"`
	Priority           bool      `orm:"default(false)"`
	Timeout            int       `orm:"default(0)"`
	HostsAPITokensJSON string    `orm:"type(text);null"`
	Active             bool      `orm:"default(false)"`
	SocketURL          string    `orm:"size(255);null"`
	AccessToken        string    `orm:"size(255);null"`
	StopTime           time.Time `orm:"type(datetime);null"`
	PaymentMethodID    string    `orm:"size(100);null"` // ID выбранного метода оплаты
	WorkerRunning      bool      `orm:"default(false)"` // флаг: запущен ли воркер
	CreatedAt          time.Time `orm:"auto_now_add;type(datetime)"`
	UpdatedAt          time.Time `orm:"auto_now;type(datetime)"`
}

func (h *Host) GetTokenForIP(ip string) (string, error) {
	// Десериализуем JSON-строку в map
	var tokensMap map[string]string
	if h.HostsAPITokensJSON == "" {
		tokensMap = make(map[string]string)
	} else {
		err := json.Unmarshal([]byte(h.HostsAPITokensJSON), &tokensMap)
		if err != nil {
			return "", err // Ошибка парсинга JSON
		}
	}

	// Проверяем, есть ли токен для IP
	token, exists := tokensMap[ip]
	if !exists {
		// Если IP нет в мапе, добавляем его с пустым токеном и обновляем JSON
		tokensMap[ip] = ""
		updatedJSON, err := json.Marshal(tokensMap)
		if err != nil {
			return "", err // Ошибка маршаллинга
		}
		h.HostsAPITokensJSON = string(updatedJSON)
		o := orm.NewOrm()
		o.Update(h)
		// Возвращаем пустой токен, внешний код решит, что с ним делать
		return "", nil
	}

	// Если токен пустой, возвращаем пустую строку
	if token == "" {
		return "", nil
	}

	// Иначе возвращаем найденный токен
	return token, nil
}

// Метод для установки токена для IP
func (h *Host) SetTokenForIP(ip, token string) error {
	var tokensMap map[string]string
	if h.HostsAPITokensJSON == "" {
		tokensMap = make(map[string]string)
	} else {
		err := json.Unmarshal([]byte(h.HostsAPITokensJSON), &tokensMap)
		if err != nil {
			return err
		}
	}

	tokensMap[ip] = token
	updatedJSON, err := json.Marshal(tokensMap)
	if err != nil {
		return err
	}
	h.HostsAPITokensJSON = string(updatedJSON)
	return nil
}

// Метод для получения *всего* мапа (может быть полезно для отладки или UI)
func (h *Host) GetFullTokensMap() (map[string]string, error) {
	var tokensMap map[string]string
	if h.HostsAPITokensJSON == "" {
		tokensMap = make(map[string]string)
	} else {
		err := json.Unmarshal([]byte(h.HostsAPITokensJSON), &tokensMap)
		if err != nil {
			return nil, err
		}
	}
	return tokensMap, nil
}

func init() {
	orm.RegisterModel(new(Host))
}
