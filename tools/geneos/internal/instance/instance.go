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

package instance

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"

	"github.com/rs/zerolog/log"
)

// The Instance type is the common data shared by all instance / component types
type Instance struct {
	geneos.Instance `json:"-"`
	L               *sync.RWMutex     `json:"-"`
	Conf            *config.Config    `json:"-"`
	InstanceHost    *geneos.Host      `json:"-"`
	Component       *geneos.Component `json:"-"`
	ConfigLoaded    bool              `json:"-"`
	Env             []string          `json:",omitempty"`
}

// IsA returns true if instance c has a type that is component if the type name.
func IsA(c geneos.Instance, name string) bool {
	ct := geneos.FindComponent(name)
	if ct == nil {
		return false
	}
	return ct.IsA(c.Type().String())
}

// DisplayName returns the type, name and non-local host as a string
// suitable for display.
func DisplayName(c geneos.Instance) string {
	if c.Host().IsLocal() {
		return fmt.Sprintf("%s %q", c.Type(), c.Name())
	}
	return fmt.Sprintf("%s \"%s@%s\"", c.Type(), c.Name(), c.Host())
}

// ReservedName returns true if in name a reserved word. Reserved names
// are checked against all the values registered by components at
// start-up.
func ReservedName(name string) (ok bool) {
	log.Debug().Msgf("checking %q", name)
	if geneos.FindComponent(name) != nil {
		log.Debug().Msg("matches a reserved word")
		return true
	}
	if config.GetString("reservednames") != "" {
		list := strings.Split(name, ",")
		for _, n := range list {
			if strings.EqualFold(name, strings.TrimSpace(n)) {
				log.Debug().Msg("matches a user defined reserved name")
				return true
			}
		}
	}
	return
}

// spaces are valid - dumb, but valid - for now. If the name starts with
// number then the next character cannot be a number or '.' to help
// distinguish from versions
var validStringRE = regexp.MustCompile(`^\w[\w-]?[:@\.\w -]*$`)

// ValidInstanceName returns true if name is considered a valid instance
// name. It is not checked against the list of reserved names.
//
// XXX used to consume instance names until parameters are then passed
// down
func ValidInstanceName(name string) (ok bool) {
	ok = validStringRE.MatchString(name)
	if !ok {
		log.Debug().Msgf("no rexexp match: %s", name)
	}
	return
}

// Abs returns an absolute path to file prepended with the instance
// working directory if file is not already an absolute path.
func Abs(c geneos.Instance, file string) (result string) {
	file = filepath.Clean(file)
	if filepath.IsAbs(file) {
		return
	}
	return path.Join(c.Home(), file)
}

// RemovePaths removes all files and directories in paths, each file or directory is separated by ListSeperator
func RemovePaths(c geneos.Instance, paths string) (err error) {
	list := filepath.SplitList(paths)
	for _, p := range list {
		// clean path, error on absolute or parent paths, like 'import'
		// walk globbed directories, remove everything
		p, err = geneos.CleanRelativePath(p)
		if err != nil {
			return fmt.Errorf("%s %w", p, err)
		}
		// glob here
		m, err := c.Host().Glob(path.Join(c.Home(), p))
		if err != nil {
			return err
		}
		for _, f := range m {
			if err = c.Host().RemoveAll(f); err != nil {
				log.Error().Err(err).Msg("")
				continue
			}
			fmt.Printf("removed %s\n", c.Host().Path(f))
		}
	}
	return
}

// LogFile returns the fulle path to the log file for the instance.
//
// XXX logdir = LogD relative to Home or absolute
func LogFile(c geneos.Instance) (logfile string) {
	logdir := filepath.Clean(c.Config().GetString("logdir"))
	switch {
	case logdir == "":
		logfile = c.Home()
	case filepath.IsAbs(logdir):
		logfile = logdir
	default:
		logfile = path.Join(c.Home(), logdir)
	}
	logfile = path.Join(logfile, c.Config().GetString("logfile"))
	return
}

