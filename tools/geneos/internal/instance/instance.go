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

package instance

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
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

type ProcessFDs struct {
	PID   int
	FD    int
	Path  string
	Lstat fs.FileInfo
	Stat  fs.FileInfo
	Conn  *SocketConnection
}

// ProcessStats is an example of a structure to pass to
// instance.ProcessStatus, using a field number for `stat` and a line
// prefix for `status` tags. OpenFiles and OpenSockets are counts of
// their respective names.
type ProcessStats struct {
	Pid         int64         `stat:"0"`
	Utime       time.Duration `stat:"13"`
	Stime       time.Duration `stat:"14"`
	CUtime      time.Duration `stat:"15"`
	CStime      time.Duration `stat:"16"`
	State       string        `status:"State"`
	Threads     int64         `status:"Threads"`
	VmRSS       int64         `status:"VmRSS"`
	VmHWM       int64         `status:"VmHWM"`
	RssAnon     int64         `status:"RssAnon"`
	RssFile     int64         `status:"RssFile"`
	RssShmem    int64         `status:"RssShmem"`
	OpenFiles   int64
	OpenSockets int64
}

// IsA returns true if instance i has a type that is component of one of
// names.
func IsA(i geneos.Instance, names ...string) bool {
	it := i.Type().String()
	for _, name := range names {
		if ct := geneos.ParseComponent(name); ct != nil && ct.IsA(it) {
			return true
		}
	}
	return false
}

// DisplayName returns the type, name and non-local host as a string
// suitable for display.
func DisplayName(i geneos.Instance) string {
	if i.Host().IsLocal() {
		return fmt.Sprintf("%s %q", i.Type(), i.Name())
	}
	return fmt.Sprintf("%s \"%s@%s\"", i.Type(), i.Name(), i.Host())
}

// ShortName returns the instance name with the "@HOST" suffix as a
// string
func ShortName(i geneos.Instance) string {
	return i.Name() + "@" + i.Host().String()
}

