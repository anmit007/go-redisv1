package main

import (
	"anmit007/go-redis/config"
	"anmit007/go-redis/core"
	"anmit007/go-redis/server"
	"flag"
	"log"
)

func setupFlags() {
	flag.StringVar(&config.Host, "host", "0.0.0.0", "host for the go-redis server")
	flag.IntVar(&config.Port, "port", 7379, "port for go-redis server")
	flag.Parse()
}

func main() {
	setupFlags()
	log.Println("Starting the go-redis server....", config.Host, config.Port)
	if err := core.LoadAOF(); err != nil {
		log.Println("Error loading AOF file", err)
	}
	server.RunAsyncTCPServer()
}
