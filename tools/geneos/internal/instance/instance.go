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
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"

	"github.com/rs/zerolog/log"
)

// The Instance type is the common data shared by all instance / component types
type Instance struct {
	geneos.Instance `json:"-"`
	L               *sync.RWMutex     `json:"-"`
	Conf            *config.Config    `json:"-"`
	InstanceHost    *host.Host        `json:"-"`
	Component       *geneos.Component `json:"-"`
	ConfigLoaded    bool              `json:"-"`
	Env             []string          `json:",omitempty"`
}

func DisplayName(c geneos.Instance) string {
	if c.Host() == host.LOCAL {
		return fmt.Sprintf("%s %q", c.Type(), c.Name())
	}
	return fmt.Sprintf("%s \"%s@%s\"", c.Type(), c.Name(), c.Host())
}

// separate reserved words and invalid syntax
func ReservedName(in string) (ok bool) {
	log.Debug().Msgf("checking %q", in)
	if geneos.ParseComponentName(in) != nil {
		log.Debug().Msg("matches a reserved word")
		return true
	}
	if config.GetString("reservednames") != "" {
		list := strings.Split(in, ",")
		for _, n := range list {
			if strings.EqualFold(in, strings.TrimSpace(n)) {
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

// return true while a string is considered a valid instance name
//
// used to consume instance names until parameters are then passed down
func ValidInstanceName(in string) (ok bool) {
	ok = validStringRE.MatchString(in)
	if !ok {
		log.Debug().Msgf("no rexexp match: %s", in)
	}
	return
}

// given a filename or path, prepend the instance home directory
// if not absolute, and clean
func Abs(c geneos.Instance, file string) (path string) {
	path = filepath.Clean(file)
	if filepath.IsAbs(path) {
		return
	}
	return utils.JoinSlash(c.Home(), path)
}

// given a pathlist (typically ':') seperated list of paths, remove all
// files and directories
func RemovePaths(c geneos.Instance, paths string) (err error) {
	list := filepath.SplitList(paths)
	for _, p := range list {
		// clean path, error on absolute or parent paths, like 'import'
		// walk globbed directories, remove everything
		p, err = host.CleanRelativePath(p)
		if err != nil {
			return fmt.Errorf("%s %w", p, err)
		}
		// glob here
		m, err := c.Host().Glob(utils.JoinSlash(c.Home(), p))
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

// logdir = LogD relative to Home or absolute
func LogFile(c geneos.Instance) (logfile string) {
	logd := filepath.Clean(c.Config().GetString("logdir"))
	switch {
	case logd == "":
		logfile = c.Home()
	case filepath.IsAbs(logd):
		logfile = logd
	default:
		logfile = utils.JoinSlash(c.Home(), logd)
	}
	logfile = utils.JoinSlash(logfile, c.Config().GetString("logfile"))
	return
}

func Signal(c geneos.Instance, signal syscall.Signal) (err error) {
	pid, err := GetPID(c)
	if err != nil {
		return os.ErrProcessDone
	}

	if c.Host() == host.LOCAL {
		proc, _ := os.FindProcess(pid)
		if err = proc.Signal(signal); err != nil && !errors.Is(err, syscall.EEXIST) {
			log.Warn().Msgf("%s FAILED to send a signal %d: %s", c, signal, err)
			return
		}
		log.Debug().Msgf("%s sent a signal %d", c, signal)
		return nil
	}

	rem, err := c.Host().Dial()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	sess, err := rem.NewSession()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	output, err := sess.CombinedOutput(fmt.Sprintf("kill -s %d %d", signal, pid))
	sess.Close()
	if err != nil {
		log.Warn().Msgf("%s FAILED to send signal %d: %s %q", c, signal, err, output)
		return
	}
	log.Debug().Msgf("%s sent a signal %d", c, signal)
	return nil
}

// return an instance of component ct, loads the config.
// it is an error if the config cannot be loaded
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

// return a slice of instances for a given component type
func GetAll(r *host.Host, ct *geneos.Component) (confs []geneos.Instance) {
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

// Looks for exactly one matching instance across types and hosts
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

// construct and return a slice of a/all component types that have
// a matching name
func MatchAll(ct *geneos.Component, name string) (c []geneos.Instance) {
	_, local, r := SplitName(name, host.ALL)
	if !r.Exists() {
		log.Debug().Msgf("host %s not loaded", r)
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
		_, ldir, _ := SplitName(name, host.ALL)
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
func MatchKeyValue(h *host.Host, ct *geneos.Component, key, value string) (confs []geneos.Instance) {
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

// get all used ports in config files on a specific remote
// this will not work for ports assigned in component config
// files, such as gateway setup or netprobe collection agent
//
// returns a map
func GetPorts(r *host.Host) (ports map[uint16]*geneos.Component) {
	if r == host.ALL {
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
func NextPort(r *host.Host, ct *geneos.Component) uint16 {
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

// Version returns the base package name and the version it links to. If
// base is not a link, then symlink is returned the same as base. If
// there are more than 10 levels of symlink then return symlink set to
// "loop-detected" and err set to  syscall.ELOOP to prevent infinite
// loops.
func Version(c geneos.Instance) (base string, symlink string, err error) {
	var st fs.FileInfo
	var i int

	basedir := c.Config().GetString("install")
	base = c.Config().GetString("version")
	symlink = base

	for i = 0; i < 10; i++ {
		basepath := utils.JoinSlash(basedir, symlink)
		st, err = c.Host().Lstat(basepath)
		if err != nil {
			symlink = "unknown"
			return
		}
		if st.Mode()&fs.ModeSymlink != 0 {
			symlink, err = c.Host().Readlink(basepath)
			if err != nil {
				symlink = "unknown"
				return
			}
		} else {
			break
		}
	}
	if i == 10 {
		err = syscall.ELOOP
		symlink = "loop-detected"
	}
	return
}

// ForAll calls the supplied function for each matching instance. It
// prints any returned error on STDOUT and the only error returned is
// os.ErrNotExist if there are no matching instances.
func ForAll(ct *geneos.Component, fn func(geneos.Instance, []string) error, args []string, params []string) (err error) {
	n := 0
	log.Debug().Msgf("args %v, params %v", args, params)
	// if args is empty, get all matching instances. this allows internal
	// calls with an empty arg list without having to do the parseArgs()
	// dance
	if len(args) == 0 {
		args = AllNames(host.ALL, ct)
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
func ForAllWithResults(ct *geneos.Component, fn func(geneos.Instance, []string) (interface{}, error), args []string, params []string) (results []interface{}, err error) {
	n := 0
	log.Debug().Msgf("args %v, params %v", args, params)
	// if args is empty, get all matching instances. this allows internal
	// calls with an empty arg list without having to do the parseArgs()
	// dance
	if len(args) == 0 {
		args = AllNames(host.ALL, ct)
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

// Return a slice of all instance names for a given component. No
// checking is done to validate that the directory is a populated
// instance.
func AllNames(h *host.Host, ct *geneos.Component) (names []string) {
	var files []fs.DirEntry

	if h == host.ALL {
		for _, r := range host.AllHosts() {
			names = append(names, AllNames(r, ct)...)
		}
		log.Debug().Msgf("names: %s", names)
		return
	}

	if ct == nil {
		for _, t := range geneos.RealComponents() {
			// ignore errors, we only care about any files found
			d, _ := h.ReadDir(t.InstancesDir(h))
			files = append(files, d...)
		}
	} else {
		// ignore errors, we only care about any files found
		files, _ = h.ReadDir(ct.InstancesDir(h))
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

// given an instance name in the format [TYPE:]NAME[@HOST] and a default
// host, return a *geneos.Component for the TYPE if given, a string
// for the NAME and a *host.Host - the latter being either from the name
// or the default provided
func SplitName(in string, defaultHost *host.Host) (ct *geneos.Component, name string, h *host.Host) {
	h = defaultHost
	parts := strings.SplitN(in, "@", 2)
	name = parts[0]
	if len(parts) > 1 {
		h = host.Get(parts[1])
	}
	parts = strings.SplitN(name, ":", 2)
	if len(parts) > 1 {
		ct = geneos.ParseComponentName(parts[0])
		name = parts[1]
	}
	return
}

// buildCmd gathers the path to the binary, arguments and any environment variables
// for an instance and returns an exec.Cmd, almost ready for execution. Callers
// will add more details such as working directories, user and group etc.
func BuildCmd(c geneos.Instance) (cmd *exec.Cmd, env []string) {
	binary := Filepath(c, "program")

	args, env := c.Command()

	opts := strings.Fields(c.Config().GetString("options"))
	args = append(args, opts...)
	// XXX find common envs - JAVA_HOME etc.
	env = append(env, c.Config().GetStringSlice("Env")...)
	env = append(env, "LD_LIBRARY_PATH="+c.Config().GetString("libpaths"))
	cmd = exec.Command(binary, args...)

	return
}

func IsDisabled(c geneos.Instance) bool {
	d := ComponentFilepath(c, geneos.DisableExtension)
	if f, err := c.Host().Stat(d); err == nil && f.Mode().IsRegular() {
		return true
	}
	return false
}

func IsProtected(c geneos.Instance) bool {
	return c.Config().GetBool("protected")
}
