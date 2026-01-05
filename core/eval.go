package core

import (
	"bytes"
	"errors"
	"io"
	"log"
	"strconv"
	"time"
)

var RESP_NIL []byte = []byte("$-1\r\n")
var RESP_OK []byte = []byte("+OK\r\n")
var RESP_ZERO []byte = []byte(":0\r\n")
var RESP_ONE []byte = []byte(":1\r\n")
var RESP_MINUS_1 []byte = []byte(":-1\r\n")
var RESP_MINUS_2 []byte = []byte(":-2\r\n")

func EvalAndResponse(cmds RedisCmds, c io.ReadWriter) {
	var response []byte
	buff := bytes.NewBuffer(response)
	for _, cmd := range cmds {
		switch cmd.Cmd {
		case "PING":
			buff.Write(evalPing(cmd.Args))
		case "SET":
			buff.Write(evalSet(cmd.Args))
		case "GET":
			buff.Write(evalGet(cmd.Args))
		case "TTL":
			buff.Write(evalTTL(cmd.Args))
		case "DEL":
			buff.Write(evalDEL(cmd.Args))
		case "EXPIRE":
			buff.Write(evalExpire(cmd.Args))
		default:
			buff.Write(evalPing(cmd.Args))
		}
	}
	c.Write(buff.Bytes())
}

func evalPing(args []string) []byte {
	var b []byte
	if len(args) >= 2 {
		return Encode(errors.New("ERR wrong number of aruguments for 'ping' command"), false)
	}

	if len(args) == 0 {
		b = Encode("PONG", true)
	} else {
		b = Encode(args[0], false)
	}
	return b
}

func evalSet(args []string) []byte {
	if len(args) <= 1 {
		return Encode(errors.New("(error) ERR wrong number of arguments for 'set' command"), false)
	}

	var key, value string
	var exDurationMs int64 = -1

	key, value = args[0], args[1]

	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "EX", "ex":
			i++
			if i == len(args) {
				return Encode(errors.New("(error) ERR Syntax error"), false)
			}
			exDurationSec, err := strconv.ParseInt(args[3], 10, 64)
			if err != nil {
				return Encode(errors.New("(error) ERR value is not an integer or out of range"), false)
			}
			exDurationMs = exDurationSec * 1000
		default:
			return Encode(errors.New("(error) ERR Syntax error"), false)
		}
	}

	Put(key, NewObj(value, exDurationMs))
	return RESP_OK
}

func evalGet(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("(error) ERR wrong number of arguments for 'get' command"), false)
	}
	var key string = args[0]
	log.Println("WORKS")
	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}
	if obj.ExpiresAt != -1 && obj.ExpiresAt <= time.Now().UnixMilli() {
		return RESP_NIL
	}

	return Encode(obj.Value, false)
}

func evalTTL(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("(error) ERR wrong number of arguments for 'ttl' command"), false)
	}
	var key string = args[0]

	Obj := Get(key)
	if Obj == nil {
		return RESP_MINUS_2
	}
	if Obj.ExpiresAt == -1 {
		return RESP_MINUS_1
	}
	durationMs := Obj.ExpiresAt - time.Now().UnixMilli()
	if durationMs < 0 {
		return RESP_MINUS_2
	}
	return (Encode(int64(durationMs/1000), false))
}

func evalDEL(args []string) []byte {
	var cntDeleted int = 0
	for _, key := range args {
		if ok := Del(key); ok {
			cntDeleted++
		}
	}
	return (Encode(cntDeleted, false))

}

func evalExpire(args []string) []byte {
	if len(args) <= 1 {
		return Encode(errors.New("(error) ERR wrong number of arguments for 'expire' command"), false)
	}
	var key string = args[0]
	exDurationSec, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return Encode(errors.New("(error) ERR value is not an integer or out of range"), false)
	}

	obj := Get(key)

	if obj == nil {
		return RESP_ZERO
	}
	obj.ExpiresAt = time.Now().UnixMilli() + exDurationSec*1000
	return RESP_ONE
}
