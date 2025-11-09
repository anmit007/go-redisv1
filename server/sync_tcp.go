package server

import (
	"anmit007/go-redis/config"
	"anmit007/go-redis/core"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

func readCommand(c net.Conn) (*core.RedisCmd, error) {
	var buff []byte = make([]byte, 512)
	n, err := c.Read(buff[:])
	if err != nil {
		return nil, err
	}

	tokens, err := core.DecodeArrayString(buff[:n])
	if err != nil {
		return nil, err
	}
	return &core.RedisCmd{
		Cmd:  strings.ToUpper(tokens[0]),
		Args: tokens[1:],
	}, nil
}

func respondError(err error, c net.Conn) {
	c.Write([]byte(fmt.Sprintf("-%s\r\n", err)))
}

func respond(cmd *core.RedisCmd, c net.Conn) {
	err := core.EvalAndResponse(cmd, c)
	if err != nil {
		respondError(err, c)
	}
}

func RunSyncTCPServer() {
	log.Println("Starting the sync TCP server....")

	var conn_clients = 0

	lsnr, err := net.Listen("tcp", config.Host+":"+strconv.Itoa(config.Port))
	if err != nil {
		panic(err)
	}

	for {
		c, err := lsnr.Accept()
		if err != nil {
			panic(err)
		}
		conn_clients += 1
		log.Println("Client connected with address: ", c.RemoteAddr(), "concurrent clients : ", conn_clients)

		for {
			cmd, err := readCommand(c)
			if err != nil {
				c.Close()
				conn_clients -= 1
				log.Println("Client Disconnected with address: ", c.RemoteAddr(), "concurrent Clients:", conn_clients)
				break
			}
			log.Println("command", cmd)
			respond(cmd, c)

		}

	}
}
