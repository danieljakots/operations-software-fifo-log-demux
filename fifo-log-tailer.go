package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
)

var (
	socketPath = flag.String("socket", "/var/run/log.socket", "socket to communicate with fifo-log-demux")
	regexp     = flag.String("regexp", " ", "regular expression against which log entries are matched")
)

func readSocket(socketPath, regexp string) error {
	// Connect to fifo-log-demux socket
	c, err := net.Dial("unix", socketPath)
	if err != nil {
		return err
	}
	defer c.Close()

	// Send regexp to fifo-log-demux
	_, err = c.Write([]byte(regexp))
	if err != nil {
		return err
	}

	// Write to standard output what is read from fifo-log-demux
	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(os.Stdout, c, buf)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()

	for {
		// By keeping this in an infinite loop fifo-log-tailer is able to re-open
		// the UNIX socket after an error
		err := readSocket(*socketPath, *regexp)
		if err != nil {
			log.Println("Unable to read from socket:", err)
		}
	}
}
