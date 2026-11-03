package process

import "io/fs"

// ProcessFDs TODO: Describe its purpose and intended use.
type ProcessFDs struct {
	PID   int
	FD    int
	Path  string
	Lstat fs.FileInfo
	Stat  fs.FileInfo
	Conn  *SocketConnection
}
