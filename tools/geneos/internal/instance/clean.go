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
	"strings"

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

	cleanlist := splitPathList(ct.GlobalSetting(ct.CleanList), pathListSeparators)
	if geneos.RootComponent.CleanList != "" {
		cleanlist = append(cleanlist, splitPathList(geneos.RootComponent.CleanList, pathListSeparators)...)
	}

	purgelist := splitPathList(ct.GlobalSetting(ct.PurgeList), pathListSeparators)
	if geneos.RootComponent.PurgeList != "" {
		purgelist = append(purgelist, splitPathList(geneos.RootComponent.PurgeList, pathListSeparators)...)
	}

	if !opts.full {
		if len(cleanlist) > 0 {
			if err = RemovePaths(i, cleanlist...); err == nil {
				i.Log().Debug("cleaned")
			}
		}
		return
	}

	if !IsRunning(i) {
		stopped = false
		// stop failed?
	} else if err := Stop(i, WithStopForce(opts.force)); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	} else {
		stopped = true
	}

	// an instance that was running is always restarted, even if some
	// paths could not be removed
	if len(cleanlist) > 0 {
		err = RemovePaths(i, cleanlist...)
	}
	if err == nil && len(purgelist) > 0 {
		err = RemovePaths(i, purgelist...)
	}
	if err == nil {
		i.Log().Debug("created files removed")
	}
	if stopped {
		err = errors.Join(err, Start(i))
	}
	return
}

// pathListSeparators are all the characters that can separate paths in
// the clean and purge lists. The entries are relative paths, which
// cannot contain either on any platform, and the defaults are always
// separated with a ':'.
const pathListSeparators = ":;"

func splitPathList(value, separators string) []string {
	if value == "" {
		return nil
	}
	return strings.FieldsFunc(value, func(r rune) bool {
		return strings.ContainsRune(separators, r)
	})
}

// RemovePaths removes all files and directories in list. All paths are
// tried and any errors are joined and returned.
func RemovePaths(i geneos.Instance, list ...string) (err error) {
	h := i.Host()

	// list := filepath.SplitList(paths)
	for _, p := range list {
		// clean path, error on absolute or parent paths, like 'import'
		// walk globbed directories, remove everything
		clean, cerr := geneos.CleanRelativePath(p)
		if cerr != nil {
			return errors.Join(err, fmt.Errorf("%s %w", p, cerr))
		}
		i.Log().Debug("going to remove", slog.String("path", h.Join(i.Home(), clean)))
		// glob here
		m, gerr := h.Glob(h.Join(i.Home(), clean))
		if gerr != nil {
			return errors.Join(err, gerr)
		}
		for _, f := range m {
			i.Log().Debug("trying to RemoveAll", slog.String("path", f))
			if rerr := h.RemoveAll(f); rerr != nil {
				i.Log().Error("failed to remove path", slog.String("path", f), slog.Any("error", rerr))
				err = errors.Join(err, rerr)
				continue
			}
			fmt.Printf("removed %s\n", h.HostPath(f))
		}
	}
	return
}
