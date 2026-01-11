package config

import "time"

var Host string = "0.0.0.0"
var Port int = 7379
var MAX_KEYS int = 100000
var LFU_DECAY_TIME int = 1
var LFU_LOG_BASE float64 = 10
var AOFFILEPATH string = "./go-redis.aof"
var BGRewriteAOFInterval = 100 * time.Second
var AOF_FYSNC_POLICY string = "always"
var EVICTION_STRATEGY string = "allkeys-random"
var EVICTION_RATIO float64 = 0.4
