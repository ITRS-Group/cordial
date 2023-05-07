/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package host

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/remote"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"

	"github.com/rs/zerolog/log"
)

const ConfigSubdirName = "geneos"
const OldUserHostFile = "geneos-hosts.json"

var UserHostFile = filepath.Join(ConfigSubdirName, "hosts.json")

const LOCALHOST = "localhost"
const ALLHOSTS = "all"

var LOCAL, ALL *Host

type Host struct {
	remote.Remote
	*config.Config

	// loaded from config or just an instance?
	// always true for LOCALHOST and ALLHOSTS
	loaded bool

	// initially, if we fail to connect to host then mark as failed
	// as we run in single shot mode, also record error
	//
	// later, once we are long-running as a daemon then we can use
	// some sort of retry mechanism, but not for now
	lastAttempt time.Time
	failed      error
}

var hosts sync.Map

// Init initialises the host settings and is only called from the root
// command to set the initial values of host.LOCAL and host.ALL and
// reads the host configuration file.
func Init() {
	LOCAL = Get(LOCALHOST)
	ALL = Get(ALLHOSTS)
	ReadConfig()
}

// interface method set

// Get returns a pointer to Host value. If passed an empty name, returns
// nil. If passed the special values LOCALHOST or ALLHOSTS then it will
// return the respective special values LOCAL or ALL. Otherwise it tries
// to lookup an existing host with the given name to return or
// initialises a new value to return. This may not be an existing host.
//
// XXX new needs the top level config and passes back a Sub()
func Get(name string) (c *Host) {
	switch name {
	case "":
		return nil
	case LOCALHOST:
		if LOCAL != nil {
			return LOCAL
		}
		c = &Host{remote.NewLocal(), config.New(), true, time.Time{}, nil}
		c.Set("name", LOCALHOST)
		c.SetOSReleaseEnv()
	case ALLHOSTS:
		if ALL != nil {
			return ALL
		}
		c = &Host{remote.NewLocal(), config.New(), true, time.Time{}, nil}
		c.Set("name", ALLHOSTS)
	default:
		r, ok := hosts.Load(name)
		if ok {
			c, ok = r.(*Host)
			if ok {
				return
			}
		}
		// or bootstrap, but NOT save a new one
		c = &Host{remote.NewSSHRemote(name), config.New(), false, time.Time{}, nil}
		c.Set("name", name)
		hosts.Store(name, c)
	}

	c.Set("geneos", config.GetString("geneos", config.Default(config.GetString("itrshome"))))
	return
}

func Add(h *Host) {
	h.loaded = true
}

func Delete(h *Host) {
	hosts.Delete(h.String())
}

func (h *Host) Exists() bool {
	if h == nil {
		return false
	}
	return h.loaded
}

// SetOSReleaseEnv sets the `osinfo` configuration map to the values
// from either `/etc/os-release` (or `/usr/lib/os-release`) on Linux or
// simulates the values for Windows
func (h *Host) SetOSReleaseEnv() (err error) {
	osinfo := make(map[string]string)
	serverVersion := h.ServerVersion()
	if h.IsLocal() {
		home, _ := os.UserHomeDir()
		h.Set("homedir", home)
	}

	if strings.Contains(strings.ToLower(serverVersion), "windows") {
		// XXX simulate values? this also applies to "localhost"
		h.Set("os", "windows")
		osinfo["id"] = "windows"
		output, _ := h.Run("systeminfo")
		// if err == nil {
		// 	log.Debug().Msg(string(output))
		// }
		l := bufio.NewScanner(bytes.NewBuffer(output))
		for l.Scan() {
			line := l.Text()
			if strings.HasPrefix(line, " ") {
				continue
			}
			s := strings.SplitN(line, ":", 2)
			if len(s) < 2 {
				continue
			}
			name := strings.TrimSpace(s[0])
			val := strings.TrimSpace(s[1])
			switch name {
			case "OS Name":
				osinfo["name"] = val
				osinfo["pretty_name"] = val
			case "OS Version":
				osinfo["version"] = val
				vers := strings.Fields(val)
				osinfo["version_id"] = vers[0]
				osinfo["build_id"] = vers[len(vers)-1]
			}
		}
		if h.String() != LOCALHOST {
			output, err := h.Run(`cmd /c echo %USERPROFILE%`)
			if err != nil {
				log.Error().Err(err).Msg("")
			} else {
				dir := strings.TrimSpace(string(output))
				// tmp fix for ssh to windows
				dir = strings.Trim(dir, `"`)
				h.Set("homedir", dir)
			}
		}
	} else {
		h.Set("os", "linux")
		f, err := h.ReadFile("/etc/os-release")
		if err != nil {
			if f, err = h.ReadFile("/usr/lib/os-release"); err != nil {
				return fmt.Errorf("cannot open /etc/os-release or /usr/lib/os-release")
			}
		}

		releaseFile := bytes.NewBuffer(f)
		scanner := bufio.NewScanner(releaseFile)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) == 0 || strings.HasPrefix(line, "#") {
				continue
			}
			s := strings.SplitN(line, "=", 2)
			if len(s) != 2 {
				return remote.ErrInvalidArgs
			}
			key, value := s[0], s[1]
			value = strings.Trim(value, "\"")
			osinfo[strings.ToLower(key)] = value
		}
		if h.String() != LOCALHOST {
			output, err := h.Run("pwd")
			if err != nil {
				log.Error().Err(err).Msg("")
			} else {
				dir := strings.TrimSpace(string(output))
				h.Set("homedir", dir)
			}
		}
	}
	h.SetStringMapString("osinfo", osinfo)
	return
}

