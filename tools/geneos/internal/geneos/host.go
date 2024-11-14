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

package geneos

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"

	"github.com/rs/zerolog/log"
)

// OldUserHostFile is a legacy name that will be deprecated in the
// future
const OldUserHostFile = "geneos-hosts.json"

// Host defines a host for seamless remote management
type Host struct {
	host.Host
	*config.Config

	// hidden from wildcard loops?
	hidden bool

	// loaded from config or just an instance?
	// always true for LOCALHOST and ALLHOSTS
	loaded bool
}

// hosts holds all of the configured remote hosts. It does not store
// localhost or all.
var hosts sync.Map

// Default host labels that always exist
const (
	LOCALHOST   = "localhost"
	ALLHOSTS    = "all"
	UNKNOWNHOST = "UNKNOWN"
)

// LOCAL and ALL are the global Host values that represent LOCALHOST and
// ALLHOSTS from above, and must always exist
var (
	LOCAL   *Host
	ALL     *Host
	UNKNOWN *Host
)

// InitHosts initialises the host settings and is only called from the
// root command to set the initial values of host.LOCAL and host.ALL and
// reads the host configuration file. LOCAL and ALL cannot be
// initialised outside a function as there would be a definition loop.
func InitHosts(app string) {
	LOCAL = NewHost(LOCALHOST)
	ALL = NewHost(ALLHOSTS)
	UNKNOWN = NewHost(UNKNOWNHOST)
	LoadHostConfig()
}

// NewHost is a factory method for Host. It returns an initialised Host
// and will store it in the global map. If name is "localhost" or "all"
// or "unknown" then it returns pseudo-hosts used for testing and
// ranges.
func NewHost(name string, options ...any) (h *Host) {
	switch name {
	case "":
		return nil
	case LOCALHOST:
		if LOCAL != nil {
			return LOCAL
		}
		h = &Host{host.NewLocal(), config.New(), false, true}
		h.Set("name", LOCALHOST)
		hostname, _ := os.Hostname()
		h.Set("hostname", hostname)
		h.SetOSReleaseEnv()
	case ALLHOSTS:
		if ALL != nil {
			return ALL
		}
		h = &Host{host.NewLocal(), config.New(), false, true}
		h.Set("name", ALLHOSTS)
	case UNKNOWNHOST:
		if UNKNOWN != nil {
			return UNKNOWN
		}
		h = &Host{host.NewLocal(), config.New(), false, true}
		h.Set("name", UNKNOWNHOST)
	default:
		r, ok := hosts.Load(name)
		if ok {
			h, ok = r.(*Host)
			if ok {
				return
			}
		}
		// or bootstrap, but NOT save a new one, with only the name set
		h = &Host{host.NewSSHRemote(name, options...), config.New(), false, false}
		h.Set("name", name)
		hosts.Store(name, h)
	}

	h.Set(execname, config.GetString(execname, config.Default(config.GetString("itrshome"))))
	return
}

func (h *Host) String() string {
	if h == nil {
		return ""
	}
	return h.GetString("name")
}

// GetHost returns a pointer to Host value. If passed an empty name,
// returns nil. If passed the special values LOCALHOST or ALLHOSTS then
// it will return the respective special values LOCAL or ALL. Otherwise
// it tries to lookup an existing host with the given name.
//
// It will return UNKNOWN if the named host is not found. Use NewHost()
// to initialise a new host
func GetHost(name string) (h *Host) {
	switch name {
	case "":
		return nil
	case LOCALHOST:
		return LOCAL
	case ALLHOSTS:
		return ALL
	default:
		r, ok := hosts.Load(name)
		if ok {
			h, ok = r.(*Host)
			if ok {
				return
			}
		}
		return UNKNOWN
	}
}

// Delete host h from the internal list of hosts. Does not change the
// on-disk configuration file
func (h *Host) Delete() {
	hosts.Delete(h.String())
}

// Valid marks the host loaded and so qualified for saving to the hosts config
// file
func (h *Host) Valid() {
	h.loaded = true
}

// Exists returns true if the host h has an initialised configuration
//
// To check is a host can be contacted use the IsAvailable() instead
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
		home, _ := config.UserHomeDir()
		h.Set("homedir", home)
	}

	if strings.Contains(strings.ToLower(serverVersion), "windows") {
		// XXX simulate values? this also applies to "localhost"
		h.Set("os", "windows")
		osinfo["id"] = "windows"
		cmd := exec.Command("systeminfo")
		output, _ := h.Run(cmd, "")
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
		if !h.IsLocal() {
			cmd := exec.Command("cmd", "/c", "echo", "%USERPROFILE%")
			output, err := h.Run(cmd, "")
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
				return ErrInvalidArgs
			}
			key, value := s[0], s[1]
			value = strings.Trim(value, "\"")
			osinfo[strings.ToLower(key)] = value
		}
		if !h.IsLocal() {
			dir, err := h.Getwd()
			if err != nil {
				return err
			}
			dir = strings.TrimSpace(string(dir))
			h.Set("homedir", dir)
		}
	}
	h.SetStringMapString("osinfo", osinfo)
	return
}

