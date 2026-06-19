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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

// The Instance type is the common data shared by all instances
type Instance struct {
	Conf         *config.Config    `json:"-"`
	InstanceHost *geneos.Host      `json:"-"`
	Component    *geneos.Component `json:"-"`
	ConfigLoaded time.Time         `json:"-"`
	Logger       *slog.Logger      `json:"-"`
}

var instanceMutex sync.Mutex

var log = cordial.Logger

// NewLogger returns a logger with the instance name, host and type in
// the context. The logger is configured with the "cordial" prefix and
// an indent, and is set to the Info level.
func NewLogger(i geneos.Instance, groups ...string) (l *slog.Logger) {
	l = slog.New(cordial.LogHandler)
	for _, group := range groups {
		l = l.WithGroup(group)
	}
	return l.With(
		slog.String("name", i.Name()),
		slog.String("host", i.Host().String()),
		slog.String("type", i.Type().String()),
	)
}

func CloneConfig(i geneos.Instance) (cf *config.Config) {
	instanceMutex.Lock()
	defer instanceMutex.Unlock()
	cf = config.New()
	cf.MergeConfigMap(i.Config().AllSettings())
	return
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
	if i == nil {
		return "<nil>"
	}
	if i.Host().IsLocalhost() {
		return fmt.Sprintf("%s %q", i.Type(), i.Name())
	}
	return fmt.Sprintf("%s \"%s@%s\"", i.Type(), i.Name(), i.Host())
}

// ShortName returns the instance name with the "@HOST" suffix as a
// string
func ShortName(i geneos.Instance) string {
	if i == nil {
		return "<nil>"
	}
	return i.Name() + "@" + i.Host().String()
}

// IDString returns a string suitable as a unique rowname for Toolkit
// output and other places. For local instance it return `TYPE:NAME` and
// for remote instances it returns `TYPE:NAME@HOST`
func IDString(i geneos.Instance) string {
	if i == nil {
		return "<nil>"
	}
	if i.Host().IsLocalhost() {
		return i.Type().String() + ":" + i.Name()
	}
	return i.Type().String() + ":" + i.Name() + "@" + i.Host().String()
}

// ReservedName returns true if name is a reserved word. Reserved names
// are checked against all the values registered by components at
// start-up.
func ReservedName(name string) (ok bool) {
	log.Debug("checking", slog.String("name", name))
	if geneos.ParseComponent(name) != nil {
		log.Debug("matches a reserved word")
		return true
	}
	if reserved := config.Get[string](config.Global(), "reservednames"); reserved != "" {
		list := strings.SplitSeq(reserved, ",")
		for n := range list {
			if strings.EqualFold(name, strings.TrimSpace(n)) {
				log.Debug("matches a user defined reserved name", slog.String("name", name), slog.String("reserved", n))
				return true
			}
		}
	}
	return
}

// LogFilePath returns the full path to the log file for the instance.
func LogFilePath(i geneos.Instance) (logfile string) {
	if i == nil {
		return ""
	}
	logdir := path.Clean(config.Get[string](i.Config(), "logdir"))
	switch {
	case logdir == "":
		logfile = i.Home()
	case path.IsAbs(logdir):
		logfile = logdir
	default:
		logfile = path.Join(i.Home(), logdir)
	}
	logfile = path.Join(logfile, config.Get[string](i.Config(), "logfile"))
	return
}

