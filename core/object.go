package core

type Obj struct {
	Value         interface{}
	ExpiresAt     int64
	LfuLogWeight  uint8  // probabiltic counter (8 bits can count upto million)
	LastDecayedAt uint16 // decay weight
	TypeEncoding  uint8
}

var OBJ_TYPE_STRING uint8 = 0 << 4
var OBJ_ENCODING_RAW uint8 = 0
var OBJ_ENCODING_INT uint8 = 1
var OBJ_ENCODING_EMBSTR uint8 = 8
