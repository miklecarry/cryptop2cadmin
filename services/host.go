package services

import (
	"hostmanager/models"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

type HostRuntimeState struct {
	Online   bool
	Enabled  bool
	LastSeen time.Time
}

var (
	hostStates = make(map[string]*HostRuntimeState) // ключ — Name
	stateMutex sync.RWMutex
)

const OfflineTimeout = 6 * time.Minute

// Обновить состояние хоста
func UpdateHostState(name string, enabled bool) {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	if state, exists := hostStates[name]; exists {
		state.Online = true
		state.Enabled = enabled
		state.LastSeen = time.Now()
	} else {
		hostStates[name] = &HostRuntimeState{
			Online:   true,
			Enabled:  enabled,
			LastSeen: time.Now(),
		}
	}
}

func HasActivePriorityHost() bool {
	o := orm.NewOrm()
	var hosts []models.Host
	o.QueryTable("host").Filter("Priority", true).All(&hosts)

	for _, h := range hosts {
		state := GetHostState(h.Name)
		if state.Online && state.Enabled {
			return true
		}
	}
	return false
}

func GetHostState(name string) *HostRuntimeState {
	stateMutex.RLock()
	state, exists := hostStates[name]
	stateMutex.RUnlock()

	if !exists {
		return &HostRuntimeState{Online: false, Enabled: false}
	}

	if time.Since(state.LastSeen) > OfflineTimeout {
		return &HostRuntimeState{Online: false, Enabled: false}
	}

	return &HostRuntimeState{
		Online:   true,
		Enabled:  state.Enabled,
		LastSeen: state.LastSeen,
	}
}

func DeleteHostState(name string) {
	stateMutex.Lock()
	delete(hostStates, name)
	stateMutex.Unlock()
}

// Фоновая очистка старых записей (опционально)
func StartStateCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			stateMutex.Lock()
			for ip, state := range hostStates {
				if time.Since(state.LastSeen) > OfflineTimeout {
					delete(hostStates, ip)
				}
			}
			stateMutex.Unlock()
		}
	}()
}
