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

func main() {
	flag.Parse()

	// Connect to fifo-log-demux socket
	c, err := net.Dial("unix", *socketPath)
	if err != nil {
		log.Fatal("Dial error: ", err)
	}
	defer c.Close()

	// Send regexp to fifo-log-demux
	_, err = c.Write([]byte(*regexp))
	if err != nil {
		log.Fatal("Write error: ", err)
	}

	// Write to standard output what is read from fifo-log-demux
	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(io.Writer(os.Stdout), c, buf)
	if err != nil {
		log.Fatal("Copy error: ", err)
	}
}
