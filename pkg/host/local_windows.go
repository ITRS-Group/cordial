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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/Microsoft/go-winio"
	"github.com/pkg/sftp"
	"golang.org/x/sys/windows"
)

// GetFileOwner TODO: Document its behavior, inputs, and effects.
func GetFileOwner(h Host, info fs.FileInfo) (uid, gid int) {
	uid, gid = -1, -1
	if h == nil || h.IsLocalhost() || info == nil {
		return
	}
	if stat, ok := info.Sys().(*sftp.FileStat); ok {
		uid = int(stat.UID)
		gid = int(stat.GID)
	}
	return
}

var setProcessAffinityMask = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetProcessAffinityMask")

func procSetupOS(cmd *exec.Cmd, out *os.File, options ...ProcessOption) (err error) {
	po := evalProcessOptions(options...)

	if po.detach {
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &windows.SysProcAttr{}
		}
		cmd.SysProcAttr.CreationFlags = windows.CREATE_NEW_PROCESS_GROUP
	}
	return
}

func postStart(cmd *exec.Cmd, options ...ProcessOption) (err error) {
	po := evalProcessOptions(options...)
	if len(po.cpuAffinity) == 0 {
		return nil
	}
	if cmd.Process == nil {
		return errors.New("process has not been started")
	}
	return setProcAffinity(cmd.Process.Pid, po.cpuAffinity...)
}

func setProcAffinity(pid int, cpus ...int) error {
	if len(cpus) == 0 {
		return nil
	}
	var mask uintptr
	for _, cpu := range cpus {
		if cpu < 0 || cpu >= runtime.NumCPU() || cpu >= 8*int(unsafe.Sizeof(mask)) {
			return fmt.Errorf("CPU index %d is outside the process affinity mask", cpu)
		}
		mask |= uintptr(1) << uint(cpu)
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(process)
	r1, _, callErr := setProcessAffinityMask.Call(uintptr(process), mask)
	if r1 == 0 {
		if callErr != syscall.Errno(0) {
			return callErr
		}
		return syscall.EINVAL
	}
	return nil
}

// Lchtimes TODO: Document its behavior, inputs, and effects.
func (h *Local) Lchtimes(path string, atime time.Time, mtime time.Time) (err error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(
		name,
		windows.FILE_WRITE_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	accessTime := windows.NsecToFiletime(atime.UnixNano())
	modifiedTime := windows.NsecToFiletime(mtime.UnixNano())
	return windows.SetFileTime(handle, nil, &accessTime, &modifiedTime)
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