// Signal sends the signal to the instance
func Signal(c geneos.Instance, signal syscall.Signal) (err error) {
	pid, err := GetPID(c)
	if err != nil {
		return os.ErrProcessDone
	}

	return c.Host().Signal(pid, signal)
}

// Get return an instance of component ct, and loads the config. It is
// an error if the config cannot be loaded.
func Get(ct *geneos.Component, name string) (c geneos.Instance, err error) {
	if ct == nil {
		return nil, geneos.ErrInvalidArgs
	}

	c = ct.New(name)
	if c == nil {
		return nil, geneos.ErrInvalidArgs
	}
	err = c.Load()
	return
}

// GetAll returns a slice of instances for a given component type on remote r
func GetAll(r *geneos.Host, ct *geneos.Component) (confs []geneos.Instance) {
	if ct == nil {
		for _, c := range geneos.RealComponents() {
			confs = append(confs, GetAll(r, c)...)
		}
		return
	}
	for _, name := range AllNames(r, ct) {
		i, err := Get(ct, name)
		if err != nil {
			continue
		}
		confs = append(confs, i)
	}

	return
}

// Match looks for exactly one matching instance across types and hosts
// returns Invalid Args if zero of more than 1 match
func Match(ct *geneos.Component, name string) (c geneos.Instance, err error) {
	list := MatchAll(ct, name)
	if len(list) == 0 {
		err = os.ErrNotExist
		return
	}
	if len(list) == 1 {
		c = list[0]
		return
	}
	err = geneos.ErrInvalidArgs
	return
}

// MatchAll constructs and returns a slice of instances that have a
// matching name
func MatchAll(ct *geneos.Component, name string) (c []geneos.Instance) {
	_, local, r := SplitName(name, geneos.ALL)
	if !r.IsAvailable() {
		log.Debug().Err(host.ErrNotAvailable).Msgf("host %s", r)
		return
	}

	if ct == nil {
		for _, t := range geneos.RealComponents() {
			c = append(c, MatchAll(t, name)...)
		}
		return
	}

	for _, name := range AllNames(r, ct) {
		// for case insensitive match change to EqualFold here
		_, ldir, _ := SplitName(name, geneos.ALL)
		if filepath.Base(ldir) == local {
			i, err := Get(ct, name)
			if err != nil {
				log.Error().Err(err).Msg("")
				continue
			}
			c = append(c, i)
		}
	}

	return
}

// MatchKeyValue returns a slice of instances where the instance
// configuration key matches the value given.
func MatchKeyValue(h *geneos.Host, ct *geneos.Component, key, value string) (confs []geneos.Instance) {
	if ct == nil {
		for _, c := range geneos.RealComponents() {
			confs = append(confs, MatchKeyValue(h, c, key, value)...)
		}
		return
	}

	for _, name := range AllNames(h, ct) {
		i, err := Get(ct, name)
		if err != nil {
			continue
		}
		confs = append(confs, i)
	}

	// filter in place
	n := 0
	for _, c := range confs {
		if c.Config().GetString(key) == value {
			confs[n] = c
			n++
		}
	}
	confs = confs[:n]

	return
}

// GetPorts gets all used ports in config files on a specific remote
// this will not work for ports assigned in component config files, such
// as gateway setup or netprobe collection agent
//
// returns a map
func GetPorts(r *geneos.Host) (ports map[uint16]*geneos.Component) {
	if r == geneos.ALL {
		log.Fatal().Msg("getports() call with all hosts")
	}
	ports = make(map[uint16]*geneos.Component)
	for _, c := range GetAll(r, nil) {
		if !c.Loaded() {
			log.Error().Msgf("cannot load configuration for %s", c)
			continue
		}
		if port := c.Config().GetInt("port"); port != 0 {
			ports[uint16(port)] = c.Type()
		}
	}
	return
}

// syntax of ranges of ints:
// x,y,a-b,c..d m n o-p
// also open ended A,N-,B
// command or space seperated?
// - or .. = inclusive range
//
// how to represent
// split, for range, check min-max -> max > min
// repeats ignored
// special ports? - nah
//

