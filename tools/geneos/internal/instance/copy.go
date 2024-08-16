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
func Copy(ct *geneos.Component, source, destination string, options ...CopyOptions) (err error) {
	var stopped, done bool
	if source == destination {
		return fmt.Errorf("%w: source and destination must have different names and/or locations", geneos.ErrInvalidArgs)
	}

	log.Debug().Msgf("%s %s %s", ct, source, destination)

	opts := evalCopyOptions(options...)

	if strings.HasPrefix(source, "@") {
		if !strings.HasPrefix(destination, "@") {
			return fmt.Errorf("%w: destination must be a host when source is a host", geneos.ErrInvalidArgs)
		}
		sHostName := source[1:]      // strings.TrimPrefix(source, "@")
		dHostName := destination[1:] // strings.TrimPrefix(destination, "@")
		if sHostName == dHostName {
			return fmt.Errorf("%w: src and destination host must be different", geneos.ErrInvalidArgs)
		}
		sHost := geneos.GetHost(sHostName)
		if !sHost.Exists() {
			return fmt.Errorf("%w: source host %q not found", host.ErrNotExist, sHostName)
		}
		dHost := geneos.GetHost(dHostName)
		if !dHost.Exists() {
			return fmt.Errorf("%w: destination host %q not found", host.ErrNotExist, dHostName)
		}
		// they both exist, now loop through all instances on src and try to move/copy
		for _, name := range InstanceNames(sHost, ct) {
			if err = Copy(ct, name, destination, options...); err != nil {
				fmt.Println("Error:", err)
			}
		}
		return nil
	}

	if ct == nil {
		for _, ct := range geneos.RealComponents() {
			if err = Copy(ct, source, destination, options...); err != nil {
				log.Debug().Err(err).Msg("")
				if errors.Is(err, host.ErrNotExist) {
					return
				}
				continue
			}
		}
		return nil
	}

	src, err := Get(ct, source)
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

	_, _, dHost := SplitName(destination, geneos.LOCAL)
	if !dHost.Exists() {
		return fmt.Errorf("%w: destination host for %q not found", host.ErrNotExist, destination)
	}

	log.Debug().Msgf("checking %s", destination)
	dst, err := Get(ct, destination)
	if err == nil && !dst.Loaded().IsZero() {
		return fmt.Errorf("%s already exists", dst)
	}
	log.Debug().Msg("destination does not already exist")
	// otherwise carry on
	if dst != nil {
		dst.Unload()
	}

	if opts.move {
		if IsRunning(src) {
			if err = Stop(src, true, false); err == nil {
				stopped = true
				// defer a call to restart the original if not "done"
				defer func(i geneos.Instance) {
					if !done {
						Start(i)
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
	ncf := newdst.Config()

	// copy over settings from source
	ncf.MergeConfigMap(src.Config().AllSettings())
	// set the port to zero so that subsequent test for valid ports
	// doesn't match this one
	ncf.Set("port", 0)

	// set override parameters here
	ncf.SetKeyValues(opts.params...)

	// copy directory
	if err = host.CopyAll(src.Host(), src.Home(), dHost, dst.Home()); err != nil {
		log.Debug().Err(err).Msg("")
		return
	}

	// delete one or the other, depending, after finishing other ops
	defer func(srcname string, srcrem *geneos.Host, srchome string, dst geneos.Instance) {
		if done {
			if opts.move {
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
	ncf.Set("home", path.Join(dst.Type().InstancesDir(dHost), dName))

	// only set a new port if not set through command line parameters
	if ncf.GetInt("port") == 0 {
		if src.Host() == dHost {
			if !opts.move {
				dPort := NextFreePort(dHost, dst.Type())
				ncf.Set("port", dPort)
			} else {
				ncf.Set("port", src.Config().GetUint16("port"))
			}
		} else {
			sPort := src.Config().GetUint16("port")
			dPortsInUse := GetAllPorts(dHost)
			if _, ok := dPortsInUse[sPort]; ok {
				log.Debug().Msgf("found port in use: %d", sPort)
				dPort := NextFreePort(dHost, dst.Type())
				ncf.Set("port", dPort)
			} else {
				ncf.Set("port", src.Config().GetUint16("port"))
			}
		}
	}

	// update any component name only if the same as the instance name
	log.Debug().Msgf("src name: %s, setting dst to %s", src.Config().GetString("name"), destination)
	_, newname, _ := SplitName(destination, geneos.LOCAL)
	ncf.Set("name", newname)

	// fix-up any other config changes here
	//
	// 1. keyfile/prevkeyfile from absolute path to using `${config:home}` if there
	keyfile := ncf.GetString("keyfile")
	log.Debug().Msgf("keyfile: %q", keyfile)
	if keyfile != "" {
		k := strings.Replace(keyfile, src.Home(), "${config:home}", 1)
		log.Debug().Msgf("setting keyfile: %q", k)
		ncf.Set("keyfile", k)
	}
	prevkeyfile := ncf.GetString("prevkeyfile")
	if keyfile != "" {
		k := strings.Replace(prevkeyfile, src.Home(), "${config:home}", 1)
		ncf.Set("prevkeyfile", k)
	}

	// config changes don't matter until writing config succeeds
	log.Debug().Msgf("writing: %v", ncf.AllSettings())
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

type copyOptions struct {
	move   bool
	params []string
}

func evalCopyOptions(options ...CopyOptions) (co *copyOptions) {
	co = &copyOptions{}

	for _, opt := range options {
		opt(co)
	}
	return
}

type CopyOptions func(*copyOptions)

// Move tells Copy to remove the source instance(s) after the copy.
func Move() CopyOptions {
	return func(co *copyOptions) {
		co.move = true
	}
}

// Params add key=value parameters to the copied/moved destination,
// overriding the values of the source. Currently only supports plain
// paramaters, not structured ones like environments.
func Params(p ...string) CopyOptions {
	return func(co *copyOptions) {
		co.params = append(co.params, p...)
	}
}
