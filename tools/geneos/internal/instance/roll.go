/*
Copyright © 2025 ITRS Group

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

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/rs/zerolog/log"
)

const lockDirSuffix = "lck"

// RollFiles atomically rolls files indicated by params with the given
// suffixes for the instance. The newSuffix files are moved to the
// active files, and existing active files are backed up with the
// oldSuffix. If an error occurs, any files that were successfully
// rolled are unrolled back to their original state.
//
// Each file is rolled individually and atomically using a lock directory
// to prevent concurrent modifications. If any file fails to roll, the
// function attempts to unroll (revert) all previously rolled files in
// this operation by calling itself with swapped suffixes and the list
// of completed files.
// A lock directory is created for each file, but not for the whole
// operation.
func RollFiles(i geneos.Instance, newSuffix, oldSuffix string, params ...string) (err error) {
	var done []string

	for _, p := range params {
		log.Debug().Str("instance", i.String()).Str("file", p).Msg("rolling file")
		if err = rollOneFileParam(i, p, newSuffix, oldSuffix); err != nil {
			break
		}
		done = append(done, p)
	}

	// unroll on error, just reversing the done list and the prefixes
	if len(done) != len(params) {
		_ = RollFiles(i, oldSuffix, newSuffix, done...)
		return fmt.Errorf("failed to roll files for %s: %w", i, err)
	}

	return nil
}

// rollOneFileParam tries to roll a single file for an instance. A lock
// directory is created to avoid concurrent rolls. The new file is
// linked into place, and the existing file is backed up.
//
// If the file does not already exist then on error the files should be
// left as they were.
func rollOneFileParam(i geneos.Instance, param, newSuffix, oldSuffix string) (err error) {
	h := i.Host()
	// move new files to active files, backup existing ones

	noOriginal := false

	path := PathTo(i, param)
	lockPath := path + "." + lockDirSuffix
	newPath := path + "." + newSuffix
	oldPath := path + "." + oldSuffix

	// basic checks

	// if there is no original file, then record that for later
	if _, err = h.Stat(path); errors.Is(err, os.ErrNotExist) {
		log.Debug().Str("instance", i.String()).Str("file", path).Msg("no original file to backup")
		noOriginal = true
	}

	// check that new file exists
	if _, err = h.Stat(newPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no .%q file found for %s", newSuffix, i)
	}

	// create lock directory or fail if it already exists
	if err = h.Mkdir(lockPath, 0o700); err != nil {
		return fmt.Errorf("could not create lock for %s: %w", i, err)
	}
	defer h.Remove(lockPath)
	log.Debug().Str("instance", i.String()).Str("lock", lockPath).Msg("acquired lock")

	// only backup original file if it exists
	if !noOriginal {
		// backup existing files. On UNIX Rename is atomic, but not on all systems
		log.Debug().Str("instance", i.String()).Str("file", path).Str("backup", oldPath).Msg("backing up original file")
		if err = h.Rename(path, oldPath); err != nil {
			return
		}
	}

	// atomically link new files into place
	log.Debug().Str("instance", i.String()).Str("file", newPath).Msg("linking new file into place")
	if err = h.Link(newPath, path); err != nil {
		log.Error().Err(err).Str("instance", i.String()).Str("file", newPath).Msg("failed to link new file into place")
		// try to restore the original file, if it existed
		if !noOriginal {
			log.Debug().Str("instance", i.String()).Str("file", oldPath).Msg("restoring original file")
			err2 := h.Rename(oldPath, path)
			if err2 != nil {
				err = fmt.Errorf("%v; additionally failed to restore original file: %w", err, err2)
			}
		}
		return
	}

	log.Debug().Str("instance", i.String()).Str("file", path).Msg("rolled file successfully")
	// remove files with new suffix
	return h.Remove(newPath)
}
