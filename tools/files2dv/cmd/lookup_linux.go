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

package cmd

import (
	"errors"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"
)

func buildFileLookupTable(dv *config.Config, path string, filetypes map[string]bool) (lookup map[string]string, skip bool) {
	lookup = map[string]string{
		"path":     path,
		"filename": filepath.Base(path),
		"status":   "OK",
	}
	st, err := os.Lstat(path)
	if err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if dv.GetBool("ignore-file-errors.exist") {
				skip = true
				return
			}
			lookup["status"] = "NOT_FOUND"
		case errors.Is(err, fs.ErrPermission):
			if dv.GetBool("ignore-file-errors.access") {
				skip = true
				return
			}
			lookup["status"] = "ACCESS_DENIED"
		case errors.Is(err, fs.ErrInvalid):
			if dv.GetBool("ignore-file-errors.other") {
				skip = true
				return
			}
			lookup["status"] = "INVALID"
		default:
			if dv.GetBool("ignore-file-errors.other") {
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
	switch {
	case mode.IsDir():
		if !filetypes["directory"] {
			skip = true
			return
		}
		lookup["type"] = "directory"
	case mode.IsRegular():
		if !filetypes["file"] {
			skip = true
			return
		}
		lookup["type"] = "file"
	case mode&fs.ModeSymlink != 0:
		if !filetypes["symlink"] {
			skip = true
			return
		}
		lookup["type"] = "symlink"
		if target, err := os.Readlink(path); err == nil {
			lookup["target"] = target
		}
	default:
		if !filetypes["other"] {
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
