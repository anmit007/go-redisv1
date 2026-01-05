//go:build windows

package core

import "syscall"

type FdComm struct {
	Fd syscall.Handle
}

func (f FdComm) Write(b []byte) (int, error) {
	return syscall.Write(f.Fd, b)
}

func (f FdComm) Read(b []byte) (int, error) {
	return syscall.Read(f.Fd, b)
}

