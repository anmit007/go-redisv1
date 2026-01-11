package core

import (
	"anmit007/go-redis/config"
	"log"
	"math/rand"
	"time"
)

var randSource = rand.NewSource(time.Now().UnixNano())

func evict() {
	switch config.EVICTION_STRATEGY {
	case "allkeys-lru":
		evictALLkeyRandom()
	case "allkeys-lfu":
		evictLFU()
	case "simple-first":
		evictFirst()
	}
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

func incrementLfuLogWeight(key string) {
	obj := store[key]
	if obj.LfuLogWeight == 255 {
		return
	}
	baseval := float64(obj.LfuLogWeight) - 5.0
	if baseval < 0 {
		baseval = 0
	}

	probability := 1.0 / (baseval*config.LFU_LOG_BASE + 1)
	random := rand.Float64()
	if random < probability {
		obj.LfuLogWeight++
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
		return
	}
	elapsed := currentTime - lastDecayedAt
	numPeriods := elapsed / uint16(config.LFU_DECAY_TIME)
	if numPeriods > 0 {
		if uint16(obj.LfuLogWeight) > numPeriods {
			obj.LfuLogWeight -= uint8(numPeriods)
		} else {
			obj.LfuLogWeight = 0
		}
		obj.LastDecayedAt = currentTime
		store[key] = obj
	}
}

func evictALLkeyRandom() {
	evictCount := int64(config.EVICTION_RATIO * float64(len(store)))
	for k := range store {
		Del(k)
		evictCount--
		if evictCount <= 0 {
			break
		}
	}
}
