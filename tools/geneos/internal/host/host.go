package host

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"
)

const UserHostFile = "geneos-hosts.json"
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
	// lastFailure time.Time
	failed error
}

var hosts sync.Map

// this is called from cmd root
func Init() {
	LOCAL = Get(LOCALHOST)
	ALL = Get(ALLHOSTS)
	ReadConfigFile()
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

// XXX new needs the top level config and passes back a Sub()
func Get(name string) (c *Host) {
	switch name {
	case LOCALHOST:
		if LOCAL != nil {
			return LOCAL
		}
		c = &Host{config.New(), true, nil}
		c.Set("name", LOCALHOST)
		c.GetOSReleaseEnv()
	case ALLHOSTS:
		if ALL != nil {
			return ALL
		}
		c = &Host{config.New(), true, nil}
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
		c = &Host{config.New(), false, nil}
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
	h.Set("osinfo", osinfo)
	return
}

// returns a slice of all matching Hosts. used mainly for range loops
// where the host could be specific or 'all'
func Match(h string) (r []*Host) {
	switch h {
	case ALLHOSTS:
		return AllHosts()
	default:
		return []*Host{Get(h)}
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

	return filepath.Join(append([]string{h.GetString("geneos")}, strParts...)...)
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
		if !h.Failed() {
			hs = append(hs, h)
		}
		return true
	})
	return
}

func ReadConfigFile() {
	h := config.New()
	h.SetConfigFile(UserHostsFilePath())
	h.ReadInConfig()

	// recreate empty
	hosts = sync.Map{}

	for n, h := range h.GetStringMap("hosts") {
		v := config.New()
		switch m := h.(type) {
		case map[string]interface{}:
			v.MergeConfigMap(m)
		default:
			log.Debug().Msgf("hosts value not a map[string]interface{} but a %T", h)
			continue
		}
		hosts.Store(n, &Host{v, true, nil})
	}
}

func WriteConfigFile() error {
	n := config.New()

	hosts.Range(func(k, v interface{}) bool {
		name := k.(string)
		switch v := v.(type) {
		case *Host:
			n.Set("hosts."+name, v.AllSettings())
		}
		return true
	})

	return n.WriteConfigAs(UserHostsFilePath())
}

func UserHostsFilePath() string {
	userConfDir, _ := os.UserConfigDir()
	return filepath.Join(userConfDir, UserHostFile)
}
