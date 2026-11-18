/*
Copyright © 2023 ITRS Group

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
	"strings"
	"sync"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// ComponentFilename returns the filename for the component used by the
// instance i with the extensions appended, joined with a ".". If no
// extensions are given then the current configuration file type is
// used, e.g. "json" or "yaml".
func ComponentFilename(i geneos.Instance, extensions ...string) string {
	parts := []string{i.Type().String()}
	// strip dot prefixes
	for _, e := range extensions {
		e = strings.TrimPrefix(e, ".")
		if e != "" {
			parts = append(parts, e)
		}
	}
	if len(extensions) == 0 {
		parts = append(parts, ConfigFileType())
	}

	return strings.Join(parts, ".")
}

// ComponentFilepath returns an absolute path to a file named for the
// component type of the instance with any extensions joined using ".", e.g.
// is c is a netprobe instance then
//
//	path := instance.ComponentFilepath(c, "xml", "orig")
//
// will return /path/to/netprobe/netprobe.xml.orig
//
// If no extensions are passed then the default us to add an extension of the
// instance.ConfigType, which defaults to "json", e.g. using the same instance
// as above:
//
//	path := instance.ComponentPath(c)
//
// will return /path/to/netprobe/netprobe.json
func ComponentFilepath(i geneos.Instance, extensions ...string) string {
	h := i.Host()

	return h.Join(i.Home(), ComponentFilename(i, extensions...))
}

// FileOf returns the basename of the file identified by the
// instance parameter name.
//
// If the parameter is unset or empty then an empty path is returned.
func FileOf(i geneos.Instance, name string) (filename string) {
	cf := i.Config()
	h := i.Host()

	if cf == nil {
		return
	}
	// return empty and not a "."
	filename = h.Base(config.Get[string](cf, name))
	if filename == "." {
		filename = ""
	}
	return
}

// HomeRel returns an absolute path to file prepended with the instance
// working directory if file is not already an absolute path. If file is
// empty then an empty result is returned.
func HomeRel(i geneos.Instance, file string) (result string) {
	h := i.Host()

	if file == "" {
		return
	}

	result = h.Clean(file)
	if h.IsAbs(result) {
		return
	}
	return h.Join(i.Home(), result)
}

// PathTo returns the full path to the file identified by the
// configuration parameter name. If the parameters value is already an
// absolute path then it is returned as-is, otherwise it is joined with
// the home directory of the instance and returned. The path is only
// useful on the host that instance i is on.
//
// If the parameter is unset or empty then an empty path is returned.
func PathTo(i geneos.Instance, name string) string {
	cf := i.Config()

	if cf == nil {
		return ""
	}
	filename := config.Get[string](cf, name)
	if filename == "" {
		return ""
	}

	return HomeRel(i, filename)
}

// PathsTo returns the full paths to the files identified by names.
//
// If the instance configuration is valid an empty slice is returned. If
// a parameter is unset or empty then an empty path is returned.
func PathsTo(i geneos.Instance, names ...string) (filenames []string) {
	cf := i.Config()

	if cf == nil {
		return
	}

	for _, name := range names {
		// note: Abs(i, "") returns ""
		filenames = append(filenames, HomeRel(i, config.Get[string](cf, name)))
	}
	return
}

// homeState is the cached result of resolving the home directory of
// one instance. The mutex is held while resolving, so that a reset can
// never be overtaken by the store of a result that was resolved before
// it.
type homeState struct {
	mutex sync.Mutex

	valid bool
	home  string

	// the inputs that the result was resolved from and that are
	// cheap to check without any file system access
	cfHome string
	cfSet  bool
}

// homeKey identifies an instance, and not a particular Go value for it,
// as there can be more than one for the same instance, such as from
// Add() and Get().
type homeKey struct {
	host *geneos.Host
	ct   string
	name string
}

// homes holds a homeState for each instance that has been looked at.
var homes sync.Map

func homeStateFor(h *geneos.Host, ct *geneos.Component, name string) *homeState {
	key := homeKey{h, ct.String(), name}
	if s, ok := homes.Load(key); ok {
		return s.(*homeState)
	}
	s, _ := homes.LoadOrStore(key, &homeState{})
	return s.(*homeState)
}

// ResetHome discards the cached home directory of the instance, so that
// the next call to [Home] looks at the host again. It must be called by
// anything that can change where the home directory of an instance is,
// such as creating, moving or removing the directory.
//
// Changes to the "home" configuration parameter of an instance are
// always noticed and do not need a reset.
func ResetHome(i geneos.Instance) {
	if i == nil || i.Config() == nil {
		return
	}
	resetHome(i.Host(), i.Type(), i.Name())
}

func resetHome(h *geneos.Host, ct *geneos.Component, name string) {
	s := homeStateFor(h, ct, name)
	s.mutex.Lock()
	s.valid = false
	s.mutex.Unlock()
}

// Home return the directory for the instance. It checks for the first existing directory from:
//
//   - The one configured for the instance factory and in the configuration parameter "home"
//   - In the default component instances directory (component.InstanceDir)
//   - If the instance's component type has a parent component then in the
//     legacy instances directory
//
// If no directory is found then a default built using PathTo() is returned.
//
// Finding an existing directory needs a stat call per candidate, which
// is slow on a remote host, so the directory that is found is
// remembered for the instance until [ResetHome] is called or its
// configured "home" changes. A default that was not found is never
// remembered.
func Home(i geneos.Instance) (home string) {
	if i == nil {
		return ""
	}
	cf := i.Config()
	if cf == nil {
		return ""
	}

	h := i.Host()
	name := i.Name()
	cfHome, cfSet := config.Lookup[string](cf, "home")

	s := homeStateFor(h, i.Type(), name)
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.valid && s.cfSet == cfSet && s.cfHome == cfHome {
		return s.home
	}

	home, found := resolveHome(i, h, name, cfHome, cfSet)
	s.valid = found
	if found {
		s.home, s.cfHome, s.cfSet = home, cfHome, cfSet
	}
	return
}

// resolveHome looks for the existing home directory of the instance
// and returns it with true. If none exists then the default directory
// is returned with false.
func resolveHome(i geneos.Instance, h *geneos.Host, name, cfHome string, cfSet bool) (home string, found bool) {
	ct := i.Type()

	// can't use c.Home() as this function is called from there!
	if cfSet {
		if d, err := h.Stat(cfHome); err == nil && d.IsDir() {
			return cfHome, true
		}
	}

	// second, does the instance exist in the default instances parentDir?
	if parentDir := ct.InstancesDir(h); parentDir != "" {
		home = h.Join(parentDir, name)
		if d, err := h.Stat(home); err == nil && d.IsDir() {
			return home, true
		}
	}

	// third, look in any "legacy" location, but only if parent type is
	// non nil
	if ct.ParentType != nil {
		if parentDir := h.PathTo(ct.String(), ct.String()+"s"); parentDir != "" {
			home = h.Join(parentDir, name)
			if d, err := h.Stat(home); err == nil && d.IsDir() {
				return home, true
			}
		}
	}

	return h.PathTo(ct, ct.String()+"s", name), false
}

// Shared returns the full path to a directory or file in the instances
// component type shared directory joined to any parts subs - the last
// element can be a filename. If the instance is not loaded then "." is
// returned for the current directory.
func Shared(i geneos.Instance, subs ...any) string {
	if i == nil {
		return "."
	}
	return i.Type().Shared(i.Host(), subs...)
}

// CheckPaths checks paths for an existing file or directory, returning
// a list of missing paths. The check performed is a simple stat() for
// now.
func CheckPaths(i geneos.Instance, paths ...string) (missing []string) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		_, err := i.Host().Stat(p)
		if err != nil {
			missing = append(missing, p)
			continue
		}
	}
	return
}