// IDString returns a string suitable as a unique rowname for Toolkit
// output and other places. For local instance it return `TYPE:NAME` and
// for remote instances it returns `TYPE:NAME@HOST`
func IDString(i geneos.Instance) string {
	if i.Host().IsLocal() {
		return i.Type().String() + ":" + i.Name()
	}
	return i.Type().String() + ":" + i.Name() + "@" + i.Host().String()
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

// LogFilePath returns the full path to the log file for the instance.
func LogFilePath(i geneos.Instance) (logfile string) {
	logdir := path.Clean(i.Config().GetString("logdir"))
	switch {
	case logdir == "":
		logfile = i.Home()
	case path.IsAbs(logdir):
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

// Do calls function f for each matching instance and gathers the return
// values into responses for handling by the caller. The functions are
// executed in goroutines and must be concurrency safe.
//
// The values are passed to each function called and must not be changes
// by the called function. The called function should validate and cast
// values for use.
//
// Do calls Instances() to resolve the names given to a list of matching
// instances on host h (which can be geneos.ALL to look on all hosts)
// and for type ct, which can be nil to look across all component types.
func Do(h *geneos.Host, ct *geneos.Component, names []string, f func(geneos.Instance, ...any) *Response, values ...any) (responses Responses) {
	var wg sync.WaitGroup

	instances := Instances(h, ct, FilterNames(names...))
	responses = make(Responses, len(instances))
	ch := make(chan *Response, len(instances))

	for _, c := range instances {
		wg.Add(1)
		go func(c geneos.Instance) {
			defer wg.Done()

			resp := f(c, values...)
			resp.Finish = time.Now()
			ch <- resp
		}(c)
	}
	wg.Wait()
	close(ch)

	for resp := range ch {
		responses[resp.Instance.String()] = resp
	}

	return
}

// Disable the instance i. Does not try to stop a running instance and
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

// Enable removes the disabled flag, if any,m from instance i.
func Enable(i geneos.Instance) (err error) {
	disableFile := ComponentFilepath(i, geneos.DisableExtension)
	if _, err = i.Host().Stat(disableFile); err != nil {
		// not disabled, return with no error
		return nil
	}
	return i.Host().Remove(disableFile)
}

// Get return instance of component type ct, and loads the config. It is
// an error if the config cannot be loaded. The instance is loaded from
// the host given in the name after any '@' or, if none, localhost is
// used.
func Get(ct *geneos.Component, name string) (instance geneos.Instance, err error) {
	if ct == nil || name == "" {
		return nil, geneos.ErrInvalidArgs
	}

	instance = ct.New(name)
	if instance == nil {
		// if no instance is created, check why
		h, _, _ := Decompose(name)
		if h == geneos.LOCAL && geneos.LocalRoot() == "" {
			err = geneos.ErrRootNotSet
			return
		}
		err = geneos.ErrInvalidArgs
		return
	}
	err = instance.Load()
	return
}

// GetWithHost return instance of component type ct from host h and
// loads the config. It is an error if the config cannot be loaded. If
// name has an embedded "@HOST" it must match h, else that is an error.
func GetWithHost(h *geneos.Host, ct *geneos.Component, name string) (instance geneos.Instance, err error) {
	if ct == nil || h == nil || h == geneos.ALL || name == "" {
		return nil, geneos.ErrInvalidArgs
	}

	if strings.Contains(name, "@") && !strings.HasSuffix(name, "@"+h.String()) {
		err = geneos.ErrInvalidArgs
		return
	}

	instance = ct.New(h.FullName(name))
	if instance == nil {
		if h == geneos.LOCAL && geneos.LocalRoot() == "" {
			err = geneos.ErrRootNotSet
			return
		}
		err = geneos.ErrInvalidArgs
		return
	}
	err = instance.Load()
	return
}

// Instances returns a slice of all instances on host h of component
// type ct, where both can be nil in which case all hosts or component
// types are used respectively. The options allow filtering based on
// names or parameter matches.
func Instances(h *geneos.Host, ct *geneos.Component, options ...InstanceOptions) (instances []geneos.Instance) {
	var instanceNames []string

	for ct := range ct.OrList() {
		for _, name := range InstanceNames(h, ct) {
			instanceNames = append(instanceNames, ct.String()+":"+name)
		}
	}

	opts := evalInstanceOptions(options...)

	if len(opts.names) > 0 {
		instanceNames = slices.DeleteFunc(instanceNames, func(n string) bool {
			ih, _, in := Decompose(n, h)
			for _, v := range opts.names {
				h, _, name := Decompose(v, h)
				if name == in && (h == geneos.ALL || h == ih) {
					return false
				}
			}
			return true
		})
	}

	for _, name := range instanceNames {
		h, ct, name := Decompose(name, h)
		instance, err := GetWithHost(h, ct, name)
		if err != nil {
			continue
		}
		instances = append(instances, instance)
	}

	if len(opts.parameters) > 0 {
		params := map[string]string{}
		for _, v := range opts.parameters {
			if v == "" {
				continue
			}
			s := strings.SplitN(v, "=", 2)
			if len(s) == 2 {
				params[s[0]] = s[1]
			}
		}

		instances = slices.DeleteFunc(instances, func(i geneos.Instance) bool {
			for p, v := range params {
				if i.Config().GetString(p) != v {
					return true
				}
			}
			return false
		})
	}

	return
}

type instanceOptions struct {
	names      []string
	parameters []string
}
type InstanceOptions func(*instanceOptions)

func evalInstanceOptions(options ...InstanceOptions) (opts *instanceOptions) {
	opts = &instanceOptions{}
	for _, opt := range options {
		opt(opts)
	}
	return
}

func FilterNames(names ...string) InstanceOptions {
	return func(io *instanceOptions) {
		io.names = names
	}
}

func FilterParameters(parameters ...string) InstanceOptions {
	return func(io *instanceOptions) {
		io.parameters = parameters
	}
}

// InstanceNames returns a slice of all the base names for instance
// directories for a given component ct on host h. No checking is done
// to validate that the directory contains a valid instance.
// InstanceNames are qualified with the host name. Regular files are
// ignored.
//
// To support the move to parent types we do a little more, looking for
// legacy locations in here
func InstanceNames(h *geneos.Host, ct *geneos.Component) (names []string) {
	for h := range h.OrList() {
		for ct := range ct.OrList() {
			for _, dir := range ct.InstancesBaseDirs(h) {
				d, err := h.ReadDir(dir)
				if err != nil {
					continue
				}
				for _, f := range d {
					if f.IsDir() {
						names = append(names, f.Name()+"@"+h.String())
					}
				}
			}
		}
	}

	return
}

// Match applies file glob patterns to all instance names (stripped of
// hostname) on the host h and of the component type ct and returns all
// matches. Valid patterns are the same as for path.Match.
//
// The returned slice is sorted and duplicates are removed.
//
// Patterns that resolve to empty (e.g. @hostname) are returned
// unchanged and unchecked against valid names.
//
// Patterns that have no globbing special characters are returned as-is
// and the caller is expected to validate them.
func Match(h *geneos.Host, ct *geneos.Component, keepHosts bool, patterns ...string) (names []string) {
	for _, pattern := range patterns {
		if pattern == "all" {
			pattern = "*"
		}

		// a host-only name implies a wildcard at host
		if !keepHosts && strings.HasPrefix(pattern, "@") {
			pattern = "*" + pattern
		}

		// check for glob chars, if none then just add the pattern as-is and loop
		if !strings.ContainsAny(pattern, `*?[`) {
			names = append(names, pattern)
			continue
		}

		h, _, p := Decompose(pattern, h) // override 'h' inside loop

		for _, name := range InstanceNames(h, ct) {
			_, _, n := Decompose(name, h)
			if match, _ := path.Match(p, n); match {
				if h == geneos.ALL {
					names = append(names, n)
				} else {
					names = append(names, n+"@"+h.String())
				}
			}
		}
	}

	// remove duplicates in the slice of names without changing order
	newNames := []string{}
	mapNames := map[string]bool{}
	for _, n := range names {
		if _, ok := mapNames[n]; !ok {
			mapNames[n] = true
			newNames = append(newNames, n)
		}
	}
	names = newNames

	// if the result is no matches but we were given patterns, return
	// those patterns and let the caller fail them when loading
	if len(names) == 0 && len(patterns) > 0 {
		return patterns
	}
	return
}

// Decompose returns the parts of an instance name in the format
// [TYPE:]NAME[@HOST]. ct defaults to nil and host to localhost unless
// an optional default host is passed as a var arg.
func Decompose(name string, defaultHost ...*geneos.Host) (host *geneos.Host, ct *geneos.Component, instance string) {
	var t, h string
	var found bool

	if len(defaultHost) == 0 {
		host = geneos.LOCAL
	} else {
		host = defaultHost[0]
	}

	t, name, found = strings.Cut(name, ":")
	if found {
		ct = geneos.ParseComponent(t)
	} else {
		name = t
	}

	instance, h, found = strings.Cut(name, "@")
	if found {
		host = geneos.GetHost(h)
	}

	return
}

func ImportFiles(s geneos.Instance, files ...string) (err error) {
	for _, source := range files {
		if _, err = geneos.ImportSource(s.Host(), s.Home(), source); err != nil {
			return
		}
	}
	return
}

// validNameRE is the test for what is a potentially valid instance name
// versus a parameter. spaces are valid - dumb, but valid - for now. If
// the name starts with number then the next character cannot be a
// number or '.' to help distinguish from versions.
//
// in addition to static names we also allow glob-style characters
// through
//
// look for "[flavour:]name[@host]" - only name can contain glob chars
var validNameRE = regexp.MustCompile(`^(\w+:)?([\w\.\-\ _\*\?\[\]\^\]]+)?(@[\w\-_\.]*)?$`)

func ValidName(name string) bool {
	return validNameRE.MatchString(name)
}
