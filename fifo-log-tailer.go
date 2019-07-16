package main

import (
	"flag"
	"io"
	"io/ioutil"
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
	tee := io.TeeReader(c, io.Writer(os.Stdout))
	_, err = ioutil.ReadAll(tee)
	if err != nil {
		log.Fatal("Read error: ", err)
	}
}
