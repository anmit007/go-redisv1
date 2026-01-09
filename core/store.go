package core

import (
	"anmit007/go-redis/config"
	"sync"
	"time"
)

var store map[string]*Obj
var storeMu sync.RWMutex

type Obj struct {
	Value         interface{}
	ExpiresAt     int64
	LfuLogWeight  uint8  // probabiltic counter (8 bits can count upto million)
	LastDecayedAt uint16 // decay weight

}

func init() {
	store = make(map[string]*Obj)
}

func NewObj(value interface{}, durationMs int64) *Obj {

	var expiresAt int64 = -1
	if durationMs > 0 {
		expiresAt = time.Now().UnixMilli() + durationMs
	}
	return &Obj{
		Value:        value,
		ExpiresAt:    expiresAt,
		LfuLogWeight: uint8(5),
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
		delete(store, k)
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
		return true
	}
	return false
}
