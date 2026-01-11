package core

import (
	"anmit007/go-redis/config"
	"sync"
	"time"
)

var store map[string]*Obj
var storeMu sync.RWMutex

func init() {
	store = make(map[string]*Obj)
}

func NewObj(value interface{}, durationMs int64, oType uint8, oEncoding uint8) *Obj {

	var expiresAt int64 = -1
	if durationMs > 0 {
		expiresAt = time.Now().UnixMilli() + durationMs
	}
	return &Obj{
		Value:        value,
		ExpiresAt:    expiresAt,
		LfuLogWeight: uint8(5),
		TypeEncoding: oType | oEncoding,
	}

}

func Put(k string, obj *Obj) {
	storeMu.Lock()
	defer storeMu.Unlock()
	_, keyExists := store[k]
	if !keyExists && len(store) >= config.MAX_KEYS {
		evict()
	}

	store[k] = obj
	if KeyspaceStat[0] == nil {
		KeyspaceStat[0] = make(map[string]int)
	}
	KeyspaceStat[0]["keys"]++
	decayWeight(k)
	incrementLfuLogWeight(k)
}

func Get(k string) *Obj {
	storeMu.Lock()
	defer storeMu.Unlock()
	v, ok := store[k]
	if !ok {
		return nil
	}
	if v.ExpiresAt != -1 && v.ExpiresAt <= time.Now().UnixMilli() {
		Del(k)
		return nil
	}
	decayWeight(k)
	incrementLfuLogWeight(k)
	return v
}
func Del(k string) bool {
	storeMu.Lock()
	defer storeMu.Unlock()
	if _, ok := store[k]; ok {
		delete(store, k)
		KeyspaceStat[0]["keys"]--
		return true
	}
	return false
}
