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
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tklauser/go-sysconf"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Files returns a map of file descriptor to file details for all files
// for the instance. An empty map is returned if the process cannot be
// found.
func Files(i geneos.Instance) (files []ProcessFDs) {
	pid, err := GetPID(i)
	if err != nil {
		return
	}

	h := i.Host()

	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := h.ReadDir(fdDir)
	if err != nil {
		return
	}
	slices.SortFunc(fds, func(a, b fs.DirEntry) int {
		an, _ := strconv.Atoi(a.Name())
		bn, _ := strconv.Atoi(b.Name())
		if an < bn {
			return -1
		}
		if an == bn {
			return 0
		}
		return 1
	})

	for _, ent := range fds {
		fd := ent.Name()
		n, _ := strconv.Atoi(fd)

		fdPath := path.Join(fdDir, fd)

		linkVal, err := h.Readlink(fdPath)
		if err != nil {
			continue
		}

		linkStat, err := h.Lstat(fdPath)
		if err != nil {
			continue
		}

		if !path.IsAbs(linkVal) {
			// check for socket
			sc, err := SocketToConn(h, linkVal)
			if err != nil {
				continue
			}

			files = append(files, ProcessFDs{
				PID:  pid,
				FD:   n,
				Path: linkVal,
				Stat: linkStat,
				Conn: sc,
			})

			continue
		}

		destStat, err := h.Stat(fdPath)
		if err != nil {
			continue
		}

		files = append(files, ProcessFDs{
			PID:   pid,
			FD:    n,
			Path:  linkVal,
			Lstat: linkStat,
			Stat:  destStat,
		})
	}
	return
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

		// special cases
		switch ft.Name {
		case "OpenFiles":
			if fv.CanSet() {
				var openfiles int64
				fdDir := fmt.Sprintf("/proc/%d/fd", pid)
				dirents, err := h.ReadDir(fdDir)
				if err != nil {
					continue
				}
				for _, d := range dirents {
					// skip non symlinks
					s, err := d.Info()
					if err != nil || s.Mode()&fs.ModeSymlink == 0 {
						continue
					}

					linkVal, err := h.Readlink(path.Join(fdDir, d.Name()))
					if err != nil {
						continue
					}
					if path.IsAbs(linkVal) {
						openfiles++
					}
				}
				fv.SetInt(openfiles)
			}

		case "OpenSockets":
			if fv.CanSet() {
				var opensockets int64
				fdDir := fmt.Sprintf("/proc/%d/fd", pid)
				dirents, err := h.ReadDir(fdDir)
				if err != nil {
					continue
				}
				for _, d := range dirents {
					// skip non symlinks
					s, err := d.Info()
					if err != nil || s.Mode()&fs.ModeSymlink == 0 {
						continue
					}
					linkVal, err := h.Readlink(path.Join(fdDir, d.Name()))
					if err != nil {
						continue
					}
					if strings.HasPrefix(linkVal, "socket:[") {
						opensockets++
					}
				}
				fv.SetInt(opensockets)
			}

		default:
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
	}

	return
}
