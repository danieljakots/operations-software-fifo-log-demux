fifo-log-demux
==============

Allow multiple clients to read logs from a named pipe.

Servers such as NGINX and Apache Traffic Server can send their access logs to a
named pipe (FIFO) on the filesystem, doing a reasonably good job at fault
isolation if the pipe can't be written to during normal operation.
Unfortunately, NGINX will still block on reloads and other occasions unless
there is an active listener on the pipe. Apache Traffic Server, instead, logs
an error message when the named pipe is full (ie: nobody is reading from it).
The purpose of this program is to constantly read from the logs FIFO, and
provide a copy of it to clients connecting to the daemon through a local UNIX
socket. Clients can specify an optional regular expression to filter for
specific log entries.

This allows real-time debugging and analysis, without necessarily
having to store logs to disk at any time (which may be undesirable for
compliance purposes).

Based on https://git.autistici.org/ai/nginx-log-peeker

## Usage

1. Start the fifo-log-demux daemon:

        $ fifo-log-demux -log-fifo /var/log/log.pipe -socket /run/user/$(id -u)/log.sock

2. Connect to the log UNIX socket to retrieve logs in real-time whenever you
   need to debug something:

        $ LOG_SOCKET="/run/user/$(id -u)/log.sock" fifo-log-tailer 'GET|POST'
