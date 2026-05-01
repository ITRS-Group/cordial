//go:build !windows

/*
Copyright © 2022 ITRS Group

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

package process

import (
	"bufio"
	"fmt"
	"io/fs"
	"os/exec"
	"os/user"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/tklauser/go-sysconf"
)

// cache lookups, including fails
const notfound = "[NOT FOUND]"

var usernames sync.Map
var groupnames sync.Map

func GetUsername(uid int) (username string) {
	if u, ok := usernames.Load(uid); ok {
		username = u.(string)
		if username == notfound {
			username = fmt.Sprint(uid)
		}
		return
	}

	username = fmt.Sprint(uid)
	u, err := user.LookupId(username)
	if err != nil || u.Username == "" {
		usernames.Store(uid, notfound)
		return
	}
	username = u.Username
	usernames.Store(uid, username)

	return
}

func GetGroupname(gid int) (groupname string) {
	if g, ok := groupnames.Load(gid); ok {
		groupname = g.(string)
		if groupname == notfound {
			groupname = fmt.Sprint(gid)
		}
		return
	}

	groupname = fmt.Sprint(gid)
	g, err := user.LookupGroupId(groupname)
	if err != nil || g.Name == "" {
		groupnames.Store(gid, notfound)
		return
	}
	groupname = g.Name
	groupnames.Store(gid, groupname)

	return
}

// ProcessStatus reads the instance process `stats` and `status` files
// in /proc and returns values that match the tags in the structure
// pstats. pstats must be a point to a struct and must be initialised
// before calling. Use the instance.ProcessStats struct as a useful
// default.
func ProcessStatus(h host.Host, pid int, pstats any) (err error) {
	var scClkTck int64

	if h.IsLocal() {
		scClkTck, _ = sysconf.Sysconf(sysconf.SC_CLK_TCK)
	} else {
		scClkTck = 100
	}

	// /proc/PID/stat contains utime and ctime, which are not in status.
	stat, err := h.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return
	}

	// field[1] is surrounds by parenthesis to protect embedded spaces,
	// so this has to be split more carefully
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
					case reflect.Slice:
						if fv.Type().Elem().Kind() == reflect.String {
							fv.Set(reflect.ValueOf(strings.Fields(v)))
						}
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

			// do nothing if there are no tags we care about
		}
	}

	return
}

func prepareCmd(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}
	} else {
		cmd.SysProcAttr.Setsid = true
	}
}

// stub
func getWindowsProcCache(resetcache bool) (c procCache, ok bool) {
	return
}
