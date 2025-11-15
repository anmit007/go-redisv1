package server

import (
	"anmit007/go-redis/config"
	"log"
	"net"
	"strconv"
)

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