// Match returns a slice of all matching Hosts. Intended for use in
// range loops where the host could be specific or 'all'. If passed
// an empty string then returns an empty slice.
func Match(h string) (r []*Host) {
	switch h {
	case "":
		return []*Host{}
	case ALLHOSTS:
		return AllHosts()
	default:
		return []*Host{Get(h)}
	}
}

// Range will either return just the specific host it is called on, or
// if that is nil than the list of all hosts passed as args. If no args
// are passed and h is nil then all hosts are returned.
//
// This is a convenience to avoid a double layer of if and range in
// callers than want to work on specific component types.
func (h *Host) Range(hosts ...*Host) []*Host {
	switch h {
	case nil:
		if len(hosts) == 0 {
			return AllHosts()
		}
		return hosts
	case ALL:
		return AllHosts()
	default:
		return []*Host{h}
	}
}

// Filepath returns an absolute path relative to the Geneos root
// directory. Each argument is used as a path component and are joined
// using filepath.Join(). Each part can be a plain string or a type with
// a String() method - non-string types are rendered using fmt.Sprint()
// without further error checking.
func (h *Host) Filepath(parts ...interface{}) string {
	strParts := []string{}

	if h == nil {
		h = LOCAL
	}

	for _, p := range parts {
		switch s := p.(type) {
		case string:
			strParts = append(strParts, s)
		default:
			strParts = append(strParts, fmt.Sprint(s))
		}
	}

	return path.Join(append([]string{h.GetString("geneos")}, strParts...)...)
}

func (h *Host) FullName(name string) string {
	if strings.Contains(name, "@") {
		return name
	}
	return name + "@" + h.String()
}

// AllHosts returns a slice of all hosts, including LOCAL
func AllHosts() (hs []*Host) {
	hs = []*Host{LOCAL}
	hs = append(hs, RemoteHosts()...)
	return
}

// RemoteHosts returns a slice of al valid remote hosts
func RemoteHosts() (hs []*Host) {
	hs = []*Host{}

	hosts.Range(func(k, v interface{}) bool {
		h := Get(k.(string))
		if h.Failed() {
			log.Debug().Err(h.failed).Msg("")
		}
		if !h.Failed() {
			hs = append(hs, h)
		}
		return true
	})
	return
}

// ReadConfig loads configuration entries from the default host
// configuration file. If that fails, it tries the old location and
// migrates that file to the new location if found.
func ReadConfig() {
	h := config.New()
	h.SetConfigFile(UserHostsFilePath())
	if err := h.ReadInConfig(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// try old location
			userConfDir, _ := config.UserConfigDir()
			oldConfigFile := filepath.Join(userConfDir, OldUserHostFile)
			if s, err := os.Stat(oldConfigFile); err == nil {
				if !s.IsDir() {
					if f, err := os.Open(oldConfigFile); err == nil {
						defer f.Close()
						if w, err := os.Create(UserHostsFilePath()); err == nil {
							defer w.Close()
							if _, err := io.Copy(w, f); err == nil {
								f.Close()
								w.Close()
								os.Remove(oldConfigFile)
								if err := h.ReadInConfig(); err != nil {
									log.Debug().Err(err).Msg("old hosts file is unreadable, but still moved")
								}
							}
						}
					}
				}
			}
		}
	}

	// recreate empty
	hosts = sync.Map{}

	for _, host := range h.GetStringMap("hosts") {
		v := config.New()
		switch m := host.(type) {
		case map[string]interface{}:
			v.MergeConfigMap(m)
		default:
			log.Debug().Msgf("hosts value not a map[string]interface{} but a %T", host)
			continue
		}
		r := remote.NewSSHRemote(v.GetString("name"),
			remote.Username(v.GetString("username")),
			remote.Port(uint16(v.GetInt("port"))),
			remote.Hostname(v.GetString("hostname")))
		hosts.Store(v.GetString("name"), &Host{r, v, true, time.Time{}, nil})
	}
}

func WriteConfig() error {
	n := config.New()

	hosts.Range(func(k, v interface{}) bool {
		name := k.(string)
		switch v := v.(type) {
		case *Host:
			name = strings.ReplaceAll(name, ".", "-")
			n.Set("hosts."+name, v.AllSettings())
		}
		return true
	})

	userhostfile := UserHostsFilePath()

	if err := os.MkdirAll(filepath.Dir(userhostfile), 0775); err != nil {
		return err
	}
	if err := n.WriteConfigAs(userhostfile); err != nil {
		return err
	}
	if utils.IsSuperuser() {
		uid, gid, _, _ := utils.GetIDs("")
		LOCAL.Chown(userhostfile, uid, gid)
	}
	return nil
}

func UserHostsFilePath() string {
	userConfDir, _ := config.UserConfigDir()
	return filepath.Join(userConfDir, UserHostFile)
}
