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
	"os"
	"path"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Copy copies the instance named source to destination for
// component type ct. If the move argument is true then the source is
// deleted after the copy.
//
// Both source and destination can include host labels as well as being
// only host labels to indicate all instances of type ct on that host.
// If source is in the form `@host` then destination must also be a host
// - and different - or the function returns an error, but is it valid
// to have a specific source and a destination of only `@host` and then
// the name of the instance is used, as with file system operations on
// files and directories normally.
//
// If ct is nil that all component types are considered
func Copy(ct *geneos.Component, source, destination string, move bool) (err error) {
	var stopped, done bool
	if source == destination {
		return fmt.Errorf("source and destination must have different names and/or locations")
	}

	log.Debug().Msgf("%s %s %s", ct, source, destination)

	if strings.HasPrefix(source, "@") {
		if !strings.HasPrefix(destination, "@") {
			return fmt.Errorf("%w: destination must be a host when source is a host", geneos.ErrInvalidArgs)
		}
		sHostName := strings.TrimPrefix(source, "@")
		dHostName := strings.TrimPrefix(destination, "@")
		if sHostName == dHostName {
			return fmt.Errorf("%w: src and destination host must be different", geneos.ErrInvalidArgs)
		}
		sHost := geneos.GetHost(sHostName)
		if !sHost.Exists() {
			return fmt.Errorf("%w: source host %q not found", os.ErrNotExist, sHostName)
		}
		dHost := geneos.GetHost(dHostName)
		if !dHost.Exists() {
			return fmt.Errorf("%w: destination host %q not found", os.ErrNotExist, dHostName)
		}
		// they both exist, now loop through all instances on src and try to move/copy
		for _, name := range AllNames(sHost, ct) {
			if err = Copy(ct, name, destination, move); err != nil {
				fmt.Println("Error:", err)
			}
		}
		return nil
	}

	if ct == nil {
		for _, t := range geneos.RealComponents() {
			if err = Copy(t, source, destination, move); err != nil {
				log.Debug().Err(err).Msg("")
				continue
			}
		}
		return nil
	}

	src, err := Match(ct, geneos.ALL, source)
	if err != nil {
		return fmt.Errorf("%w: %q %q", err, ct, source)
	}

	if err = Migrate(src); err != nil {
		return fmt.Errorf("%s %s cannot be migrated to new configuration format", ct, source)
	}

	// if destination is just a host, tack the src prefix on to the start
	// let further calls check for syntax and validity
	if strings.HasPrefix(destination, "@") {
		destination = src.Name() + destination
	}

	log.Debug().Msgf("checking %s", destination)
	dst, err := Get(ct, destination)
	if err == nil && dst.Loaded() {
		log.Debug().Msg("already exists")
		return fmt.Errorf("%s already exists", dst)
	}
	log.Debug().Msg("not found")
	// otherwise carry on
	dst.Unload()

	if move {
		if IsRunning(src) {
			if err = Stop(src, true, false); err == nil {
				stopped = true
				// defer a call to restart the original if not "done"
				defer func(c geneos.Instance) {
					if !done {
						Start(c)
					}
				}(src)
			} else {
				return fmt.Errorf("cannot stop %s", source)
			}
		}
	}

	_, dName, dHost := SplitName(destination, geneos.LOCAL)

	// do a dance here to deep copy-ish the dst
	newdst := src.Type().New(destination)

	// copy over settings from source
	newdst.Config().MergeConfigMap(src.Config().AllSettings())
	// set the port to zero so that subsequent test for valid ports
	// doesn't match this one
	newdst.Config().Set("port", 0)

	// copy directory
	if err = host.CopyAll(src.Host(), src.Home(), dHost, dst.Home()); err != nil {
		log.Debug().Err(err).Msg("")
		return
	}

	// delete one or the other, depending
	defer func(srcname string, srcrem *geneos.Host, srchome string, dst geneos.Instance) {
		if done {
			if move {
				// once we are done, try to delete old instance
				log.Debug().Msgf("removing old instance %s", srcname)
				srcrem.RemoveAll(srchome)
				fmt.Println(srcname, "moved to", dst)
			} else {
				fmt.Println(srcname, "copied to", destination)
			}
		} else {
			// remove new instance
			log.Debug().Msgf("removing new instance %s", dst)
			dst.Host().RemoveAll(dst.Home())
		}
	}(src.String(), src.Host(), src.Home(), dst)

	// XXX update *Home manually, as it's not just the prefix
	newdst.Config().Set("home", path.Join(dst.Type().InstancesDir(dHost), dName))

	if src.Host() == dHost {
		if !move {
			dPort := NextPort(dHost, dst.Type())
			newdst.Config().Set("port", dPort)
		} else {
			newdst.Config().Set("port", src.Config().GetUint16("port"))
		}
	} else {
		sPort := src.Config().GetUint16("port")
		dPortsInUse := GetPorts(dHost)
		if _, ok := dPortsInUse[sPort]; ok {
			log.Debug().Msgf("found port in use: %d -> %s", sPort, dPortsInUse[sPort])
			dPort := NextPort(dHost, dst.Type())
			newdst.Config().Set("port", dPort)
		} else {
			newdst.Config().Set("port", src.Config().GetUint16("port"))
		}
	}

	// update any component name only if the same as the instance name
	log.Debug().Msgf("src name: %s, setting dst to %s", src.Config().GetString("name"), destination)
	_, newname, _ := SplitName(destination, geneos.LOCAL)
	newdst.Config().Set("name", newname)

	// config changes don't matter until writing config succeeds
	log.Debug().Msgf("writing: %v", newdst.Config().AllSettings())
	if err = SaveConfig(newdst); err != nil {
		log.Debug().Err(err).Msg("")
		return
	}

	// src.Unload()
	if err = newdst.Rebuild(false); err != nil && err != geneos.ErrNotSupported {
		log.Debug().Err(err).Msg("")
		return
	}

	done = true

	// now a full clean on the destination
	if err = Clean(newdst, true); err != nil {
		return
	}

	if stopped {
		return Start(newdst)
	}
	return nil
}
