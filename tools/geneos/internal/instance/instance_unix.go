//go:build !windows

/*
Copyright © 2023 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package instance

import (
	"bufio"
	"fmt"
	"io/fs"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/tklauser/go-sysconf"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

type OpenFiles struct {
	Path   string
	Stat   fs.FileInfo
	FD     string
	FDMode fs.FileMode
}

// Files returns a map of file descriptor (int) to file details
// (InstanceProcFiles) for all open, real, files for the process running
// as the instance. All paths that are not absolute paths are ignored.
// An empty map is returned if the process cannot be found.
func Files(i geneos.Instance) (files map[int]OpenFiles) {
	pid, err := GetPID(i)
	if err != nil {
		return
	}

	h := i.Host()

	file := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := h.ReadDir(file)
	if err != nil {
		return
	}

	files = make(map[int]OpenFiles, len(fds))

	for _, ent := range fds {
		fd := ent.Name()
		dest, err := h.Readlink(path.Join(file, fd))
		if err != nil {
			continue
		}
		if !path.IsAbs(dest) {
			continue
		}
		n, _ := strconv.Atoi(fd)

		fdPath := path.Join(file, fd)
		fdMode, err := h.Lstat(fdPath)
		if err != nil {
			continue
		}

		s, err := h.Stat(dest)
		if err != nil {
			continue
		}

		files[n] = OpenFiles{
			Path:   dest,
			Stat:   s,
			FD:     fdPath,
			FDMode: fdMode.Mode(),
		}
	}
	return
}

// ProcessStats is an example of a structure to pass to
// instance.ProcessStatus, using `stats` and `status` tags
type ProcessStats struct {
	Pid      int64         `stat:"0"`
	Utime    time.Duration `stat:"13"`
	Stime    time.Duration `stat:"14"`
	CUtime   time.Duration `stat:"15"`
	CStime   time.Duration `stat:"16"`
	State    string        `status:"State"`
	Threads  int64         `status:"Threads"`
	VmRSS    int64         `status:"VmRSS"`
	VmHWM    int64         `status:"VmHWM"`
	RssAnon  int64         `status:"RssAnon"`
	RssFile  int64         `status:"RssFile"`
	RssShmem int64         `status:"RssShmem"`
}

// ProcessStatus reads the instance process `stats` and `status` files
// in /proc and returns values that match the tags in the structure
// pstats. pstats must be a point to a struct and must be initialised
// before calling. Use the instance.ProcessStats struct as a useful
// default.
func ProcessStatus(i geneos.Instance, pstats any) (err error) {
	pid, err := GetPID(i)
	if err != nil {
		return
	}

	h := i.Host()
	// allow clock tick override for systems
	scClkTck := h.GetInt64("SC_CLK_TCK")
	if h == geneos.LOCAL {
		scClkTck, _ = sysconf.Sysconf(sysconf.SC_CLK_TCK)
	}

	if scClkTck == 0 {
		scClkTck = 100
	}

	// /proc/PID/stat contains utime and ctime, which are not in status.
	// field[1] is surrounds by parenthesis to protect embedded spaces,
	// so this has to be split more carefully
	stat, err := h.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return
	}
	pidStat, rest, _ := strings.Cut(string(stat), " (")
	execStat, rest, _ := strings.Cut(rest, ") ")

	statFields := []string{pidStat, execStat}
	statFields = append(statFields, strings.Split(rest, " ")...)

	status, err := h.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return
	}
	defer status.Close()

	statusFields := map[string]string{}

	statusLines := bufio.NewScanner(status)
	for statusLines.Scan() {
		name, value, found := strings.Cut(statusLines.Text(), ":")
		if !found {
			break
		}
		statusFields[name] = strings.TrimSpace(value)
	}

	st := reflect.TypeOf(pstats).Elem()
	sv := reflect.ValueOf(pstats).Elem()
	for i := 0; i < st.NumField(); i++ {
		ft := st.Field(i)
		fv := sv.Field(i)
		// for "stat" tag, lookup the field by index, if it exists
		if statField, ok := ft.Tag.Lookup("stat"); ok {
			if idx, err := strconv.Atoi(statField); err == nil {
				if len(statFields) > idx && fv.CanSet() {
					switch fv.Kind() {
					case reflect.String:
						fv.SetString(statFields[idx])
					case reflect.Int64:
						if iv, err := strconv.ParseInt(statFields[idx], 10, 64); err == nil {
							if fv.Type().String() == "time.Duration" {
								fv.SetInt(iv * int64(time.Second) / scClkTck)
							} else {
								fv.SetInt(iv)
							}
						}

					}
				}
			}
		}

		// for "status" tag, lookup the value by matching name, if it exists
		if statusField, ok := ft.Tag.Lookup("status"); ok {
			if v, ok := statusFields[statusField]; ok && fv.CanSet() {
				switch fv.Kind() {
				case reflect.String:
					fv.SetString(v)
				case reflect.Int64: // assume kB values
					v, found := strings.CutSuffix(v, " kB")
					if iv, err := strconv.ParseInt(v, 10, 64); err == nil {
						if found {
							fv.SetInt(iv * 1024)
						} else {
							fv.SetInt(iv)
						}
					}
				}
			}
		}
	}

	return
}
