package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
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
	lock     sync.Mutex
	pipe     io.Reader
	conns    map[net.Conn]*regexp.Regexp
	listener *net.UnixListener
}

func NewServer(pipe io.Reader, socketPath string) (*Server, error) {
	os.Remove(socketPath)

	l, err := net.ListenUnix("unix", &net.UnixAddr{socketPath, "unix"})
	if err != nil {
		return nil, err
	}
	return &Server{
		pipe:     pipe,
		listener: l,
		conns:    make(map[net.Conn]*regexp.Regexp),
	}, nil
}

func (s *Server) readLogs() {
	scanner := bufio.NewScanner(s.pipe)

	for scanner.Scan() {
		data := scanner.Bytes()

		s.lock.Lock()
		for conn, exp := range s.conns {
			// If the data read from the named pipe matches the regular
			// expression provided by this client
			if exp.Match(data) {
				if _, err := conn.Write(append(data, byte('\n'))); err != nil {
					if err != syscall.EPIPE {
						log.Println("Error writing to client connection:", err)
					}
					delete(s.conns, conn)
					conn.Close()
				}
			}
		}
		s.lock.Unlock()
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

	f, err := os.Open(*logFifoPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	srv, err := NewServer(f, *socketPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Waiting for connections on", *socketPath)
	srv.Run()
}