// given a range, find the first unused port
//
// range is comma or two-dot separated list of
// single number, e.g. "7036"
// min-max inclusive range, e.g. "7036-8036"
// start- open ended range, e.g. "7041-"
//
// some limits based on https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers
//
// not concurrency safe at this time
func NextPort(r *geneos.Host, ct *geneos.Component) uint16 {
	from := config.GetString(ct.PortRange)
	used := GetPorts(r)
	ps := strings.Split(from, ",")
	for _, p := range ps {
		// split on comma or ".."
		m := strings.SplitN(p, "-", 2)
		if len(m) == 1 {
			m = strings.SplitN(p, "..", 2)
		}

		if len(m) > 1 {
			var min uint16
			mn, err := strconv.Atoi(m[0])
			if err != nil {
				continue
			}
			if mn < 0 || mn > 65534 {
				min = 65535
			} else {
				min = uint16(mn)
			}
			if m[1] == "" {
				m[1] = "49151"
			}
			max, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			if int(min) >= max {
				continue
			}
			for i := min; int(i) <= max; i++ {
				if _, ok := used[i]; !ok {
					// found an unused port
					return i
				}
			}
		} else {
			var p1 uint16
			p, err := strconv.Atoi(m[0])
			if err != nil {
				continue
			}
			if p < 0 || p > 65534 {
				p1 = 65535
			} else {
				p1 = uint16(p)
			}
			if _, ok := used[p1]; !ok {
				return p1
			}
		}
	}
	return 0
}

// BaseVersion returns the absolute path of the base package directory
// for the instance c. No longer references the instance "install" parameter.
func BaseVersion(c geneos.Instance) (dir string) {
	return c.Host().Filepath("packages", c.Type().String(), c.Config().GetString("version"))
}

// Version returns the base package name and the underlying package
// version for the instance c. If base is not a link, then base is also
// returned as the symlink. If there are more than 10 levels of symlink
// then return symlink set to "loop-detected" and err set to
// syscall.ELOOP to prevent infinite loops.
func Version(c geneos.Instance) (base string, version string, err error) {
	cf := c.Config()
	base = cf.GetString("version")
	pkgtype := cf.GetString("pkgtype")
	ct := c.Type()
	if pkgtype != "" {
		ct = geneos.FindComponent(pkgtype)
	}
	version, err = geneos.CurrentVersion(c.Host(), ct, cf.GetString("version"))
	return
}

// AtLeastVersion returns true if the installed version for instance c
// is version or greater. If the version of the instance is somehow
// unparseable then this returns false.
func AtLeastVersion(c geneos.Instance, version string) bool {
	_, iv, err := Version(c)
	if err != nil {
		return false
	}
	return geneos.CompareVersion(iv, version) >= 0
}

// ForAll calls the supplied function for each matching instance. It
// prints any returned error on STDOUT and the only error returned is
// os.ErrNotExist if there are no matching instances.
func ForAll(ct *geneos.Component, hostname string, fn func(geneos.Instance, []string) error, args []string, params []string) (err error) {
	n := 0
	log.Debug().Msgf("args %v, params %v", args, params)
	// if args is empty, get all matching instances. this allows internal
	// calls with an empty arg list without having to do the parseArgs()
	// dance
	h := geneos.GetHost(hostname)
	if h == nil {
		h = geneos.ALL
	}
	if len(args) == 0 {
		args = AllNames(h, ct)
	}
	for _, name := range args {
		cs := MatchAll(ct, name)
		if len(cs) == 0 {
			log.Debug().Msgf("no match for %s", name)
			continue
		}
		n++
		for _, c := range cs {
			if err = fn(c, params); err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, geneos.ErrNotSupported) {
				fmt.Printf("%s: %s\n", c, err)
			}
		}
	}
	if n == 0 {
		return os.ErrNotExist
	}
	return nil
}

