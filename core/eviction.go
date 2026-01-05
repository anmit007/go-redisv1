package core

import (
	"anmit007/go-redis/config"
	"log"
	"math/rand"
	"time"
)

var randSource = rand.NewSource(time.Now().UnixNano())

func evict() {
	evictLFU()
}

func evictFirst() {
	for k := range store {
		delete(store, k)
		return
	}
}

func evictLFU() {
	SAMPLING_SIZE := 5
	var victimKey string
	minWeight := uint8(255) // max value for int8
	count := 0
	for k, v := range store {
		if v.LfuLogWeight < minWeight {
			minWeight = v.LfuLogWeight
			victimKey = k
		}
		count++
		if count >= SAMPLING_SIZE {
			break
		}
	}
	if victimKey != "" {
		log.Println("Evicting key:", victimKey, "with weight:", minWeight)
		delete(store, victimKey)
	}
}

func getProbabilityDivisior(lfuWeight uint8) float64 {
	if lfuWeight > 62 {
		return float64(int64(1) << 62)
	}
	return float64(int64(1) << int(lfuWeight))
}
func incrementLfuLogWeight(key string) {
	obj := store[key]
	if obj.LfuLogWeight == 255 {
		return
	}
	lfuWeight := obj.LfuLogWeight
	probability := 1 / getProbabilityDivisior(lfuWeight)
	random := rand.Float64()
	if random < probability {
		obj.LfuLogWeight = lfuWeight + 1
		store[key] = obj
	}
}

func decayWeight(key string) {
	obj := store[key]
	currentTime := uint16(time.Now().Unix() / 60) // in minutes
	lastDecayedAt := obj.LastDecayedAt
	if lastDecayedAt == 0 {
		obj.LastDecayedAt = currentTime
		lastDecayedAt = currentTime
	}
	if currentTime-lastDecayedAt > uint16(config.LFU_DECAY_TIME) {
		if obj.LfuLogWeight <= 10 {
			obj.LfuLogWeight = obj.LfuLogWeight - 1
		} else {
			obj.LfuLogWeight = obj.LfuLogWeight / 2
		}
		obj.LastDecayedAt = currentTime
		store[key] = obj
		log.Println("decaying weight for key:", key, "from", lastDecayedAt, "to", currentTime, "weight:", store[key].LfuLogWeight)
	}
}
