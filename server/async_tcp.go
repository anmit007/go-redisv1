package server

import (
	"anmit007/go-redis/core"
	"io"
	"strings"
	"time"
)

var conn_clients = 0
var cronFrequency time.Duration = 1 * time.Second
var lastCronExecTime time.Time = time.Now()

func toArrayString(ai []interface{}) ([]string, error) {
	as := make([]string, len(ai))
	for i := range ai {
		as[i] = ai[i].(string)
	}
	return as, nil
}

func readCommands(c io.ReadWriter) (core.RedisCmds, error) {
	var buff []byte = make([]byte, 512)
	n, err := c.Read(buff[:])
	if err != nil {
		return nil, err
	}
	values, err := core.Decode(buff[:n])
	if err != nil {
		return nil, err
	}
	var cmds []*core.RedisCmd = make([]*core.RedisCmd, 0)
	for _, val := range values {
		tokens, err := toArrayString(val.([]interface{}))
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, &core.RedisCmd{
			Cmd:  strings.ToUpper(tokens[0]),
			Args: tokens[1:],
		})
	}
	return cmds, nil
}

func respond(cmds core.RedisCmds, c io.ReadWriter) {
	core.EvalAndResponse(cmds, c)
}

func shouldRunCron() bool {
	if time.Now().After(lastCronExecTime.Add(cronFrequency)) {
		core.DeleteExpiredKeys()
		lastCronExecTime = time.Now()
		return true
	}
	return false
}
