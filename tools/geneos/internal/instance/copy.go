package instance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
)

func CopyInstance(ct *geneos.Component, srcname, dstname string, remove bool) (err error) {
	var stopped, done bool
	if srcname == dstname {
		return fmt.Errorf("source and destination must have different names and/or locations")
	}

	log.Debug().Msgf("%s %s %s", ct, srcname, dstname)

	// move/copy all instances from host
	// destination must also be a host and different and exist
	if strings.HasPrefix(srcname, "@") {
		if !strings.HasPrefix(dstname, "@") {
			return fmt.Errorf("%w: destination must be a host when source is a host", geneos.ErrInvalidArgs)
		}
		srchost := strings.TrimPrefix(srcname, "@")
		dsthost := strings.TrimPrefix(dstname, "@")
		if srchost == dsthost {
			return fmt.Errorf("%w: src and destination host must be different", geneos.ErrInvalidArgs)
		}
		sr := host.Get(srchost)
		if !sr.Exists() {
			return fmt.Errorf("%w: source host %q not found", os.ErrNotExist, srchost)
		}
		dr := host.Get(dsthost)
		if !dr.Exists() {
			return fmt.Errorf("%w: destination host %q not found", os.ErrNotExist, dsthost)
		}
		// they both exist, now loop through all instances on src and try to move/copy
		for _, name := range AllNames(sr, ct) {
			CopyInstance(ct, name, dstname, remove)
		}
		return nil
	}

	if ct == nil {
		for _, t := range geneos.RealComponents() {
			if err = CopyInstance(t, srcname, dstname, remove); err != nil {
				log.Debug().Err(err).Msg("")
				continue
			}
		}
		return nil
	}

	src, err := Match(ct, srcname)
	if err != nil {
		return fmt.Errorf("%w: %q %q", err, ct, srcname)
	}

	if err = Migrate(src); err != nil {
		return fmt.Errorf("%s %s cannot be migrated to new configuration format", ct, srcname)
	}

	// if dstname is just a host, tack the src prefix on to the start
	// let further calls check for syntax and validity
	if strings.HasPrefix(dstname, "@") {
		dstname = src.Name() + dstname
	}

	dst, err := Get(ct, dstname)
	if err != nil {
		log.Debug().Err(err).Msg("")
	}
	if dst.Loaded() {
		return fmt.Errorf("%s already exists", dst)
	}
	dst.Unload()

	if _, err = GetPID(src); err != os.ErrProcessDone {
		if err = Stop(src, false); err == nil {
			stopped = true
			// defer a call to restart the original if not "done"
			defer func(c geneos.Instance) {
				if !done {
					Start(c)
				}
			}(src)
		} else {
			return fmt.Errorf("cannot stop %s", srcname)
		}
	}

	// now a full clean
	if err = Clean(src, geneos.Restart(true)); err != nil {
		return
	}

	_, ds, dr := SplitName(dstname, host.LOCAL)

	// do a dance here to deep copy-ish the dst
	realdst := dst
	b, _ := json.Marshal(src)
	if err = json.Unmarshal(b, &realdst); err != nil {
		log.Error().Err(err).Msg("")
	}

	// move directory
	if err = host.CopyAll(src.Host(), src.Home(), dr, dst.Home()); err != nil {
		return
	}

	// delete one or the other, depending
	defer func(srcname string, srcrem *host.Host, srchome string, dst geneos.Instance) {
		if done {
			if remove {
				// once we are done, try to delete old instance
				log.Debug().Msgf("removing old instance %s", srcname)
				srcrem.RemoveAll(srchome)
				fmt.Println(srcname, "moved to", dst)
			} else {
				fmt.Println(srcname, "copied to", dstname)
			}
		} else {
			// remove new instance
			log.Debug().Msgf("removing new instance %s", dst)
			dst.Host().RemoveAll(dst.Home())
		}
	}(src.String(), src.Host(), src.Home(), dst)

	// update *Home manually, as it's not just the prefix
	realdst.Config().Set("home", filepath.Join(dst.Type().ComponentDir(dr), ds))
	// dst.Unload()

	// fetch a new port if hosts are different and port is already used
	if src.Host() != dr {
		srcport := src.Config().GetInt64("port")
		dstports := GetPorts(dr)
		if _, ok := dstports[uint16(srcport)]; ok {
			dstport := NextPort(dr, dst.Type())
			realdst.Config().Set("port", fmt.Sprint(dstport))
		}
	}

	// update any component name only if the same as the instance name
	if src.Config().GetString("name") == srcname {
		realdst.Config().Set("name", dstname)
	}

	// config changes don't matter until writing config succeeds
	if err = WriteConfig(realdst); err != nil {
		log.Debug().Err(err).Msg("")
		return
	}

	// src.Unload()
	if err = realdst.Rebuild(false); err != nil && err != geneos.ErrNotSupported {
		log.Debug().Err(err).Msg("")
		return
	}

	done = true
	if stopped {
		return Start(realdst)
	}
	return nil
}
