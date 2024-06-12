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

package cmd

import (
	"errors"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"
)

func buildFileLookupTable(dv *config.Config, path, pattern string) (lookup map[string]string, skip bool) {
	fullpath, err := filepath.Abs(path)
	if err != nil {
		fullpath = path
	}
	lookup = map[string]string{
		"fullpath": fullpath,
		"path":     path,
		"pattern":  pattern,
		"filename": filepath.Base(path),
		"status":   "OK",
	}
	st, err := os.Lstat(path)
	if err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if slices.Contains(dv.GetStringSlice("ignore-file-errors"), "match") {
				skip = true
				return
			}
			lookup["status"] = "NOT_FOUND"
		case errors.Is(err, fs.ErrPermission):
			if slices.Contains(dv.GetStringSlice("ignore-file-errors"), "access") {
				skip = true
				return
			}
			lookup["status"] = "ACCESS_DENIED"
		case errors.Is(err, fs.ErrInvalid):
			if slices.Contains(dv.GetStringSlice("ignore-file-errors"), "other") {
				skip = true
				return
			}
			lookup["status"] = "INVALID"
		default:
			if slices.Contains(dv.GetStringSlice("ignore-file-errors"), "other") {
				skip = true
				return
			}
			lookup["status"] = "UNKNOWN_ERROR"
		}
		return
	}
	lookup["modtime"] = st.ModTime().Format(time.RFC3339)
	lookup["size"] = strconv.FormatInt(st.Size(), 10)

	mode := st.Mode()

	lookup["mode"] = mode.String()
	types := dv.GetStringSlice("types", config.Default([]string{"file", "directory", "symlink", "other"}))
	switch {
	case mode.IsDir():
		if !slices.Contains(types, "directory") {
			skip = true
			return
		}
		lookup["type"] = "directory"
	case mode.IsRegular():
		if !slices.Contains(types, "file") {
			skip = true
			return
		}
		lookup["type"] = "file"
	case mode&fs.ModeSymlink != 0:
		if !slices.Contains(types, "symlink") {
			skip = true
			return
		}
		lookup["type"] = "symlink"
		if target, err := os.Readlink(path); err == nil {
			lookup["target"] = target
		}
	default:
		if !slices.Contains(types, "other") {
			skip = true
			return
		}
		lookup["type"] = "other"
	}
	uid := strconv.Itoa(int(st.Sys().(*syscall.Stat_t).Uid))
	lookup["uid"] = uid
	lookup["user"] = uid
	if u, err := user.LookupId(uid); err == nil { // no error
		lookup["user"] = u.Username
	} else {
		log.Error().Err(err).Msg("")
	}
	gid := strconv.Itoa(int(st.Sys().(*syscall.Stat_t).Gid))
	lookup["gid"] = gid
	lookup["group"] = gid
	if g, err := user.LookupGroupId(gid); err == nil { //no error
		lookup["group"] = g.Name
	}

	device := strconv.FormatUint(st.Sys().(*syscall.Stat_t).Dev, 10)
	lookup["device"] = device
	inode := strconv.FormatUint(st.Sys().(*syscall.Stat_t).Ino, 10)
	lookup["inode"] = inode

	return
}
