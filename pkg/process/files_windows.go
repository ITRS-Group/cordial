package process

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/itrs-group/cordial/pkg/host"
	gprocess "github.com/shirou/gopsutil/v4/process"
)

// OpenFiles TODO: Document its behavior, inputs, and effects.
func OpenFiles(h host.Host, pid int) (files []ProcessFDs) {
	if h == nil || !h.IsLocalhost() {
		return
	}
	proc, err := gprocess.NewProcess(int32(pid))
	if err != nil {
		return
	}
	openFiles, err := proc.OpenFiles()
	if err != nil {
		return
	}
	for _, openFile := range openFiles {
		name := filepath.Clean(openFile.Path)
		info, err := os.Stat(name)
		if err != nil {
			continue
		}
		linkInfo, err := os.Lstat(name)
		if err != nil {
			linkInfo = info
		}
		files = append(files, ProcessFDs{
			PID:   pid,
			FD:    int(openFile.Fd),
			Path:  name,
			Lstat: linkInfo,
			Stat:  info,
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].FD < files[j].FD })
	return
}
