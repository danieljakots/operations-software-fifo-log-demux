package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"syscall"
)

var (
	logFifoPath = flag.String("log-fifo", "/var/run/fifo.pipe", "named pipe to read from")
	socketPath  = flag.String("socket", "/var/run/log.socket", "socket for local connections")
)

// Server listens on a UNIX socket and multiplexes data read from a pipe Reader
// onto eventual connections. Each connection has an associated optional
// regexp. If a client specifies a regular expression, only matching log lines
// will be sent to the client. Data is consumed from the pipe Reader even if
// there are no connections. It is assumed that clients can consume input as
// quickly as it is written, otherwise the whole Server will block waiting for
// them (which isn't a particularly clever design, but it works for the
// intended use case).
type Server struct {
	lock        sync.Mutex
	logFifoPath string
	conns       map[net.Conn]*regexp.Regexp
	listener    *net.UnixListener
}

func NewServer(logFifoPath, socketPath string) (*Server, error) {
	os.Remove(socketPath)

	l, err := net.ListenUnix("unix", &net.UnixAddr{socketPath, "unix"})
	if err != nil {
		return nil, err
	}
	return &Server{
		logFifoPath: logFifoPath,
		listener:    l,
		conns:       make(map[net.Conn]*regexp.Regexp),
	}, nil
}

func (s *Server) readLogs() {
	for {
		// By keeping this in an infinite loop fifo-log-demux is able to re-open
		// the logFifoPath after an EOF that's usually received cause the other side of
		// the pipe has been closed
		f, err := os.Open(s.logFifoPath)
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			data := scanner.Bytes()

			s.lock.Lock()
			for conn, exp := range s.conns {
				// If the data read from the named pipe doesn't match the
				// regular expression provided by this client, ignore it
				if !exp.Match(data) {
					continue
				}

				if _, err := conn.Write(append(data, byte('\n'))); err == nil {
					continue
				}

				if opErr, ok := err.(*net.OpError); ok {
					if syscallErr, ok := opErr.Err.(*os.SyscallError); ok {
						if errno, ok := syscallErr.Err.(syscall.Errno); ok && errno != syscall.EPIPE {
							log.Println("Error writing to client connection:", err)
						}
					}
				}
				delete(s.conns, conn)
				conn.Close()
			}
			s.lock.Unlock()
		}
		f.Close()
	}
}

func returnError(err error, conn net.Conn) {
	conn.Write([]byte(err.Error()))
	conn.Close()
}

func (s *Server) Run() {
	buf := make([]byte, 65536)

	go s.readLogs()

	for {
		conn, err := s.listener.AcceptUnix()
		if err != nil {
			log.Fatal(err)
		}

		// Read optional regex specified by the client program
		n, err := conn.Read(buf[:])
		if err != nil {
			returnError(err, conn)
			continue
		}

		regex, err := regexp.Compile(strings.TrimSuffix(string(buf[:n]), "\n"))
		if err != nil {
			// The client-supplied regular expression cannot be parsed. Return
			// an error and close the connection.
			returnError(err, conn)
			continue
		}

		s.lock.Lock()
		// one regex per connection
		s.conns[conn] = regex
		s.lock.Unlock()
	}
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	srv, err := NewServer(*logFifoPath, *socketPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Waiting for connections on", *socketPath)
	srv.Run()
}