// PlatformID returns the platform ID for the host h.
func PlatformID(h *Host) (platformID string) {
	p := h.GetString(h.Join("osinfo", "platform_id"))
	s := strings.Split(p, ":")
	if len(s) > 1 {
		platformID = s[1]
	}
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
		return []*Host{GetHost(h)}
	}
}

// OrList will return the receiver unless it is nil, or the list of all hosts
// passed as args. If no args are given and the receiver is nil then all hosts
// are returned.
func (h *Host) OrList(hosts ...*Host) []*Host {
	switch h {
	case nil:
		if len(hosts) == 0 {
			return AllHosts()
		}
		return hosts
	case ALL:
		return AllHosts()
	case UNKNOWN:
		return []*Host{}
	default:
		return []*Host{h}
	}
}

// PathTo builds an absolute path based on the Geneos root of the host
// h (using the executable name as the key) and the parts passed as
// arguments. Each part can be a pointer to a geneos.Component, in which
// case the component name or the parent component name is used, or any
// other type is passed to fmt.Sprint to be stringified. The path is
// returned from path.Join
//
// If calling this against the "packages" directory remember to use
// ct.String() to not deference the parent type, which is done if a part
// is a *Component
func (h *Host) PathTo(parts ...any) string {
	if h == nil {
		h = LOCAL
	}

	strParts := []string{h.GetString(execname)}

	for _, p := range parts {
		switch s := p.(type) {
		case *Component:
			if s.ParentType != nil {
				strParts = append(strParts, s.ParentType.Name)
			} else {
				strParts = append(strParts, s.Name)
			}
		case []any:
			for _, t := range s {
				strParts = append(strParts, fmt.Sprint(t))
			}
		case string:
			strParts = append(strParts, s)
		case fmt.Stringer:
			strParts = append(strParts, s.String())
		default:
			strParts = append(strParts, fmt.Sprint(s))
		}
	}

	return path.Join(strParts...)
}

// FullName returns name with the host h label appended if there is no
// existing host label in the form `instance@host`. Any existing label
// is not checked or changed.
func (h *Host) FullName(name string) string {
	if strings.Contains(name, "@") {
		return name
	}
	return name + "@" + h.String()
}

// AllHosts returns a slice of all hosts, including LOCAL
func AllHosts() (hs []*Host) {
	hs = []*Host{LOCAL}
	hs = append(hs, RemoteHosts(false)...)
	return
}

// RemoteHosts returns a slice of all valid (loaded and reachable)
// remote hosts sorted by name
func RemoteHosts(includeHidden bool) (hs []*Host) {
	hs = []*Host{}

	hosts.Range(func(k, v interface{}) bool {
		h := GetHost(k.(string))
		if h.Exists() && (includeHidden || !h.hidden) {
			hs = append(hs, h)
		}
		return true
	})
	slices.SortFunc(hs, func(i, j *Host) int { return strings.Compare(i.String(), j.String()) })
	return
}

// Hidden returns true is the host is marked hidden
func (h *Host) Hidden() bool {
	return h.hidden
}

// LoadHostConfig loads configuration entries from the host
// configuration file.
func LoadHostConfig() {
	var err error
	var confFile string
	userConfDir, err := config.UserConfigDir()
	if err != nil {
		log.Error().Err(err).Msg("user lookup failed, skipping user config directory")
	} else {
		confFile = config.PromoteFile(host.Localhost, path.Join(userConfDir, execname), path.Join(userConfDir, OldUserHostFile))
	}
	// note that SetAppName only matters when PromoteFile returns an empty path
	h, err := config.Load("hosts",
		config.SetAppName(execname),
		config.SetConfigFile(confFile),
		config.UseDefaults(false),
		config.IgnoreWorkingDir(),
	)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	hosts = sync.Map{}

	for _, hostval := range h.GetStringMap("hosts") {
		v := config.New()
		switch m := hostval.(type) {
		case map[string]interface{}:
			v.MergeConfigMap(m)
		default:
			log.Debug().Msgf("hosts value not a map[string]interface{} but a %T", hostval)
			continue
		}

		r := host.NewSSHRemote(v.GetString("name"),
			host.Username(v.GetString("username")), // username is the login name for the remote host
			host.Hostname(v.GetString("hostname")),
			host.Port(uint16(v.GetInt("port"))),
			host.Password(v.GetPassword("password").Enclave),
		)
		hosts.Store(v.GetString("name"), &Host{r, v, v.GetBool("hidden"), true})
	}
}

// SaveHostConfig writes the current hosts to the users hosts configuration file
func SaveHostConfig() error {
	n := config.New()

	hosts.Range(func(k, v interface{}) bool {
		name := k.(string)
		switch v := v.(type) {
		case *Host:
			n.Set(n.Join("hosts", name), v.AllSettings())
		}
		return true
	})

	return n.Save("hosts", config.SetAppName(execname))
}
