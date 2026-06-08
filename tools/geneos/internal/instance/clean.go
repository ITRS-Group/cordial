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
	"os"
	"path/filepath"

	zlog "github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Clean removes all the files and directories listed in the component
// type of the instance i for removal. First all entries in the
// CleanList are removed and then, if full is true, the instance is
// stopped and the entries in FullClean are removed. Any instances
// stopped are started up, but any that were already stopped will be
// left stopped.
func Clean(i geneos.Instance, options ...CleanOption) (err error) {
	var stopped bool

	opts := evalCleanOptions(options...)

	ct := i.Type()

	// TODO: move from filepath.SplitList to strings.Split and use platform specific separator in config
	cleanlist := filepath.SplitList(config.Get[string](config.Global(), ct.CleanList, config.DefaultValue(ct.ConfigAliases[ct.CleanList])))
	if geneos.RootComponent.CleanList != "" {
		cleanlist = append(cleanlist, filepath.SplitList(geneos.RootComponent.CleanList)...)
	}

	// TODO: move from filepath.SplitList to strings.Split and use platform specific separator in config
	purgelist := filepath.SplitList(config.Get[string](config.Global(), ct.PurgeList, config.DefaultValue(ct.ConfigAliases[ct.PurgeList])))
	if geneos.RootComponent.PurgeList != "" {
		purgelist = append(purgelist, filepath.SplitList(geneos.RootComponent.PurgeList)...)
	}

	if !opts.full {
		if len(cleanlist) > 0 {
			if err = RemovePaths(i, cleanlist...); err == nil {
				zlog.Debug().Msgf("%s cleaned", i)
			}
		}
		return
	}

	if !IsRunning(i) {
		stopped = false
		// stop failed?
	} else if err := Stop(i, opts.force, false); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	} else {
		stopped = true
	}

	if len(cleanlist) > 0 {
		if err = RemovePaths(i, cleanlist...); err != nil {
			return
		}
	}
	if len(purgelist) > 0 {
		if err = RemovePaths(i, purgelist...); err != nil {
			return
		}
	}
	zlog.Debug().Msgf("%s created files removed", i)
	if stopped {
		err = Start(i)
	}
	return
}

// RemovePaths removes all files and directories in list
func RemovePaths(i geneos.Instance, list ...string) (err error) {
	h := i.Host()

	// list := filepath.SplitList(paths)
	for _, p := range list {
		// clean path, error on absolute or parent paths, like 'import'
		// walk globbed directories, remove everything
		if p, err = geneos.CleanRelativePath(p); err != nil {
			return fmt.Errorf("%s %w", p, err)
		}
		zlog.Debug().Msgf("going to remove %s", h.Join(i.Home(), p))
		// glob here
		m, err := i.Host().Glob(h.Join(i.Home(), p))
		if err != nil {
			return err
		}
		for _, f := range m {
			zlog.Debug().Msgf("trying to RemoveAll(%s)", f)
			if err = i.Host().RemoveAll(f); err != nil {
				zlog.Error().Err(err).Msg("")
				continue
			}
			fmt.Printf("removed %s\n", i.Host().HostPath(f))
		}
	}
	return
}
