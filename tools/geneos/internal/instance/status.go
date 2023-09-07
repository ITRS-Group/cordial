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

package instance

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// IsDisabled returns true if the instance c is disabled.
func IsDisabled(i geneos.Instance) bool {
	d := ComponentFilepath(i, geneos.DisableExtension)
	if f, err := i.Host().Stat(d); err == nil && f.Mode().IsRegular() {
		return true
	}
	return false
}

// IsProtected returns true if instance c is marked protected
func IsProtected(i geneos.Instance) bool {
	return i.Config().GetBool("protected")
}

// IsRunning returns true if the instance is running
func IsRunning(i geneos.Instance) bool {
	_, err := GetPID(i)
	return err != os.ErrProcessDone
}

// IsAutoStart returns true is the instance is set to autostart
func IsAutoStart(i geneos.Instance) bool {
	return i.Config().GetBool("autostart")
}

// BaseVersion returns the absolute path of the base package directory
// for the instance c.
func BaseVersion(i geneos.Instance) (dir string) {
	t := i.Type().String()
	if i.Type().ParentType != nil && len(i.Type().RelatedTypes) > 0 {
		t = i.Type().ParentType.String()
	}
	pkgtype := i.Config().GetString("pkgtype", config.Default(t))
	return i.Host().PathTo("packages", pkgtype, i.Config().GetString("version"))
}

// Version returns the base package name, the underlying package version
// and the actual version in use for the instance c. If base is not a
// link, then base is also returned as the symlink. If there are more
// than 10 levels of symlink then return symlink set to "loop-detected"
// and err set to syscall.ELOOP to prevent infinite loops. If the
// instance is not running or the executable path cannot be determined
// then actual will be returned as "unknown".
func Version(i geneos.Instance) (base string, version string, err error) {
	cf := i.Config()
	base = cf.GetString("version")
	t := i.Type().String()
	if i.Type().ParentType != nil && len(i.Type().RelatedTypes) > 0 {
		t = i.Type().ParentType.String()
	}
	pkgtype := cf.GetString("pkgtype", config.Default(t))
	ct := geneos.ParseComponent(pkgtype)

	version, err = geneos.CurrentVersion(i.Host(), ct, cf.GetString("version"))
	return
}

// LiveVersion returns the base package name, the underlying package version
// and the actual version in use for the instance c. If base is not a
// link, then base is also returned as the symlink. If there are more
// than 10 levels of symlink then return symlink set to "loop-detected"
// and err set to syscall.ELOOP to prevent infinite loops. If the
// instance is not running or the executable path cannot be determined
// then actual will be returned as "unknown".
func LiveVersion(i geneos.Instance, pid int) (base string, version string, actual string, err error) {
	actual = "unknown"
	cf := i.Config()
	base = cf.GetString("version")

	t := i.Type().String()
	if i.Type().ParentType != nil && len(i.Type().RelatedTypes) > 0 {
		t = i.Type().ParentType.String()
	}
	pkgtype := cf.GetString("pkgtype", config.Default(t))
	ct := geneos.ParseComponent(pkgtype)

	version, err = geneos.CurrentVersion(i.Host(), ct, cf.GetString("version"))
	if err != nil {
		return
	}

	// This is the path on the target host, and only linux is supported anyway
	actual, err = i.Host().Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		actual = "unknown"
		return
	}
	log.Debug().Msgf("actual=%s pkgtype=%s", actual, pkgtype)
	actual = strings.TrimPrefix(actual, i.Host().PathTo("packages", pkgtype)+"/")
	if strings.Contains(actual, "/") {
		actual = actual[:strings.Index(actual, "/")]
	}
	if actual == "" {
		actual = "unknown"
	}
	return
}

// AtLeastVersion returns true if the installed version for instance c
// is version or greater. If the version of the instance is somehow
// unparseable then this returns false.
func AtLeastVersion(i geneos.Instance, version string) bool {
	_, iv, err := Version(i)
	if err != nil {
		return false
	}
	return geneos.CompareVersion(iv, version) >= 0
}

// GetPID returns the PID of the process running for the instance. If
// not found then an err of os.ErrProcessDone is returned.
//
// The process is identified by checking the conventions used to start
// Geneos processes.
func GetPID(i geneos.Instance) (pid int, err error) {
	return process.GetPID(i.Host(), i.Config().GetString("binary"), i.Type().GetPID, i, i.Name())
}

// GetPIDInfo returns the PID of the process for the instance c along
// with the owner uid and gid and the start time.
func GetPIDInfo(i geneos.Instance) (pid int, uid int, gid int, mtime time.Time, err error) {
	if pid, err = GetPID(i); err != nil {
		return
	}

	var st os.FileInfo
	st, err = i.Host().Stat(fmt.Sprintf("/proc/%d", pid))
	s := i.Host().GetFileOwner(st)
	return pid, s.Uid, s.Gid, st.ModTime(), err
}
