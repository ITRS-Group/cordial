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

package host

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

func procSetupOS(cmd *exec.Cmd, out *os.File, detach bool) (err error) {
	if detach {
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &windows.SysProcAttr{}
		}
		cmd.SysProcAttr.CreationFlags = windows.CREATE_NEW_PROCESS_GROUP
	}
	return
}

func (h *Local) Lchtimes(path string, atime time.Time, mtime time.Time) (err error) {
	return
}

// Symlink on Windows will use a junction if the target is a directory,
// to avoid extra privileges. It falls back to os.Symlink if the target
// is not a directory.
//
// core code from https://github.com/nyaosorg/go-windows-junction (MIT
// License)
func (h *Local) Symlink(oldname, newname string) (err error) {
	_target := oldname
	if !filepath.IsAbs(oldname) {
		// target should be relative to newname's parent
		_target = filepath.Join(filepath.Dir(newname), oldname)
	}
	st, err := os.Stat(_target)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return os.Symlink(oldname, newname)
	}
	_mountPt, err := windows.UTF16PtrFromString(newname)
	if err != nil {
		return fmt.Errorf("%s: %s", newname, err)
	}

	err = os.Mkdir(newname, 0777)
	if err != nil {
		return fmt.Errorf("%s: %s", newname, err)
	}
	ok := false
	defer func() {
		if !ok {
			os.Remove(newname)
		}
	}()

	handle, err := windows.CreateFile(_mountPt,
		windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0)
	if err != nil {
		return fmt.Errorf("%s: %s", newname, err)
	}
	defer windows.CloseHandle(handle)

	rp := winio.ReparsePoint{
		Target:       _target,
		IsMountPoint: true,
	}

	data := winio.EncodeReparsePoint(&rp)

	var size uint32

	err = windows.DeviceIoControl(
		handle,
		windows.FSCTL_SET_REPARSE_POINT,
		&data[0],
		uint32(len(data)),
		nil,
		0,
		&size,
		nil)

	if err != nil {
		return fmt.Errorf("windows.DeviceIoControl: %s", err)
	}
	ok = true
	return nil
}
