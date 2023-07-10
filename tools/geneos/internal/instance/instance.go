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

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// The Instance type is the common data shared by all instances
type Instance struct {
	geneos.Instance `json:"-"`
	Conf            *config.Config    `json:"-"`
	InstanceHost    *geneos.Host      `json:"-"`
	Component       *geneos.Component `json:"-"`
	ConfigLoaded    bool              `json:"-"`
}

// IsA returns true if instance c has a type that is component of the
// type name. If name is not a known component type then false is
// returned without checking the instance.
func IsA(c geneos.Instance, name string) bool {
	ct := geneos.ParseComponent(name)
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

// ReservedName returns true if name is a reserved word. Reserved names
// are checked against all the values registered by components at
// start-up.
func ReservedName(name string) (ok bool) {
	log.Debug().Msgf("checking %q", name)
	if geneos.ParseComponent(name) != nil {
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

// ValidName returns true if name is considered a valid instance
// name. It is not checked against the list of reserved names.
//
// XXX used to consume instance names until parameters are then passed
// down
func ValidName(name string) (ok bool) {
	ok = validStringRE.MatchString(name)
	if !ok {
		log.Debug().Msgf("no rexexp match: %s", name)
	}
	return
}

// LogFilePath returns the full path to the log file for the instance.
func LogFilePath(c geneos.Instance) (logfile string) {
	logdir := path.Clean(c.Config().GetString("logdir"))
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

	if err = c.Host().Signal(pid, signal); err != nil {
		return
	}

	_, err = GetPID(c)
	return
}

// Get return an instance of component ct, and loads the config. It is
// an error if the config cannot be loaded.
func Get(ct *geneos.Component, name string) (c geneos.Instance, err error) {
	if ct == nil || name == "" {
		return nil, geneos.ErrInvalidArgs
	}

	c = ct.New(name)
	if c == nil {
		// if no instance is created, check why
		_, _, h := NameParts(name, geneos.LOCAL)
		if h == geneos.LOCAL && geneos.Root() == "" {
			err = geneos.ErrRootNotSet
			return
		}
		err = geneos.ErrInvalidArgs
		return
	}
	err = c.Load()
	return
}

// GetAll returns a slice of instances for a given component type on remote h
func GetAll(h *geneos.Host, ct *geneos.Component) (confs []geneos.Instance) {
	if ct == nil {
		for _, c := range geneos.RealComponents() {
			confs = append(confs, GetAll(h, c)...)
		}
		return
	}
	for _, name := range Names(h, ct) {
		i, err := Get(ct, name)
		if err != nil {
			continue
		}
		confs = append(confs, i)
	}

	return
}

// MatchAll constructs and returns a slice of instances that have a
// matching name
func MatchAll(h *geneos.Host, ct *geneos.Component, name string) (c []geneos.Instance) {
	_, local, r := NameParts(name, h)
	if !r.IsAvailable() {
		log.Debug().Err(host.ErrNotAvailable).Msgf("host %s", r)
		return
	}

	if h != geneos.ALL && r.String() != h.String() {
		return
	}

	if ct == nil {
		for _, ct := range geneos.RealComponents() {
			c = append(c, MatchAll(h, ct, name)...)
		}
		return
	}

	for _, name := range Names(r, ct) {
		// for case insensitive match change to EqualFold here
		_, ldir, _ := NameParts(name, geneos.ALL)
		if path.Base(ldir) == local {
			i, err := Get(ct, name)
			if err != nil {
				continue
			}
			c = append(c, i)
		}
	}

	return
}

// Match looks for exactly one matching instance across types and hosts
// returns Invalid Args if zero of more than 1 match
func Match(h *geneos.Host, ct *geneos.Component, name string) (c geneos.Instance, err error) {
	list := MatchAll(h, ct, name)
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

// MatchKeyValue returns a slice of instances where the instance
// configuration key matches the value given.
func MatchKeyValue(h *geneos.Host, ct *geneos.Component, key, value string) (confs []geneos.Instance) {
	confs = GetAll(h, ct)

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

// ForAll calls the supplied function for each matching instance. It
// sends any returned error on STDOUT and the only error returned is
// os.ErrNotExist if there are no matching instances.
func ForAll(ct *geneos.Component, hostname string, fn func(geneos.Instance, []string) error, args []string, params []string) (err error) {
	n := 0
	// if args is empty, get all matching instances. this allows internal
	// calls with an empty arg list without having to do the parseArgs()
	// dance
	h := geneos.GetHost(hostname)
	if h == nil {
		h = geneos.ALL
	}
	if len(args) == 0 {
		args = Names(h, ct)
	}

	allcs := []geneos.Instance{}

	for _, name := range args {
		cs := MatchAll(h, ct, name)
		if len(cs) == 0 {
			continue
		}
		n++
		allcs = append(allcs, cs...)
	}

	if n == 0 {
		return os.ErrNotExist
	}

	var wg sync.WaitGroup
	for _, c := range allcs {
		wg.Add(1)
		go func(c geneos.Instance) {
			defer wg.Done()
			if err = fn(c, params); err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, geneos.ErrNotSupported) {
				fmt.Printf("%s: %s\n", c, err)
			}
		}(c)
	}
	wg.Wait()

	return nil
}

// ForAllWithResults calls the function fn for each matching instance
// and gather the return values into a slice of interfaces for handling
// upstream. The slice is sorted by host, type and name. Errors are printed
// on STDOUT for each call and the only error returned ErrNotExist if
// there are no matches.
func ForAllWithResults(
	ct *geneos.Component,
	hostname string,
	fn func(geneos.Instance, []string) (interface{}, error),
	args []string,
	params []string,
) (results []interface{}, err error) {
	var instances []geneos.Instance

	n := 0
	// if args is empty, get all matching instances. this allows internal
	// calls with an empty arg list without having to do the parseArgs()
	// dance
	h := geneos.GetHost(hostname)
	if h == nil {
		h = geneos.ALL
	}
	if len(args) == 0 {
		args = Names(h, ct)
	}
	allcs := []geneos.Instance{}

	for _, name := range args {
		cs := MatchAll(h, ct, name)
		if len(cs) == 0 {
			continue
		}
		n++
		allcs = append(allcs, cs...)
	}

	var mutex sync.Mutex
	var wg sync.WaitGroup

	for _, c := range allcs {
		instances = make([]geneos.Instance, 0, len(allcs))
		wg.Add(1)
		go func(c geneos.Instance) {
			var res interface{}
			defer wg.Done()
			if res, err = fn(c, params); err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, geneos.ErrNotSupported) {
				fmt.Printf("%s: %s\n", c, err)
			}
			if res != nil {
				mutex.Lock()
				instances = append(instances, c)
				results = append(results, res)
				mutex.Unlock()
			}
		}(c)
	}
	wg.Wait()
	if n == 0 {
		return nil, os.ErrNotExist
	}

	sort.Sort(SortInstanceResults{Instances: instances, Results: results})
	return results, nil
}

// Names returns a slice of all instance names for a given component ct
// on host h. No checking is done to validate that the directory is a
// populated instance.
//
// To support the move to parent types we do a little more, looking for
// legacy locations in here
func Names(h *geneos.Host, ct *geneos.Component) (names []string) {
	var files []fs.DirEntry

	if h == nil {
		h = geneos.ALL
	}

	if h == geneos.ALL {
		for _, h := range geneos.AllHosts() {
			names = append(names, Names(h, ct)...)
		}
		return
	}

	if ct == nil {
		for _, ct := range geneos.RealComponents() {
			// ignore errors, we only care about any files found
			for _, dir := range ct.InstancesDirs(h) {
				// log.Debug().Msgf("ct, dirs: %s %s", ct, dir)
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

// NameParts returns the parts of an instance name given an instance
// name in the format [TYPE:]NAME[@HOST] and a default host, return a
// *geneos.Component for the TYPE if given, a string for the NAME and a
// *geneos.Host - the latter being either from the name or the default
// provided
func NameParts(in string, defaultHost *geneos.Host) (ct *geneos.Component, name string, h *geneos.Host) {
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
		ct = geneos.ParseComponent(parts[0])
		name = parts[1]
	}
	return
}

// BuildCmd gathers the path to the binary, arguments and any
// environment variables for an instance and returns an exec.Cmd, almost
// ready for execution. Callers will add more details such as working
// directories, user and group etc.
//
// If noDecode is set then any secure environment variables are not decoded,
// so can be used for display
func BuildCmd(c geneos.Instance, noDecode bool) (cmd *exec.Cmd, env []string, home string) {
	binary := PathOf(c, "program")

	args, env, home := c.Command()

	opts := strings.Fields(c.Config().GetString("options"))
	args = append(args, opts...)

	envs := c.Config().GetStringSlice("Env", config.NoDecode(noDecode))
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
		env = append(env, "LD_LIBRARY_PATH="+strings.Join(libs, ":"))
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

type OpenFiles struct {
	Path   string
	Stat   fs.FileInfo
	FD     string
	FDMode fs.FileMode
}

// Files returns a map of file descriptor (int) to file details
// (InstanceProcFiles) for all open, real, files for the process running
// as the instance. All paths that are not absolute paths are ignored.
// An empty map is returned if the process cannot be found.
func Files(c geneos.Instance) (openfiles map[int]OpenFiles) {
	pid, err := GetPID(c)
	if err != nil {
		return
	}

	file := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := c.Host().ReadDir(file)
	if err != nil {
		return
	}

	openfiles = make(map[int]OpenFiles, len(fds))

	for _, ent := range fds {
		fd := ent.Name()
		dest, err := c.Host().Readlink(path.Join(file, fd))
		if err != nil {
			continue
		}
		if !filepath.IsAbs(dest) {
			continue
		}
		n, _ := strconv.Atoi(fd)

		fdPath := path.Join(file, fd)
		fdMode, err := c.Host().Lstat(fdPath)
		if err != nil {
			continue
		}

		s, err := c.Host().Stat(dest)
		if err != nil {
			continue
		}

		openfiles[n] = OpenFiles{
			Path:   dest,
			Stat:   s,
			FD:     fdPath,
			FDMode: fdMode.Mode(),
		}

		log.Debug().Msgf("\tfd %s points to %q", fd, dest)
	}
	return
}
