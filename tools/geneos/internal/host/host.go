package host

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

const UserHostFile = "geneos-hosts.json"
const LOCALHOST = "localhost"
const ALLHOSTS = "all"

var LOCAL, ALL *Host

type Host struct {
	// use a viper to store config
	*viper.Viper

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
	home := viper.GetString("geneos")
	if home == "" {
		// fallback to support breaking change
		return viper.GetString("itrshome")
	}
	return home
}

// interface method set

// XXX new needs the top level viper and passes back a Sub()
func Get(name string) (c *Host) {
	switch name {
	case LOCALHOST:
		if LOCAL != nil {
			return LOCAL
		}
		c = &Host{viper.New(), true, nil}
		c.Set("name", LOCALHOST)
		c.GetOSReleaseEnv()
	case ALLHOSTS:
		if ALL != nil {
			return ALL
		}
		c = &Host{viper.New(), true, nil}
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
		c = &Host{viper.New(), false, nil}
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

// return an absolute path anchored in the root directory of the remote host
// this can also be LOCAL
func (h *Host) GeneosJoinPath(paths ...string) string {
	if h == nil {
		logError.Fatalln("host is nil")
	}

	return filepath.Join(append([]string{h.GetString("geneos")}, paths...)...)
}

func (h *Host) FullName(name string) string {
	if strings.Contains(name, "@") {
		return name
	}
	return name + "@" + h.String()
}

func AllHosts() (hs []*Host) {
	hs = []*Host{LOCAL}

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
	var hs *viper.Viper

	h := viper.New()
	h.SetConfigFile(UserHostsFilePath())
	h.ReadInConfig()
	if h.InConfig("hosts") {
		hs = h.Sub("hosts")
	}

	// recreate empty
	hosts = sync.Map{}
	// LOCAL = New(LOCALHOST)
	// ALL = New(ALLHOSTS)

	if hs != nil {
		for n, h := range hs.AllSettings() {
			v := viper.New()
			v.MergeConfigMap(h.(map[string]interface{}))
			hosts.Store(n, &Host{v, true, nil})
		}
	}
}

func WriteConfigFile() error {
	n := viper.New()

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
	userConfDir, err := os.UserConfigDir()
	if err != nil {
		logError.Fatalln(err)
	}
	return filepath.Join(userConfDir, UserHostFile)
}
