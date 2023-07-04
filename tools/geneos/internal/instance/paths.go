/*
Copyright © 2023 ITRS Group

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
	"path"
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/rs/zerolog/log"
)

// ComponentFilename returns the filename for the component used by the
// instance c with the extensions appended, joined with a ".". If no
// extensions are given then the current configuration file type is
// used, e.g. "json" or "yaml".
func ComponentFilename(c geneos.Instance, extensions ...string) string {
	parts := []string{c.Type().String()}
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
func ComponentFilepath(c geneos.Instance, extensions ...string) string {
	return path.Join(c.Home(), ComponentFilename(c, extensions...))
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

// Filename returns the basename of the file identified by the
// configuration parameter name.
//
// If the parameter is unset or empty then an empty path is returned.
func Filename(c geneos.Instance, name string) (filename string) {
	cf := c.Config()

	if cf == nil {
		return
	}
	// return empty and not a "."
	filename = filepath.Base(cf.GetString(name))
	if filename == "." {
		filename = ""
	}
	return
}

// Filepath returns the full path to the file identified by the
// configuration parameter name. If the parameters value is already an
// absolute path then it is returned as-is, otherwise it is joined with
// the home directory of the instance and returned. The path is only
// useful on the host that instance c is on.
//
// If the parameter is unset or empty then an empty path is returned.
func Filepath(c geneos.Instance, name string) string {
	cf := c.Config()

	if cf == nil {
		return ""
	}
	filename := cf.GetString(name)
	if filename == "" {
		return ""
	}

	if filepath.IsAbs(filename) {
		return filename
	}

	return path.Join(c.Home(), filename)
}

// Filepaths returns the full paths to the files ideitified by the list
// of parameters in names.
//
// If the instance configuration is valid an empty slice is returned. If
// a parameter is unset or empty then an empty path is returned.
func Filepaths(c geneos.Instance, names ...string) (filenames []string) {
	cf := c.Config()

	if cf == nil {
		return
	}

	dir := HomeDir(c)

	for _, name := range names {
		if filepath.IsAbs(name) {
			filenames = append(filenames, name)
		} else {
			if !cf.IsSet(name) {
				continue
			}
			filename := filepath.Join(dir, cf.GetString(name))
			// return empty and not a "."
			if filename == "." {
				filename = ""
			}
			filenames = append(filenames, filename)
		}
	}
	return
}

// ConfigFileType returns the current primary configuration file
// extension
func ConfigFileType() (conftype string) {
	conftype = config.GetString("configtype")
	if conftype == "" {
		conftype = "json"
	}
	return
}

// ConfigFileTypes contains a list of supported configuration file
// extensions
func ConfigFileTypes() []string {
	return []string{"json", "yaml"}
}

// HomeDir return the validated-to-exist directory for the instance, or
// an empty string
func HomeDir(c geneos.Instance) (home string) {
	if c.Config() == nil {
		return ""
	}

	ct := c.Type()
	h := c.Host()

	// can't use c.Home() as this function is called from there!
	home = c.Config().GetString("home")
	if _, err := h.Stat(home); err != nil {
		home = h.Filepath(ct, ct.String()+"s", c.Name())
	}
	return
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
	home := c.Config().GetString("home")
	if home != "" {
		dir = filepath.Dir(home)
		// but check the configured home, not the parent
		if d, err := h.Stat(home); err == nil && d.IsDir() {
			log.Debug().Msgf("default home %s as defined", dir)
			return
		}
	}

	// second, does the instance exist in the default instances dir?
	dir = c.Type().InstancesDir(h)
	if dir != "" {
		if d, err := h.Stat(dir); err == nil && d.IsDir() {
			log.Debug().Msgf("instanceDir home %s selected", dir)
			return
		}
	}

	// third, look in any "legacy" location, but only if parent type is
	// non nil
	if c.Type().ParentType != nil {
		dir = filepath.Join(h.Filepath(c.Type(), c.Type().String()+"s"))
		if dir != "" {
			if d, err := h.Stat(dir); err == nil && d.IsDir() {
				log.Debug().Msgf("new home %s from legacy", dir)
				return
			}
		}
	}

	log.Debug().Msgf("default %s", dir)
	return dir
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
