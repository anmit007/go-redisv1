package server

import (
	"anmit007/go-redis/config"
	"anmit007/go-redis/core"
	"log"
	"net"
	"syscall"
)

func RunAsyncTCPServer() error {
	log.Println("Starting the async TCP server (epoll) on", config.Host, ":", config.Port)

	max_clients := 20000
	events := make([]syscall.EpollEvent, max_clients)

	serverFd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(serverFd)

	if err = syscall.SetNonblock(serverFd, true); err != nil {
		return err
	}

	ip4 := net.ParseIP(config.Host)
	if err = syscall.Bind(serverFd, &syscall.SockaddrInet4{
		Port: config.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	}); err != nil {
		return err
	}

	if err = syscall.Listen(serverFd, max_clients); err != nil {
		return err
	}

	epollFd, err := syscall.EpollCreate1(0)
	if err != nil {
		return err
	}
	defer syscall.Close(epollFd)

	event := syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:     int32(serverFd),
	}

	if err = syscall.EpollCtl(epollFd, syscall.EPOLL_CTL_ADD, serverFd, &event); err != nil {
		return err
	}

	log.Println("Epoll initialized, waiting for connections....")

	for {
		shouldRunCron()

		nevents, err := syscall.EpollWait(epollFd, events, 100)
		if err != nil {
			log.Println("EpollWait error", err)
			continue
		}

		for i := 0; i < nevents; i++ {
			fd := int(events[i].Fd)

			if fd == serverFd {
				for {
					clientFd, _, err := syscall.Accept(serverFd)
					if err != nil {
						if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
							break
						}
						log.Println("accept error", err)
						break
					}
					conn_clients++
					log.Println("New client connected, Fd:", clientFd, "Total Clients:", conn_clients)

					if err := syscall.SetNonblock(clientFd, true); err != nil {
						log.Println("SetNonblock error", err)
						syscall.Close(clientFd)
						continue
					}

					clientEvent := syscall.EpollEvent{
						Events: syscall.EPOLLIN,
						Fd:     int32(clientFd),
					}

					if err = syscall.EpollCtl(epollFd, syscall.EPOLL_CTL_ADD, clientFd, &clientEvent); err != nil {
						log.Println("epoll add client error:", err)
						syscall.Close(clientFd)
						continue
					}
				}
			} else {
				comm := core.FdComm{Fd: fd}
				cmds, err := readCommands(comm)
				if err != nil {
					if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
						continue
					}
					log.Println("Client disconnected with fd:", fd)
					syscall.EpollCtl(epollFd, syscall.EPOLL_CTL_DEL, fd, nil)
					syscall.Close(fd)
					conn_clients--
					continue
				}
				respond(cmds, comm)
			}
		}
	}
}
