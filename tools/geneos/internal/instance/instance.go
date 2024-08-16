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
	"os"
	"path"
	"regexp"
	"slices"
	"sort"
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
// distinguish from versions.
//
// # In addition to static names we also allow glob-style characters through
//
// look for "[flavour:]name[@host]" - only name can contain glob chars
var validNameRE = regexp.MustCompile(`^(\w+:)?([\w\.\-\ _\*\?\[\]\^\]]+)?(@[\w\-_\.]*)?$`)

// ValidName returns true if name is considered a valid instance
// name. It is not checked against the list of reserved names.
//
// XXX used to consume instance names until parameters are then passed
// down
func ValidName(name string) (ok bool) {
	ok = validNameRE.MatchString(name)
	if !ok {
		log.Debug().Msgf("not a valid instance name: %s", name)
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
// values into a slice of Response for handling by the caller. The
// functions are executed in goroutines and must be concurrency safe.
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

	instances, err := Instances(h, ct, FilterNames(names...))
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

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
		responses = append(responses, resp)
	}

	sort.Sort(responses)
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
		return nil
	}
	return i.Host().Remove(disableFile)
}

// Get return instance name of component type ct, and loads the config.
// It is an error if the config cannot be loaded. The instance is loaded
// from the host given in the name after any '@' or, if none, localhost
// is used.
func Get(ct *geneos.Component, name string) (instance geneos.Instance, err error) {
	if ct == nil || name == "" {
		return nil, geneos.ErrInvalidArgs
	}

	instance = ct.New(name)
	if instance == nil {
		// if no instance is created, check why
		_, _, h := SplitName(name, geneos.LOCAL)
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
func Instances(h *geneos.Host, ct *geneos.Component, options ...InstanceOptions) (instances []geneos.Instance, err error) {
	for _, ct := range ct.OrList() {
		for _, name := range InstanceNames(h, ct) {
			instance, err := Get(ct, name)
			if err != nil {
				continue
			}
			instances = append(instances, instance)
		}
	}

	opts := evalInstanceOptions(options...)

	if len(opts.names) > 0 {
		instances = slices.DeleteFunc(instances, func(i geneos.Instance) bool {
			for _, v := range opts.names {
				_, name, h := SplitName(v, h)
				if name == i.Name() && (h == geneos.ALL || h == i.Host()) {
					return false
				}
			}
			return true
		})
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
	for _, h := range h.OrList() {
		for _, ct := range ct.OrList() {
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
func Match(h *geneos.Host, ct *geneos.Component, patterns ...string) (names []string) {
	for _, pattern := range patterns {
		// check for glob chars
		if !strings.ContainsAny(pattern, `*?[`) {
			names = append(names, pattern)
		}
		// a host only name implies a wildcard
		if strings.HasPrefix(pattern, "@") {
			pattern = "*" + pattern
		}
		_, p, h := SplitName(pattern, h) // override 'h' inside loop
		for _, name := range InstanceNames(h, ct) {
			_, n, _ := SplitName(name, h)
			if match, _ := path.Match(p, n); match {
				if h == geneos.ALL {
					names = append(names, n)
				} else {
					names = append(names, n+"@"+h.String())
				}
			}
		}
	}
	slices.Sort(names)
	names = slices.Compact(names)
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
