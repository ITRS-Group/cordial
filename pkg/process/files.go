package process

import "io/fs"

type ProcessFDs struct {
	PID   int
	FD    int
	Path  string
	Lstat fs.FileInfo
	Stat  fs.FileInfo
	Conn  *SocketConnection
}
