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
func IsDisabled(c geneos.Instance) bool {
	d := ComponentFilepath(c, geneos.DisableExtension)
	if f, err := c.Host().Stat(d); err == nil && f.Mode().IsRegular() {
		return true
	}
	return false
}

// IsProtected returns true if instance c is marked protected
func IsProtected(c geneos.Instance) bool {
	return c.Config().GetBool("protected")
}

// IsRunning returns true if the instance is running
func IsRunning(c geneos.Instance) bool {
	_, err := GetPID(c)
	return err != os.ErrProcessDone
}

// IsAutoStart returns true is the instance is set to autostart
func IsAutoStart(c geneos.Instance) bool {
	return c.Config().GetBool("autostart")
}

// BaseVersion returns the absolute path of the base package directory
// for the instance c. No longer references the instance "install" parameter.
func BaseVersion(c geneos.Instance) (dir string) {
	t := c.Type().String()
	if c.Type().ParentType != nil {
		t = c.Type().ParentType.String()
	}
	pkgtype := c.Config().GetString("pkgtype", config.Default(t))
	return c.Host().PathTo("packages", pkgtype, c.Config().GetString("version"))
}

// Version returns the base package name, the underlying package version
// and the actual version in use for the instance c. If base is not a
// link, then base is also returned as the symlink. If there are more
// than 10 levels of symlink then return symlink set to "loop-detected"
// and err set to syscall.ELOOP to prevent infinite loops. If the
// instance is not running or the executable path cannot be determined
// then actual will be returned as "unknown".
func Version(c geneos.Instance) (base string, version string, err error) {
	cf := c.Config()
	base = cf.GetString("version")
	t := c.Type().String()
	if c.Type().ParentType != nil {
		t = c.Type().ParentType.String()
	}
	pkgtype := cf.GetString("pkgtype", config.Default(t))
	ct := geneos.ParseComponent(pkgtype)

	version, err = geneos.CurrentVersion(c.Host(), ct, cf.GetString("version"))
	return
}

// LiveVersion returns the base package name, the underlying package version
// and the actual version in use for the instance c. If base is not a
// link, then base is also returned as the symlink. If there are more
// than 10 levels of symlink then return symlink set to "loop-detected"
// and err set to syscall.ELOOP to prevent infinite loops. If the
// instance is not running or the executable path cannot be determined
// then actual will be returned as "unknown".
func LiveVersion(c geneos.Instance, pid int) (base string, version string, actual string, err error) {
	actual = "unknown"
	cf := c.Config()
	base = cf.GetString("version")

	t := c.Type().String()
	if c.Type().RelatedTypes != nil && c.Type().ParentType != nil {
		t = c.Type().ParentType.String()
	}
	pkgtype := cf.GetString("pkgtype", config.Default(t))
	ct := geneos.ParseComponent(pkgtype)

	version, err = geneos.CurrentVersion(c.Host(), ct, cf.GetString("version"))
	if err != nil {
		return
	}

	// This is the path on the target host, and only linux is supported anyway
	actual, err = c.Host().Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		actual = "unknown"
		return
	}
	log.Debug().Msgf("actual=%s pkgtype=%s", actual, pkgtype)
	actual = strings.TrimPrefix(actual, c.Host().PathTo("packages", pkgtype)+"/")
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
func AtLeastVersion(c geneos.Instance, version string) bool {
	_, iv, err := Version(c)
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
func GetPID(c geneos.Instance) (pid int, err error) {
	return process.GetPID(c.Host(), c.Config().GetString("binary"), c.Type().GetPID, c, c.Name())
}

// GetPIDInfo returns the PID of the process for the instance c along
// with the owner uid and gid and the start time.
func GetPIDInfo(c geneos.Instance) (pid int, uid int, gid int, mtime time.Time, err error) {
	if pid, err = GetPID(c); err != nil {
		return
	}

	var st os.FileInfo
	st, err = c.Host().Stat(fmt.Sprintf("/proc/%d", pid))
	s := c.Host().GetFileOwner(st)
	return pid, s.Uid, s.Gid, st.ModTime(), err
}
