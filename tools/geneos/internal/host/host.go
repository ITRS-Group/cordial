package host

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
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

// this is called from cmd root
func Init() {
	LOCAL = Get(LOCALHOST)
	ALL = Get(ALLHOSTS)
	ReadConfig()
}

// return the absolute path to the local Geneos installation
func Geneos() string {
	home := config.GetString("geneos")
	if home == "" {
		// fallback to support breaking change
		return config.GetString("itrshome")
	}
	return home
}

// interface method set

// Get returns a pointer to Host value. If passed an empty name, returns
// nil. If passed the special values LOCALHOST or ALL then it will
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
		c = &Host{config.New(), true, time.Time{}, nil}
		c.Set("name", LOCALHOST)
		c.GetOSReleaseEnv()
	case ALLHOSTS:
		if ALL != nil {
			return ALL
		}
		c = &Host{config.New(), true, time.Time{}, nil}
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
		c = &Host{config.New(), false, time.Time{}, nil}
		c.Set("name", name)
		hosts.Store(name, c)
	}

	c.Set("geneos", Geneos())
	return
}

func Add(h *Host) {
	h.loaded = true
}

func Delete(h *Host) {
	hosts.Delete(h.String())
}

func (h *Host) Exists() bool {
	return h.loaded
}

func (h *Host) Failed() bool {
	// if the failure was a while back, try again (XXX crude)
	if !h.lastAttempt.IsZero() && time.Since(h.lastAttempt) > 5*time.Second {
		return false
	}
	return h.failed != nil
}

func (h *Host) String() string {
	if h.IsSet("name") {
		return h.GetString("name")
	}
	return "unknown"
}

// return a string of the form "host:/path" for consistent use in output
func (h *Host) Path(path string) string {
	if h == LOCAL {
		return path
	}
	return fmt.Sprintf("%s:%s", h, path)
}

func (h *Host) GetOSReleaseEnv() (err error) {
	osinfo := make(map[string]string)
	switch runtime.GOOS {
	case "windows":
	default:
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
				return ErrInvalidArgs
			}
			key, value := s[0], s[1]
			value = strings.Trim(value, "\"")
			osinfo[key] = value
		}
	}
	h.Set("osinfo", osinfo)
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

// Filepath returns an absolute path relative to the Geneos installation
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

	return utils.JoinSlash(append([]string{h.GetString("geneos")}, strParts...)...)
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

	for name, host := range h.GetStringMap("hosts") {
		v := config.New()
		switch m := host.(type) {
		case map[string]interface{}:
			v.MergeConfigMap(m)
		default:
			log.Debug().Msgf("hosts value not a map[string]interface{} but a %T", host)
			continue
		}
		hosts.Store(name, &Host{v, true, time.Time{}, nil})
	}
}

func WriteConfig() error {
	n := config.New()

	hosts.Range(func(k, v interface{}) bool {
		name := k.(string)
		switch v := v.(type) {
		case *Host:
			n.Set("hosts."+name, v.AllSettings())
		}
		return true
	})

	if err := n.WriteConfigAs(UserHostsFilePath()); err != nil {
		return err
	}
	if utils.IsSuperuser() {
		uid, gid, _, _ := utils.GetIDs("")
		LOCAL.Chown(UserHostsFilePath(), uid, gid)
	}
	return nil
}

func UserHostsFilePath() string {
	userConfDir, _ := config.UserConfigDir()
	return filepath.Join(userConfDir, UserHostFile)
}
