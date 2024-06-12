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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/hectane/go-acl/api"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
)

func buildFileLookupTable(dv *config.Config, path, pattern string) (lookup map[string]string, skip bool) {
	fullpath, err := filepath.Abs(path)
	if err != nil {
		fullpath = path
	}
	lookup = map[string]string{
		"fullpath": fullpath,
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

	var (
		owner   *windows.SID
		secDesc windows.Handle
	)

	err = api.GetNamedSecurityInfo(
		path,
		api.SE_FILE_OBJECT,
		api.OWNER_SECURITY_INFORMATION,
		&owner,
		nil,
		nil,
		nil,
		&secDesc,
	)
	if err == nil {
		defer windows.LocalFree(secDesc)
		lookup["sid"] = owner.String()
		lookup["owner"] = owner.String()
		if account, domain, accType, err := owner.LookupAccount(""); err == nil {
			log.Debug().Msgf("account/domain/accType: %s/%s/%v", account, domain, accType)
			lookup["owner"] = domain + "\\" + account
		}
	}

	info, err := fileinfoWindows(path)
	log.Debug().Msgf("info: %#v", info)
	lookup["device"] = fmt.Sprintf("0x%08X", info.VolumeSerialNumber)
	lookup["index"] = fmt.Sprintf("0x%08X%08X", info.FileIndexHigh, info.FileIndexLow)

	return
}

func fileinfoWindows(name string) (info syscall.ByHandleFileInformation, err error) {
	if len(name) == 0 {
		err = &os.PathError{Op: "fileinfoWindows", Path: name, Err: syscall.Errno(syscall.ERROR_PATH_NOT_FOUND)}
		return
	}
	namep, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		err = &os.PathError{Op: "fileinfoWindows", Path: name, Err: err}
		return
	}
	h, err := syscall.CreateFile(namep, 0, 0, nil,
		syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		err = &os.PathError{Op: "CreateFile", Path: name, Err: err}
		return
	}
	defer syscall.CloseHandle(h)
	err = syscall.GetFileInformationByHandle(h, &info)
	return
}
