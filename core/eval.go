package core

import (
	"errors"
	"net"
)

func EvalAndResponse(cmd *RedisCmd, c net.Conn) error {

	switch cmd.Cmd {
	case "PING":
		return evalPing(cmd.Args, c)
	default:
		return evalPing(cmd.Args, c)
	}
}

func evalPing(args []string, c net.Conn) error {
	var b []byte
	if len(args) >= 2 {
		return errors.New("ERR wrong number of aruguments for 'ping' command")
	}

	if len(args) == 0 {
		b = Encode("PONG", true)
	} else {
		b = Encode(args[0], false)
	}
	_, err := c.Write(b)
	return err
}
