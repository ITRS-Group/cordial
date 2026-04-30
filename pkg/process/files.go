package process

import "io/fs"

type ProcessFDs struct {
	PID   int64
	FD    int
	Path  string
	Lstat fs.FileInfo
	Stat  fs.FileInfo
	Conn  *SocketConnection
}
