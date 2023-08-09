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
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

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
	ConfigLoaded    time.Time         `json:"-"`
}

// IsA returns true if instance c has a type that is component of the
// type name. If name is not a known component type then false is
// returned without checking the instance.
func IsA(i geneos.Instance, name string) bool {
	ct := geneos.ParseComponent(name)
	if ct == nil {
		return false
	}
	return ct.IsA(i.Type().String())
}

// DisplayName returns the type, name and non-local host as a string
// suitable for display.
func DisplayName(i geneos.Instance) string {
	if i.Host().IsLocal() {
		return fmt.Sprintf("%s %q", i.Type(), i.Name())
	}
	return fmt.Sprintf("%s \"%s@%s\"", i.Type(), i.Name(), i.Host())
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
	if reserved := config.GetString("reservednames"); reserved != "" {
		list := strings.Split(reserved, ",")
		for _, n := range list {
			if strings.EqualFold(name, strings.TrimSpace(n)) {
				log.Debug().Msgf("%s matches a user defined reserved name %s", name, n)
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
func LogFilePath(i geneos.Instance) (logfile string) {
	logdir := path.Clean(i.Config().GetString("logdir"))
	switch {
	case logdir == "":
		logfile = i.Home()
	case filepath.IsAbs(logdir):
		logfile = logdir
	default:
		logfile = path.Join(i.Home(), logdir)
	}
	logfile = path.Join(logfile, i.Config().GetString("logfile"))
	return
}

// Signal sends the signal to the instance
func Signal(i geneos.Instance, signal syscall.Signal) (err error) {
	pid, err := GetPID(i)
	if err != nil {
		return os.ErrProcessDone
	}

	if err = i.Host().Signal(pid, signal); err != nil {
		return
	}

	_, err = GetPID(i)
	return
}

// Get return an instance of component ct, and loads the config. It is
// an error if the config cannot be loaded.
func Get(ct *geneos.Component, name string) (i geneos.Instance, err error) {
	if ct == nil || name == "" {
		return nil, geneos.ErrInvalidArgs
	}

	i = ct.New(name)
	if i == nil {
		// if no instance is created, check why
		_, _, h := SplitName(name, geneos.LOCAL)
		if h == geneos.LOCAL && geneos.Root() == "" {
			err = geneos.ErrRootNotSet
			return
		}
		err = geneos.ErrInvalidArgs
		return
	}
	err = i.Load()
	return
}

// GetAll returns a slice of instances for a given component type on remote h
func GetAll(h *geneos.Host, ct *geneos.Component) (instances []geneos.Instance) {
	if ct == nil {
		for _, c := range geneos.RealComponents() {
			instances = append(instances, GetAll(h, c)...)
		}
		return
	}
	for _, name := range Names(h, ct) {
		i, err := Get(ct, name)
		if err != nil {
			continue
		}
		instances = append(instances, i)
	}

	return
}

// ByName looks for exactly one matching instance across types and hosts
// returns Invalid Args if zero of more than 1 match
func ByName(h *geneos.Host, ct *geneos.Component, name string) (i geneos.Instance, err error) {
	list := ByNameAll(h, ct, name)
	if len(list) == 0 {
		err = os.ErrNotExist
		return
	}
	if len(list) == 1 {
		i = list[0]
		return
	}
	err = geneos.ErrInvalidArgs
	return
}

// ByNames returns a slice of instances that match any of the names
// given, using the host h as a validation check against names with a
// host qualification
func ByNames(h *geneos.Host, ct *geneos.Component, names ...string) (instances []geneos.Instance, err error) {
	n := 0
	// if args is empty, get all matching instances. this allows internal
	// calls with an empty arg list without having to do the parseArgs()
	// dance
	// h := geneos.GetHost(hostname)
	if h == nil {
		h = geneos.ALL
	}

	if len(names) == 0 {
		instances = GetAll(h, ct)
	} else {
		for _, name := range names {
			cs := ByNameAll(h, ct, name)
			if len(cs) == 0 {
				continue
			}
			n++
			instances = append(instances, cs...)
		}
		if n == 0 {
			return nil, os.ErrNotExist
		}
	}
	return
}

// ByNameAll constructs and returns a slice of instances that have a
// matching name. Host h is used to validate the host portion of the
// full name of the instance, if given.
func ByNameAll(h *geneos.Host, ct *geneos.Component, name string) (instances []geneos.Instance) {
	_, local, r := SplitName(name, h)
	if !r.IsAvailable() {
		log.Debug().Err(host.ErrNotAvailable).Msgf("host %s", r)
		return
	}

	if h != geneos.ALL && r.String() != h.String() {
		return
	}

	if ct == nil {
		for _, ct := range geneos.RealComponents() {
			instances = append(instances, ByNameAll(h, ct, name)...)
		}
		return
	}

	for _, name := range Names(r, ct) {
		_, ldir, _ := SplitName(name, geneos.ALL)
		if path.Base(ldir) == local {
			if i, err := Get(ct, name); err == nil {
				instances = append(instances, i)
			}
		}
	}

	return
}

// ByKeyValue returns a slice of instances where the instance
// configuration key matches the value given.
func ByKeyValue(h *geneos.Host, ct *geneos.Component, key, value string) (confs []geneos.Instance) {
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

// Do calls function f for each matching instance and gathers the return
// values into a slice of Response for handling by the caller. The
// functions are executed in goroutines and must be concurrency safe.
//
// The values are passed to each function called and must not be changes
// by the called function. The called function should validate and cast
// values for use.
//
// Do calls ByNames() to resolve the names given to a list of matching
// instances on host h (which can be geneos.ALL to look on all hosts)
// and for type ct, which can be nil to look across all component types.
func Do(h *geneos.Host, ct *geneos.Component, names []string, f func(geneos.Instance, ...any) *Response, values ...any) (responses Responses) {
	var wg sync.WaitGroup

	instances, err := ByNames(h, ct, names...)
	if err != nil {
		return
	}

	ch := make(chan *Response, len(instances))
	for _, c := range instances {
		wg.Add(1)
		go func(i geneos.Instance) {
			defer wg.Done()

			resp := f(i, values...)
			resp.Finish = time.Now()
			ch <- resp
		}(c)
	}
	wg.Wait()
	close(ch)

	for resp := range ch {
		responses = append(responses, resp)
	}

	sort.Sort(responses)
	return
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
			for _, dir := range ct.InstancesDir(h) {
				d, _ := h.ReadDir(dir)
				files = append(files, d...)
			}
		}
	} else {
		// ignore errors, we only care about any files found
		for _, dir := range ct.InstancesDir(h) {
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
		ct = geneos.ParseComponent(parts[0])
		name = parts[1]
	}
	return
}

// Disable the instance c. Does not try to stop a running instance and
// returns an error if it is running.
func Disable(i geneos.Instance) (err error) {
	if IsRunning(i) {
		return fmt.Errorf("instance %s running", i)
	}

	disablePath := ComponentFilepath(i, geneos.DisableExtension)

	h := i.Host()

	f, err := h.Create(disablePath, 0664)
	if err != nil {
		return err
	}
	f.Close()
	return
}

// Enable removes the disabled flag, if any,m from instance c.
func Enable(i geneos.Instance) (err error) {
	disableFile := ComponentFilepath(i, geneos.DisableExtension)
	if _, err = i.Host().Stat(disableFile); err != nil {
		return nil
	}
	return i.Host().Remove(disableFile)
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
func Files(i geneos.Instance) (openfiles map[int]OpenFiles) {
	pid, err := GetPID(i)
	if err != nil {
		return
	}

	file := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := i.Host().ReadDir(file)
	if err != nil {
		return
	}

	openfiles = make(map[int]OpenFiles, len(fds))

	for _, ent := range fds {
		fd := ent.Name()
		dest, err := i.Host().Readlink(path.Join(file, fd))
		if err != nil {
			continue
		}
		if !filepath.IsAbs(dest) {
			continue
		}
		n, _ := strconv.Atoi(fd)

		fdPath := path.Join(file, fd)
		fdMode, err := i.Host().Lstat(fdPath)
		if err != nil {
			continue
		}

		s, err := i.Host().Stat(dest)
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