// Signal sends the signal to the instance
func Signal(i geneos.Instance, signal syscall.Signal) (err error) {
	if i == nil {
		return os.ErrInvalid
	}
	pid, err := GetPID(i) // check cache first
	if err != nil && errors.Is(err, os.ErrProcessDone) {
		// only check live PID if no entry found in cache. If a PID is
		// found in the cache but the process has terminated, signal
		// will just return os.ErrProcessDone, so we don't want to check
		// live PID in that case.
		pid, err = GetLivePID(i)
		if err != nil {
			return os.ErrProcessDone
		}
	}

	return i.Host().Signal(int(pid), signal)
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
func Do(h *geneos.Host, ct *geneos.Component, names []string, f func(geneos.Instance, ...any) *responses.General, values ...any) (rs responses.GeneralResponses) {
	var wg sync.WaitGroup

	instances := Instances(h, ct, MatchNames(names...))
	rs = make(responses.GeneralResponses, len(instances))
	ch := make(chan *responses.General, len(instances))

	for _, c := range instances {
		wg.Add(1)
		go func(c geneos.Instance) {
			defer wg.Done()

			resp := f(c, values...)
			responses.Finished(resp)
			ch <- resp
		}(c)
	}
	wg.Wait()
	close(ch)

	for resp := range ch {
		rs[resp.Instance.String()] = resp
	}

	return
}

// DoSerial is a variant of Do that executes the function calls
// serially. This is for use by functions that are not concurrency safe.
func DoSerial(h *geneos.Host, ct *geneos.Component, names []string, f func(geneos.Instance, ...any) *responses.General, values ...any) (rs responses.GeneralResponses) {
	rs = make(responses.GeneralResponses)

	instances := Instances(h, ct, MatchNames(names...))

	for _, c := range instances {
		resp := f(c, values...)
		responses.Finished(resp)
		rs[resp.Instance.String()] = resp
	}

	return
}

// DoInstances is a variant of Do that takes a slice of instances
// instead of looking them up by host, type and name. This is for use by
// functions that have already looked up instances and want to call a
// function on them concurrently.
func DoInstances(instances []geneos.Instance, f func(geneos.Instance, ...any) *responses.General, values ...any) (rs responses.GeneralResponses) {
	var wg sync.WaitGroup

	rs = make(responses.GeneralResponses, len(instances))
	ch := make(chan *responses.General, len(instances))

	for _, c := range instances {
		if c == nil {
			continue
		}
		wg.Add(1)
		go func(c geneos.Instance) {
			defer wg.Done()

			resp := f(c, values...)
			responses.Finished(resp)
			ch <- resp
		}(c)
	}
	wg.Wait()
	close(ch)

	for resp := range ch {
		rs[resp.Instance.String()] = resp
	}

	return
}

// Disable the instance i. Does not try to stop a running instance and
// returns an error if it is running.
func Disable(i geneos.Instance) (err error) {
	if i == nil {
		return os.ErrInvalid
	}
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
	if i == nil {
		return os.ErrInvalid
	}
	disableFile := ComponentFilepath(i, geneos.DisableExtension)
	if _, err = i.Host().Stat(disableFile); err != nil {
		// not disabled, return with no error
		return nil
	}
	return i.Host().Remove(disableFile)
}

// Get return instance `name` of component type ct, and loads the
// config. It is an error if the config cannot be loaded. The instance
// is loaded from the host given in the name after any '@' or, if none,
// localhost is used.
func Get(ct *geneos.Component, name string) (instance geneos.Instance, err error) {
	if ct == nil || name == "" {
		return nil, geneos.ErrInvalidArgs
	}

	instance = ct.New(name)
	if instance == nil {
		// if no instance is created, check why
		h, _, _ := ParseName(name)
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

// GetWithHost return instance of component name of type ct from host h
// and loads the config. It is an error if the config cannot be loaded.
// If name has an embedded "@HOST" it must match h, else that is an
// error.
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
func Instances(h *geneos.Host, ct *geneos.Component, options ...InstanceOption) (instances []geneos.Instance) {
	var instanceNames []string

	opts := evalInstanceOptions(options...)

	for ct := range ct.OrList() {
		for _, name := range AllInstanceNames(h, ct) {
			instanceNames = append(instanceNames, ct.String()+":"+name)
		}
	}

	if len(opts.names) > 0 {
		instanceNames = slices.DeleteFunc(instanceNames, func(n string) bool {
			ih, _, in := ParseName(n, h)
			for _, v := range opts.names {
				h, _, name := ParseName(v, h)
				if name == in && (h == geneos.ALL || h == ih) {
					return false
				}
			}
			return true
		})
	}

	for _, name := range instanceNames {
		h, ct, name := ParseName(name, h)
		i, err := GetWithHost(h, ct, name)
		if err != nil {
			continue
		}
		if opts.matchParentPackage {
			if pkgtype, ok := config.Lookup[string](i.Config(), "pkgtype"); ok {
				if pkgtype != ct.String() {
					continue
				}
			}
		}
		instances = append(instances, i)
	}

	// now add parent type matches
	if opts.matchParentPackage {
		pt := ct.ParentType

		var parentNames []string
		for _, name := range AllInstanceNames(h, pt) {
			parentNames = append(parentNames, pt.String()+":"+name)
		}

		if len(opts.names) > 0 {
			parentNames = slices.DeleteFunc(parentNames, func(n string) bool {
				ih, _, in := ParseName(n, h)
				for _, v := range opts.names {
					h, _, name := ParseName(v, h)
					if name == in && (h == geneos.ALL || h == ih) {
						return false
					}
				}
				return true
			})
		}

		for _, name := range parentNames {
			h, nct, name := ParseName(name, h)
			i, err := GetWithHost(h, nct, name)
			if err != nil {
				continue
			}
			if pkgtype, ok := config.Lookup[string](i.Config(), "pkgtype"); ok {
				// check the ORIGINAL component here
				if pkgtype != ct.String() {
					continue
				}
			}

			instances = append(instances, i)
		}
	}

	if len(opts.parameters) > 0 {
		params := map[string]string{}
		for _, v := range opts.parameters {
			if v == "" {
				continue
			}
			k, v, found := strings.Cut(v, "=")
			if !found {
				continue
			}
			params[k] = v
		}

		instances = slices.DeleteFunc(instances, func(i geneos.Instance) bool {
			for p, v := range params {
				if config.Get[string](i.Config(), p) != v {
					return true
				}
			}
			return false
		})
	}

	return
}

type instanceOptions struct {
	names              []string
	parameters         []string
	matchParentPackage bool
}
type InstanceOption func(*instanceOptions)

func evalInstanceOptions(options ...InstanceOption) (opts *instanceOptions) {
	opts = &instanceOptions{}
	for _, opt := range options {
		opt(opts)
	}
	return
}

// MatchNames returns an InstanceOption that filters instances based on
// matching names. The names should be in the form [TYPE:]NAME[@HOST] and
// are ORed together. If a name is not in that form it is ignored.
func MatchNames(names ...string) InstanceOption {
	return func(io *instanceOptions) {
		io.names = names
	}
}

// MatchParameters returns an InstanceOption that filters instances
// based on matching config parameters. The parameters should be in the
// form "key=value" and are ANDed together. If a parameter is not in
// that form it is ignored.
func MatchParameters(parameters ...string) InstanceOption {
	return func(io *instanceOptions) {
		io.parameters = parameters
	}
}

// MatchParentPackage returns an InstanceOption that filters instances
// to those whose component could be a parent type for another component
// *and* the package type matches. For example, a `netprobe` may use a
// `pkgtype` of `minimal`, so when looking for `minimal` instances, also
// include `netprobe` instances that have a `pkgtype` of `minimal`.
func MatchParentPackage() InstanceOption {
	return func(io *instanceOptions) {
		io.matchParentPackage = true
	}
}

// AllInstanceNames returns a slice of all the base names for instance
// directories for a given component ct on host h. No checking is done
// to validate that the directory contains a valid instance.
// AllInstanceNames are qualified with the host name. Regular files are
// ignored.
func AllInstanceNames(h *geneos.Host, ct *geneos.Component) (names []string) {
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
func Match(h *geneos.Host, ct *geneos.Component, keepHosts bool, mustMatch bool, patterns ...string) (names []string, err error) {
	for _, pattern := range patterns {
		if pattern == "all" {
			pattern = "*"
		}

		// a host-only name implies a wildcard at host
		if !keepHosts && strings.HasPrefix(pattern, "@") {
			pattern = "*" + pattern
		}

		h, _, p := ParseName(pattern, h) // override 'h' inside loop

		var matched bool
		for _, name := range AllInstanceNames(h, ct) {
			_, _, n := ParseName(name, h)
			if match, _ := path.Match(p, n); match {
				log.Debug("pattern matches instance name", slog.String("pattern", pattern), slog.String("name", name))
				matched = true
				if h == geneos.ALL {
					names = append(names, n)
				} else {
					names = append(names, n+"@"+h.String())
				}
			}
		}

		if !matched {
			if mustMatch {
				err = fmt.Errorf("%q does not match any instance names", pattern)
				return
			}

			// if it's a valid name just save it for the caller to check later, otherwise ignore it
			if ValidName(pattern) {
				log.Debug("pattern does not match any instance names but is a valid name, returning it for caller to check", slog.String("pattern", pattern))
				names = append(names, pattern)
			} else {
				log.Debug("pattern does not match any instance names and is not a valid name, ignoring", slog.String("pattern", pattern))
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
		return patterns, nil
	}
	return names, nil
}

// ParseName returns the parts of an instance name in the format
// [TYPE:]NAME[@HOST]. ct defaults to nil and host to localhost unless
// an optional default host is passed as a var arg.
func ParseName(name string, defaultHost ...*geneos.Host) (host *geneos.Host, ct *geneos.Component, instance string) {
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
	for _, item := range files {
		if _, err = geneos.ImportSource(s.Host(), s.Home(), item); err != nil {
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
