package server

import (
	"anmit007/go-redis/config"
	"anmit007/go-redis/core"
	"io"
	"log"
	"net"
	"strings"
	"syscall"
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

		if time.Now().After(lastCronExecTime.Add(cronFrequency)) {
			core.DeleteExpiredKeys()
			lastCronExecTime = time.Now()
		}

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
				cmds, err := readCommands(comm)
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
				respond(cmds, comm)
			}
		}

	}
}
