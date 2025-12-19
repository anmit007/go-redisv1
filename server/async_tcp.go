package server

import (
	"anmit007/go-redis/config"
	"anmit007/go-redis/core"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"syscall"
)

var conn_clients = 0

func readCommand(c io.ReadWriter) (*core.RedisCmd, error) {
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

func respondError(err error, c io.ReadWriter) {
	c.Write([]byte(fmt.Sprintf("-%s\r\n", err)))
}

func respond(cmd *core.RedisCmd, c io.ReadWriter) {
	err := core.EvalAndResponse(cmd, c)
	if err != nil {
		respondError(err, c)
	}
}
func RunAsyncTCPServer() error {
	log.Println("Starting the async TCP server on ", config.Host, ":", config.Port)

	max_clients := 20000

	events := make([]syscall.Kevent_t, max_clients)

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

	kqueueFd, err := syscall.Kqueue()
	if err != nil {
		log.Fatal(err)
	}
	defer syscall.Close(kqueueFd)

	change := syscall.Kevent_t{
		Ident:  uint64(serverFd),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
	}

	_, err = syscall.Kevent(kqueueFd, []syscall.Kevent_t{change}, nil, nil)
	if err != nil {
		return err
	}

	log.Println("Kqueue initialized, waiting for connections....")

	for {
		nevents, err := syscall.Kevent(
			kqueueFd,
			nil,
			events,
			nil,
		)

		if err != nil {
			log.Println("Kevent error", err)
			continue
		}

		for i := 0; i < nevents; i++ {
			fd := int(events[i].Ident)

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
					log.Println("New client connected , Fd:", clientFd, " Total Clients :", conn_clients)

					if err := syscall.SetNonblock(clientFd, true); err != nil {
						log.Println("SetNonblock error", err)
						syscall.Close(clientFd)
						continue
					}

					clientChange := syscall.Kevent_t{
						Ident:  uint64(clientFd),
						Filter: syscall.EVFILT_READ,
						Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
					}

					_, err = syscall.Kevent(kqueueFd, []syscall.Kevent_t{clientChange}, nil, nil)
					if err != nil {
						log.Println("kevent add client error:", err)
						syscall.Close(clientFd)
						continue
					}
				}
			} else {
				comm := core.FdComm{Fd: fd}
				cmd, err := readCommand(comm)
				if err != nil {
					log.Println("Client disconnected with fd:", fd)

					deleteChange := syscall.Kevent_t{
						Ident:  uint64(fd),
						Filter: syscall.EVFILT_READ,
						Flags:  syscall.EV_DELETE,
					}
					syscall.Kevent(kqueueFd, []syscall.Kevent_t{deleteChange}, nil, nil)
					syscall.Close(fd)
					conn_clients--
					continue

				}
				respond(cmd, comm)
			}
		}

	}
}
