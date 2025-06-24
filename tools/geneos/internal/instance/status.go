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

package instance

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// IsDisabled returns true if the instance i is disabled.
func IsDisabled(i geneos.Instance) bool {
	d := ComponentFilepath(i, geneos.DisableExtension)
	if f, err := i.Host().Stat(d); err == nil && f.Mode().IsRegular() {
		return true
	}
	return false
}

// IsProtected returns true if instance i is marked protected
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
// for the instance i.
func BaseVersion(i geneos.Instance) (dir string) {
	t := i.Type().String()
	if i.Type().ParentType != nil && len(i.Type().PackageTypes) > 0 {
		t = i.Type().ParentType.String()
	}
	pkgtype := i.Config().GetString("pkgtype", config.Default(t))
	return i.Host().PathTo("packages", pkgtype, i.Config().GetString("version"))
}

// Version returns the base package name, the underlying package version
// and the actual version in use for the instance i. If base is not a
// link, then base is also returned as the symlink. If there are more
// than 10 levels of symlink then return symlink set to "loop-detected"
// and err set to syscall.ELOOP to prevent infinite loops. If the
// instance is not running or the executable path cannot be determined
// then actual will be returned as "unknown".
func Version(i geneos.Instance) (base string, version string, err error) {
	cf := i.Config()
	base = cf.GetString("version")
	t := i.Type().String()
	if i.Type().ParentType != nil && len(i.Type().PackageTypes) > 0 {
		t = i.Type().ParentType.String()
	}
	pkgtype := cf.GetString("pkgtype", config.Default(t))
	ct := geneos.ParseComponent(pkgtype)

	version, err = geneos.CurrentVersion(i.Host(), ct, base)
	return
}

// CompareVersion returns -1, 0 or +1 if the version of the instance is
// less than, equal or greater than version respectively.
func CompareVersion(i geneos.Instance, version string) int {
	_, iv, err := Version(i)
	if err != nil {
		return -1
	}

	return geneos.CompareVersion(iv, version)
}

// LiveVersion returns the base package name, the underlying package
// version and the actual version in use for the instance i. If base is
// not a link, then base is also returned as the symlink. If there are
// more than 10 levels of symlink then return symlink set to
// "loop-detected" and err set to syscall.ELOOP to prevent infinite
// loops. If the instance is not running or the executable path cannot
// be determined then actual will be returned as "unknown".
func LiveVersion(i geneos.Instance, pid int) (base string, version string, actual string, err error) {
	actual = "unknown"
	cf := i.Config()
	base = cf.GetString("version")

	t := i.Type().String()
	if i.Type().ParentType != nil && len(i.Type().PackageTypes) > 0 {
		t = i.Type().ParentType.String()
	}
	pkgtype := cf.GetString("pkgtype", config.Default(t))
	ct := geneos.ParseComponent(pkgtype)

	version, err = geneos.CurrentVersion(i.Host(), ct, base)
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

	// account for java based components, like webserver, sso-agent and
	// ca3. just return the version the base points to, which may not be
	// true during an update but it's the best we can do.
	if path.Base(actual) == "java" {
		actual = version
		return
	}

	actual = strings.TrimPrefix(actual, i.Host().PathTo("packages", pkgtype)+"/")
	if strings.Contains(actual, "/") {
		actual = actual[:strings.Index(actual, "/")]
	}
	if actual == "" {
		actual = "unknown"
	}
	return
}

// GetPID returns the PID of the process running for the instance. If
// not found then an err of os.ErrProcessDone is returned.
//
// The process is identified by checking the conventions used to start
// Geneos processes. If a component type defines it's own GetPID()
// customer check then that is used instead.
func GetPID(i geneos.Instance) (pid int, err error) {
	return process.GetPID(i.Host(), i.Config().GetString("binary"), i.Type().GetPID, i, i.Name())
}

// GetPIDInfo returns the PID of the process for the instance i along
// with the owner uid and gid and the start time.
func GetPIDInfo(i geneos.Instance) (pid int, owner geneos.FileOwner, mtime time.Time, err error) {
	if pid, err = GetPID(i); err != nil {
		return
	}

	var st os.FileInfo
	st, err = i.Host().Stat(fmt.Sprintf("/proc/%d", pid))
	if err == nil {
		owner = i.Host().GetFileOwner(st)
	}
	return pid, owner, st.ModTime(), err
}