// ForAllWithResults calls the given function for each matching instance
// and gather the return values into a slice of interfaces for handling
// upstream. Errors are printed on STDOUT for each call and the only
// error returned ErrNotExist if there are no matches.
func ForAllWithResults(ct *geneos.Component, hostname string, fn func(geneos.Instance, []string) (interface{}, error), args []string, params []string) (results []interface{}, err error) {
	n := 0
	log.Debug().Msgf("args %v, params %v", args, params)
	// if args is empty, get all matching instances. this allows internal
	// calls with an empty arg list without having to do the parseArgs()
	// dance
	h := geneos.GetHost(hostname)
	if h == nil {
		h = geneos.ALL
	}
	if len(args) == 0 {
		args = AllNames(h, ct)
	}
	for _, name := range args {
		var res interface{}
		cs := MatchAll(ct, name)
		if len(cs) == 0 {
			log.Debug().Msgf("no match for %s", name)
			continue
		}
		n++
		for _, c := range cs {
			if res, err = fn(c, params); err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, geneos.ErrNotSupported) {
				fmt.Printf("%s: %s\n", c, err)
			}
			if res != nil {
				results = append(results, res)
			}
		}
	}
	if n == 0 {
		return nil, os.ErrNotExist
	}
	return results, nil
}

// ParentDirectory returns the first directory that contains the
// instance from:
//
//   - The one configured for the instance factory and accessed via Home()
//   - In the default component instances directory (component.InstanceDir)
//   - If the instance's component type has a parent component then in the
//     legacy instances directory
//
// The function has to accept an interface as it is called from inside
// the factory methods for each component type
func ParentDirectory(i interface{}) (dir string) {
	c, ok := i.(geneos.Instance)
	if !ok {
		log.Debug().Msg("i is not a geneos instance")
		return ""
	}
	h := c.Host()

	// first, does the configured home exist as a dir?
	if c.Home() != "" {
		dir = filepath.Dir(c.Home())
		// but check the configured home, not the parent
		if d, err := h.Stat(c.Home()); err == nil && d.IsDir() {
			log.Debug().Msg("default home, as defined")
			return
		}
	}

	// second, does the instance exist in the default instances dir?
	dir = c.Type().InstancesDir(h)
	if dir != "" {
		if d, err := h.Stat(dir); err == nil && d.IsDir() {
			log.Debug().Msg("instanceDir home selected")
			return
		}
	}

	// third, look in any "legacy" location, but only if parent type is
	// non nil
	if c.Type().ParentType != nil {
		dir = filepath.Join(h.Filepath(c.Type(), c.Type().String()+"s"))
		if dir != "" {
			if d, err := h.Stat(dir); err == nil && d.IsDir() {
				log.Debug().Msgf("new home, from legacy %s", dir)
				return
			}
		}
	}

	log.Debug().Msgf("default %s", dir)
	return dir
}

// AllNames returns a slice of all instance names for a given component.
// No checking is done to validate that the directory is a populated
// instance.
//
// To support the move to parent types we do a little more, looking for
// legacy locations in here
func AllNames(h *geneos.Host, ct *geneos.Component) (names []string) {
	var files []fs.DirEntry

	if h == nil {
		h = geneos.ALL
	}

	if h == geneos.ALL {
		for _, r := range geneos.AllHosts() {
			names = append(names, AllNames(r, ct)...)
		}
		log.Debug().Msgf("names: %s", names)
		return
	}

	if ct == nil {
		for _, ct := range geneos.RealComponents() {
			// ignore errors, we only care about any files found
			for _, dir := range ct.InstancesDirs(h) {
				log.Debug().Msgf("ct, dirs: %s %s", ct, dir)
				d, _ := h.ReadDir(dir)
				files = append(files, d...)
			}
		}
	} else {
		// ignore errors, we only care about any files found
		for _, dir := range ct.InstancesDirs(h) {
			d, _ := h.ReadDir(dir)
			files = append(files, d...)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})
	for i, file := range files {
		// skip for values with the same name as previous
		if i > 0 && i < len(files) && file.Name() == files[i-1].Name() {
			continue
		}
		if file.IsDir() {
			names = append(names, file.Name()+"@"+h.String())
		}
	}
	return
}

