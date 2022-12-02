package instance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
)

func CopyInstance(ct *geneos.Component, source, destination string, move bool) (err error) {
	var stopped, done bool
	if source == destination {
		return fmt.Errorf("source and destination must have different names and/or locations")
	}

	log.Debug().Msgf("%s %s %s", ct, source, destination)

	// move/copy all instances from host
	// destination must also be a host and different and exist
	if strings.HasPrefix(source, "@") {
		if !strings.HasPrefix(destination, "@") {
			return fmt.Errorf("%w: destination must be a host when source is a host", geneos.ErrInvalidArgs)
		}
		sHostName := strings.TrimPrefix(source, "@")
		dHostName := strings.TrimPrefix(destination, "@")
		if sHostName == dHostName {
			return fmt.Errorf("%w: src and destination host must be different", geneos.ErrInvalidArgs)
		}
		sHost := host.Get(sHostName)
		if !sHost.Exists() {
			return fmt.Errorf("%w: source host %q not found", os.ErrNotExist, sHostName)
		}
		dHost := host.Get(dHostName)
		if !dHost.Exists() {
			return fmt.Errorf("%w: destination host %q not found", os.ErrNotExist, dHostName)
		}
		// they both exist, now loop through all instances on src and try to move/copy
		for _, name := range AllNames(sHost, ct) {
			if err = CopyInstance(ct, name, destination, move); err != nil {
				fmt.Println("Error:", err)
			}
		}
		return nil
	}

	if ct == nil {
		for _, t := range geneos.RealComponents() {
			if err = CopyInstance(t, source, destination, move); err != nil {
				fmt.Println("Error:", err)
				continue
			}
		}
		return nil
	}

	src, err := Match(ct, source)
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
		if _, err = GetPID(src); err != os.ErrProcessDone {
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

	_, dName, dHost := SplitName(destination, host.LOCAL)

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
	defer func(srcname string, srcrem *host.Host, srchome string, dst geneos.Instance) {
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

	// update *Home manually, as it's not just the prefix
	newdst.Config().Set("home", filepath.Join(dst.Type().InstancesDir(dHost), dName))

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
	_, newname, _ := SplitName(destination, host.LOCAL)
	newdst.Config().Set("name", newname)

	// config changes don't matter until writing config succeeds
	log.Debug().Msgf("writing: %v", newdst.Config().AllSettings())
	if err = WriteConfig(newdst); err != nil {
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
	if err = Clean(newdst, geneos.FullClean(true)); err != nil {
		return
	}

	if stopped {
		return Start(newdst)
	}
	return nil
}
