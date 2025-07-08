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
	"path"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// ComponentFilename returns the filename for the component used by the
// instance i with the extensions appended, joined with a ".". If no
// extensions are given then the current configuration file type is
// used, e.g. "json" or "yaml".
func ComponentFilename(i geneos.Instance, extensions ...string) string {
	parts := []string{i.Type().String()}
	if len(extensions) > 0 {
		parts = append(parts, extensions...)
	} else {
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
	return path.Join(i.Home(), ComponentFilename(i, extensions...))
}

// FileOf returns the basename of the file identified by the
// instance parameter name.
//
// If the parameter is unset or empty then an empty path is returned.
func FileOf(i geneos.Instance, name string) (filename string) {
	cf := i.Config()

	if cf == nil {
		return
	}
	// return empty and not a "."
	filename = path.Base(cf.GetString(name))
	if filename == "." {
		filename = ""
	}
	return
}

// PathOf returns the full path to the file identified by the
// configuration parameter name. If the parameters value is already an
// absolute path then it is returned as-is, otherwise it is joined with
// the home directory of the instance and returned. The path is only
// useful on the host that instance i is on.
//
// If the parameter is unset or empty then an empty path is returned.
func PathOf(i geneos.Instance, name string) string {
	cf := i.Config()

	if cf == nil {
		return ""
	}
	filename := cf.GetString(name)
	if filename == "" {
		return ""
	}

	if path.IsAbs(filename) {
		return filename
	}

	return path.Join(i.Home(), filename)
}

// Abs returns an absolute path to file prepended with the instance
// working directory if file is not already an absolute path. If file is
// empty then an empty result is returned.
func Abs(i geneos.Instance, file string) (result string) {
	if file == "" {
		return
	}
	result = path.Clean(file)
	if path.IsAbs(result) {
		return
	}
	return path.Join(i.Home(), result)
}

// Filepaths returns the full paths to the files identified by names.
//
// If the instance configuration is valid an empty slice is returned. If
// a parameter is unset or empty then an empty path is returned.
func Filepaths(i geneos.Instance, names ...string) (filenames []string) {
	cf := i.Config()

	if cf == nil {
		return
	}

	for _, name := range names {
		// note: Abs(i, "") returns ""
		filenames = append(filenames, Abs(i, cf.GetString(name)))
	}
	return
}

// Home return the directory for the instance. It checks for the first existing directory from:
//
//   - The one configured for the instance factory and in the configuration parameter "home"
//   - In the default component instances directory (component.InstanceDir)
//   - If the instance's component type has a parent component then in the
//     legacy instances directory
//
// If no directory is found then a default built using PathTo() is returned
func Home(i geneos.Instance) (home string) {
	if i.Config() == nil {
		return ""
	}

	ct := i.Type()
	h := i.Host()

	// can't use c.Home() as this function is called from there!
	if i.Config().IsSet("home") {
		home = i.Config().GetString("home")
		if d, err := h.Stat(home); err == nil && d.IsDir() {
			return
		}
	}

	// second, does the instance exist in the default instances parentDir?
	parentDir := i.Type().InstancesDir(h)
	if parentDir != "" {
		home = path.Join(parentDir, i.Name())
		if d, err := h.Stat(home); err == nil && d.IsDir() {
			return
		}
	}

	// third, look in any "legacy" location, but only if parent type is
	// non nil
	if i.Type().ParentType != nil {
		parentDir := h.PathTo(i.Type().String(), i.Type().String()+"s")
		if parentDir != "" {
			home = path.Join(parentDir, i.Name())
			if d, err := h.Stat(home); err == nil && d.IsDir() {
				return
			}
		}
	}

	home = h.PathTo(ct, ct.String()+"s", i.Name())
	return
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
func CheckPaths(i geneos.Instance, paths []string) (missing []string) {
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