// SplitName returns the parts of an instance name given an instance
// name in the format [TYPE:]NAME[@HOST] and a default host, return a
// *geneos.Component for the TYPE if given, a string for the NAME and a
// *geneos.Host - the latter being either from the name or the default
// provided
func SplitName(in string, defaultHost *geneos.Host) (ct *geneos.Component, name string, h *geneos.Host) {
	if defaultHost == nil {
		h = geneos.ALL
	} else {
		h = defaultHost
	}
	parts := strings.SplitN(in, "@", 2)
	name = parts[0]
	if len(parts) > 1 {
		h = geneos.GetHost(parts[1])
	}
	parts = strings.SplitN(name, ":", 2)
	if len(parts) > 1 {
		ct = geneos.FindComponent(parts[0])
		name = parts[1]
	}
	return
}

// BuildCmd gathers the path to the binary, arguments and any environment variables
// for an instance and returns an exec.Cmd, almost ready for execution. Callers
// will add more details such as working directories, user and group etc.
func BuildCmd(c geneos.Instance) (cmd *exec.Cmd, env []string, home string) {
	binary := Filepath(c, "program")

	args, env, home := c.Command()

	opts := strings.Fields(c.Config().GetString("options"))
	args = append(args, opts...)

	envs := c.Config().GetStringSlice("Env")
	libs := []string{}
	if c.Config().GetString("libpaths") != "" {
		libs = append(libs, c.Config().GetString("libpaths"))
	}

	for _, e := range envs {
		switch {
		case strings.HasPrefix(e, "LD_LIBRARY_PATH="):
			libs = append(libs, strings.TrimPrefix(e, "LD_LIBRARY_PATH="))
		default:
			env = append(env, e)
		}
	}
	if len(libs) > 0 {
		env = append(env, "LD_LIBRARY_PATH="+strings.Join(libs, string(filepath.ListSeparator)))
	}
	cmd = exec.Command(binary, args...)

	return
}

// Disable the instance c. Does not try to stop a running instance and
// returns an error if it is running.
func Disable(c geneos.Instance) (err error) {
	if IsRunning(c) {
		return fmt.Errorf("instance %s running", c)
	}

	disablePath := ComponentFilepath(c, geneos.DisableExtension)

	h := c.Host()

	f, err := h.Create(disablePath, 0664)
	if err != nil {
		return err
	}
	f.Close()
	return
}

// Enable removes the disabled flag, if any,m from instance c.
func Enable(c geneos.Instance) (err error) {
	disableFile := ComponentFilepath(c, geneos.DisableExtension)
	if _, err = c.Host().Stat(disableFile); err != nil {
		return nil
	}
	return c.Host().Remove(disableFile)
}

// IsDisabled returns true if the instance c is disabled.
func IsDisabled(c geneos.Instance) bool {
	d := ComponentFilepath(c, geneos.DisableExtension)
	if f, err := c.Host().Stat(d); err == nil && f.Mode().IsRegular() {
		return true
	}
	return false
}

// IsProtected returns true if instance c is marked protected
func IsProtected(c geneos.Instance) bool {
	return c.Config().GetBool("protected")
}

// IsRunning returns true if the instance is running
func IsRunning(c geneos.Instance) bool {
	_, err := GetPID(c)
	return err != os.ErrProcessDone
}

// IsAutoStart returns true is the instance is set to autostart
func IsAutoStart(c geneos.Instance) bool {
	return c.Config().GetBool("autostart")
}

// SharedPath returns the full path a directory or file in the instances
// component type shared directory joined to any parts subs - the last
// element can be a filename. If the instance is not loaded then "." is
// returned for the current directory.
func SharedPath(c geneos.Instance, subs ...interface{}) string {
	if !c.Loaded() {
		return "."
	}
	return c.Type().SharedPath(c.Host(), subs...)
}
