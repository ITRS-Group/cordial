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
	"os/exec"
	"os/user"
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
func ProcessStatus[T any](h host.Host, pid int) (pstats T, err error) {
	var scClkTck int64

	if reflect.TypeOf(pstats).Kind() != reflect.Ptr || reflect.TypeOf(pstats).Elem().Kind() != reflect.Struct {
		panic("pstats must be a pointer to a struct")
	}

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

	if reflect.ValueOf(pstats).IsNil() {
		pstats = reflect.New(st).Interface().(T)
	}

	sv := reflect.ValueOf(pstats).Elem()
	for i := 0; i < st.NumField(); i++ {
		ft := st.Field(i)
		fv := sv.Field(i)

		// special cases first
		switch ft.Name {
		case "OpenFiles":
			// if an int, then a count, if a slice or string, then a
			// list of open files. For now we just support count, but we
			// could add the list in future if needed.
			if fv.CanSet() {
				of := OpenFiles(h, pid)
				switch fv.Type().Kind() {
				case reflect.Int64, reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8:
					fv.SetInt(int64(len(of)))
				case reflect.Slice:
					switch fv.Type().Elem().Kind() {
					case reflect.Struct:
						if fv.Type().Elem() == reflect.TypeOf(ProcessFDs{}) {
							fv.Set(reflect.ValueOf(of))
						}

					case reflect.String:
						var openfiles []string
						for _, f := range of {
							openfiles = append(openfiles, f.Path)
						}
						fv.Set(reflect.ValueOf(openfiles))
					}
				case reflect.String:
					var openfiles []string
					for _, f := range of {
						openfiles = append(openfiles, f.Path)
					}
					if len(openfiles) > 0 {
						fv.SetString(strings.Join(openfiles, ","))
					} else {
						fv.SetString("NONE")
					}
				default:
					// do nothing if it's not an int or a slice or string
				}
			}

		case "OpenSockets":
			if fv.CanSet() {
				var opensockets int64 = 0
				of := OpenFiles(h, pid)
				for _, f := range of {
					if f.Conn != nil {
						// socket
						c := f.Conn
						if strings.HasPrefix(c.Protocol, "tcp") || strings.HasPrefix(c.Protocol, "udp") {
							opensockets++
						}
					}
				}
				fv.SetInt(opensockets)
			}

			// ListeningPorts can be a slice or a string. If a slice it
			// can be a slice of int or string.
		case "ListeningPorts":
			if fv.CanSet() {
				ports := ListeningPorts(h, pid)
				switch fv.Type().Kind() {
				case reflect.Slice:
					switch fv.Type().Elem().Kind() {
					case reflect.Int:
						fv.Set(reflect.ValueOf(ports))
					case reflect.String:
						var portlist []string
						for _, p := range ports {
							portlist = append(portlist, fmt.Sprint(p))
						}
						fv.Set(reflect.ValueOf(portlist))
					}
				case reflect.String:
					var portlist []string
					for _, p := range ports {
						portlist = append(portlist, fmt.Sprint(p))
					}
					if len(portlist) > 0 {
						fv.SetString(strings.Join(portlist, ","))
					} else {
						fv.SetString("NONE")
					}
				default:
					// do nothing if it's not a slice of int or string, or a string
				}
			}

		case "Exe":
			if fv.CanSet() && fv.Type().Kind() == reflect.String {
				exe := fmt.Sprintf("/proc/%d/exe", pid)
				exeVal, err := h.Readlink(exe)
				if err == nil {
					fv.SetString(strings.TrimSuffix(exeVal, " (deleted)"))
				}
			}

		case "Cmdline":
			if fv.CanSet() && fv.Type().Kind() == reflect.Slice && fv.Type().Elem().Kind() == reflect.String {
				cmdlineBytes, err := h.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
				if err == nil {
					fv.Set(reflect.ValueOf(strings.Split(strings.TrimSuffix(string(cmdlineBytes), "\000"), "\000")))
				}
			}

		case "Cwd":
			if fv.CanSet() && fv.Type().Kind() == reflect.String {
				cwd := fmt.Sprintf("/proc/%d/cwd", pid)
				cwdVal, err := h.Readlink(cwd)
				if err == nil {
					fv.SetString(cwdVal)
				}
			}

		case "UID":
			if fv.CanSet() && fv.Type().Kind() == reflect.Int {
				if v, ok := statusFields["Uid"]; ok {
					uidStr := strings.Fields(v)[0]
					if uid, err := strconv.Atoi(uidStr); err == nil {
						fv.SetInt(int64(uid))
						// find a Username field and set that too if it exists
						if uv := sv.FieldByName("Username"); uv.IsValid() {
							if uv.CanSet() && uv.Type().Kind() == reflect.String {
								uv.SetString(GetUsername(uid))
							}
						}
					}
				}
			}

		case "GID":
			if fv.CanSet() && fv.Type().Kind() == reflect.Int {
				if v, ok := statusFields["Gid"]; ok {
					gidStr := strings.Fields(v)[0]
					if gid, err := strconv.Atoi(gidStr); err == nil {
						fv.SetInt(int64(gid))
						// find a Groupname field and set that too if it exists
						if gv := sv.FieldByName("Groupname"); gv.IsValid() {
							if gv.CanSet() && gv.Type().Kind() == reflect.String {
								gv.SetString(GetGroupname(gid))
							}
						}
					}
				}
			}

		default:
			// for "stat" tag, lookup the field by index, if it exists
			if i, ok := ft.Tag.Lookup("stat"); ok {
				if idx, err := strconv.Atoi(i); err == nil {
					if len(statFields) > idx && fv.CanSet() {
						switch fv.Kind() {
						case reflect.String:
							fv.SetString(statFields[idx])
						case reflect.Int64, reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8:
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
					case reflect.Int64, reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8:
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
