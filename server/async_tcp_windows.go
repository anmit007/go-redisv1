//go:build windows

package server

import (
	"anmit007/go-redis/config"
	"fmt"
	"log"
	"net"
	"syscall"
	"time"
	"unsafe"
)

// --- Windows DLL Definitions ---
var (
	ws2_32   = syscall.NewLazyDLL("ws2_32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procWSARecv                   = ws2_32.NewProc("WSARecv")
	procWSASend                   = ws2_32.NewProc("WSASend")
	procCreateIoCompletionPort    = kernel32.NewProc("CreateIoCompletionPort")
	procGetQueuedCompletionStatus = kernel32.NewProc("GetQueuedCompletionStatus")
)

const (
	OP_ACCEPT = iota
	OP_READ
	OP_WRITE
)

// --- Structs for IOCP ---

type WSABUF struct {
	Len uint32
	Buf *byte
}

type overlappedEx struct {
	overlapped syscall.Overlapped
	opType     int
	socket     syscall.Handle
	wsaBuf     WSABUF
	buffer     [512]byte
	flags      uint32
	bytes      uint32
}

// --- Helper Functions for Syscalls ---

func createIoCompletionPort(handle syscall.Handle, port syscall.Handle, key uintptr, threads uint32) (syscall.Handle, error) {
	r1, _, err := procCreateIoCompletionPort.Call(
		uintptr(handle),
		uintptr(port),
		key,
		uintptr(threads),
	)
	if r1 == 0 {
		return 0, err
	}
	return syscall.Handle(r1), nil
}

func getQueuedCompletionStatus(port syscall.Handle, bytes *uint32, key *uintptr, overlapped **syscall.Overlapped, timeout uint32) error {
	r1, _, err := procGetQueuedCompletionStatus.Call(
		uintptr(port),
		uintptr(unsafe.Pointer(bytes)),
		uintptr(unsafe.Pointer(key)),
		uintptr(unsafe.Pointer(overlapped)),
		uintptr(timeout),
	)
	if r1 == 0 {
		if err != nil && err != syscall.Errno(0) {
			return err
		}
	}
	return nil
}

func wsaRecv(socket syscall.Handle, buffers *WSABUF, bufferCount uint32, bytes *uint32, flags *uint32, overlapped *syscall.Overlapped) error {
	r1, _, err := procWSARecv.Call(
		uintptr(socket),
		uintptr(unsafe.Pointer(buffers)),
		uintptr(bufferCount),
		uintptr(unsafe.Pointer(bytes)),
		uintptr(unsafe.Pointer(flags)),
		uintptr(unsafe.Pointer(overlapped)),
		0,
	)
	if r1 != 0 {
		if err == syscall.ERROR_IO_PENDING {
			return nil
		}
		return err
	}
	return nil
}

func postRecv(socket syscall.Handle, ovEx *overlappedEx) error {
	ovEx.opType = OP_READ
	ovEx.socket = socket
	ovEx.flags = 0
	ovEx.wsaBuf.Len = uint32(len(ovEx.buffer))
	ovEx.wsaBuf.Buf = &ovEx.buffer[0]

	// Reset overlapped structure
	ovEx.overlapped.Internal = 0
	ovEx.overlapped.InternalHigh = 0
	ovEx.overlapped.Offset = 0
	ovEx.overlapped.OffsetHigh = 0
	ovEx.overlapped.HEvent = 0

	return wsaRecv(socket, &ovEx.wsaBuf, 1, &ovEx.bytes, &ovEx.flags, &ovEx.overlapped)
}

// --- IO Wrapper for Core Logic ---

type socketComm struct {
	socket syscall.Handle
	buffer []byte
}

func (s socketComm) Read(p []byte) (n int, err error) {
	if len(s.buffer) > 0 {
		n = copy(p, s.buffer)
		return n, nil
	}
	return 0, fmt.Errorf("no data")
}

func (s socketComm) Write(p []byte) (n int, err error) {
	var sent uint32
	wsaBuf := WSABUF{
		Len: uint32(len(p)),
		Buf: &p[0],
	}

	r1, _, e := procWSASend.Call(
		uintptr(s.socket),
		uintptr(unsafe.Pointer(&wsaBuf)),
		1,
		uintptr(unsafe.Pointer(&sent)),
		0,
		0,
		0,
	)

	if r1 != 0 {
		return 0, e
	}
	return int(sent), nil
}

// --- Main Server Function ---

func RunAsyncTCPServer() error {
	log.Println("Starting the async TCP server (IOCP) on", config.Host, ":", config.Port)

	// 1. Initialize Winsock
	var wsaData syscall.WSAData
	if err := syscall.WSAStartup(uint32(0x202), &wsaData); err != nil {
		return fmt.Errorf("WSAStartup failed: %v", err)
	}
	defer syscall.WSACleanup()

	// 2. Create IOCP
	iocpHandle, err := createIoCompletionPort(syscall.InvalidHandle, 0, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to create IOCP: %v", err)
	}
	defer syscall.CloseHandle(iocpHandle)

	// 3. Create Listener using Standard Go Net (Hybrid Approach)
	// This avoids the "syscall.Accept not supported" error on Windows
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port))
	if err != nil {
		return err
	}
	// We cast to TCPListener to get the underlying file handle if needed,
	// but for the accept loop, we will use the high-level listener.SetDeadline
	tcpListener := listener.(*net.TCPListener)
	defer tcpListener.Close()

	log.Println("IOCP initialized. Waiting for connections...")

	clients := make(map[syscall.Handle]*overlappedEx)

	// --- The Event Loop ---
	for {
		// Run Cron Job
		shouldRunCron()

		// A. Check for IOCP Events (Existing Clients)
		var bytesTransferred uint32
		var completionKey uintptr
		var pOverlapped *syscall.Overlapped

		// Non-blocking check (0 timeout) or very short timeout
		err := getQueuedCompletionStatus(iocpHandle, &bytesTransferred, &completionKey, &pOverlapped, 10)

		if pOverlapped != nil {
			// --- Handle IO Event ---
			ovEx := (*overlappedEx)(unsafe.Pointer(pOverlapped))
			clientSocket := ovEx.socket

			if bytesTransferred == 0 {
				// Client Disconnected
				// log.Println("Client disconnected, Socket:", clientSocket)
				syscall.Closesocket(clientSocket)
				delete(clients, clientSocket)
				conn_clients--
			} else if ovEx.opType == OP_READ {
				// Data Received
				comm := socketComm{
					socket: clientSocket,
					buffer: ovEx.buffer[:bytesTransferred],
				}

				// Process Command (Single-threaded)
				cmds, cmdErr := readCommands(comm)
				if cmdErr != nil {
					// log.Println("Read error:", cmdErr)
					syscall.Closesocket(clientSocket)
					delete(clients, clientSocket)
					conn_clients--
				} else {
					respond(cmds, comm)
					// Post next read
					postRecv(clientSocket, ovEx)
				}
			}
		}

		// B. Check for New Connections (Accept)
		// We set a very short deadline to make Accept non-blocking
		tcpListener.SetDeadline(time.Now().Add(1 * time.Millisecond))
		conn, err := tcpListener.Accept()

		if err == nil {
			// --- New Connection Accepted ---
			tcpConn := conn.(*net.TCPConn)

			// Extract the Raw File Handle
			file, err := tcpConn.File()
			if err != nil {
				log.Println("Failed to get file handle:", err)
				conn.Close()
				continue
			}
			// Important: We must detach the file handle or Go will close it when 'file' is GC'd
			// But since we can't easily detach in pure Go, we just dup it effectively by using it.
			// Actually, on Windows, file.Fd() returns the handle.
			// We need to be careful: Go's net poller might fight us if we don't duplicate.
			// However, a simpler way is to just use the handle and ensure 'conn' stays alive or is managed.

			// BETTER WAY: Use the handle directly.
			fd := syscall.Handle(file.Fd())

			// Go puts the socket in non-blocking mode by default.
			conn_clients++
			log.Println("New client connected, Socket:", fd, "Total:", conn_clients)

			// Add to our IOCP Port
			_, err = createIoCompletionPort(fd, iocpHandle, uintptr(fd), 0)
			if err != nil {
				log.Println("IOCP Assoc Failed:", err)
				conn.Close()
				continue
			}

			// Post Initial Read
			ovEx := &overlappedEx{}
			clients[fd] = ovEx

			// We must keep 'file' alive so the handle isn't closed by Go's finalizer immediately
			// But actually, 'file' is a copy. 'conn' is the original.
			// We will let Go manage the 'conn' object lifecycle if we want, OR we take over.
			// To take over completely and prevent Go from interfering:
			// Using the raw handle from File() usually dups it.

			if err := postRecv(fd, ovEx); err != nil {
				log.Println("PostRecv Failed:", err)
				conn.Close() // This closes the underlying FD too
				delete(clients, fd)
				conn_clients--
			}

			// We don't need 'conn' anymore for Read/Write since we use raw syscalls,
			// but we need to keep the handle open.
			// file.Close() closes the *copy* of the file descriptor returned by File(), NOT the connection itself.
			// BUT: we want to keep the FD open.
			// Let's NOT close 'file' or 'conn' here, or the socket dies.
			// We rely on syscall.Closesocket(clientSocket) in the event loop to kill it.

		} else {
			// Check if it's a timeout (normal) or a real error
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				// This is fine, just no new connections
			} else {
				// log.Println("Accept error:", err)
			}
		}
	}
}
