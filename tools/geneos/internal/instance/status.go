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
	"os"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/rs/zerolog/log"
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
	return config.Get[bool](i.Config(), "protected")
}

// IsRunning returns true if the instance is running
func IsRunning(i geneos.Instance) bool {
	return Signal(i, 0) != os.ErrProcessDone
}

// IsAutoStart returns true is the instance is set to autostart
func IsAutoStart(i geneos.Instance) bool {
	return config.Get[bool](i.Config(), "autostart")
}

// BaseVersion returns the absolute path of the base package directory
// for the instance i.
func BaseVersion(i geneos.Instance) (dir string) {
	t := i.Type().String()
	if i.Type().ParentType != nil && len(i.Type().PackageTypes) > 0 {
		t = i.Type().ParentType.String()
	}

	pkgtype := config.Get[string](i.Config(), "pkgtype", config.DefaultValue(t))
	return i.Host().PathTo("packages", pkgtype, config.Get[string](i.Config(), "version"))
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
	base = config.Get[string](cf, "version")
	t := i.Type().String()
	if i.Type().ParentType != nil && len(i.Type().PackageTypes) > 0 {
		t = i.Type().ParentType.String()
	}
	pkgtype := config.Get[string](cf, "pkgtype", config.DefaultValue(t))
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
func LiveVersion(i geneos.Instance, pi *ProcessInfo) (base string, version string, actual string, err error) {
	ct := i.Type()
	cf := i.Config()
	h := i.Host()

	actual = "unknown"

	base = config.Get[string](cf, "version")

	t := ct.String()
	if ct.ParentType != nil && len(ct.PackageTypes) > 0 {
		t = ct.ParentType.String()
	}
	pkgtype := config.Get[string](cf, "pkgtype", config.DefaultValue(t))
	nct := geneos.ParseComponent(pkgtype)

	version, err = geneos.CurrentVersion(h, nct, base)
	if err != nil {
		return
	}

	actual = pi.Exe

	// account for java based components, like webserver, sso-agent and
	// ca3. just return the version the base points to, which may not be
	// true during an update but it's the best we can do.
	if h.Base(actual) == "java" {
		actual = version
		return
	}

	log.Debug().Msgf("instance %s: PID %d, base version %s, actual exe %s", i.Name(), pi.PID, version, actual)

	// check if the exe is a link, and if so resolve it to get the
	// actual version in use. If it's not a link then return the base
	// version as the actual version.
	dir := h.Dir(actual)
	if dir == "" {
		actual = "unknown"
		return
	}
	ls, err := h.Lstat(dir)
	if err != nil {
		actual = "unknown"
		return
	}
	if ls.Mode()&os.ModeSymlink == 0 {
		actual = h.Base(dir)
		return
	}

	// otherwise follow links to get the actual version.
	b, err := h.Readlink(h.Dir(actual))
	if err != nil {
		log.Debug().Err(err).Msgf("instance %s: PID %d, base version %s, exe %s is not a link", i.Name(), pi.PID, version, h.Dir(actual))
		actual = "unknown"
		return
	}
	log.Debug().Msgf("instance %s: PID %d, base version %s, link target %s", i.Name(), pi.PID, version, b)
	actual = h.Base(b)

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
// custom check then that is used instead.
func GetPID(i geneos.Instance) (pid int, err error) {
	return process.PID(i.Host(), config.Get[string](i.Config(), "binary"), []string{i.Name()}, process.CustomChecker(i.Type().GetPID, i))
}

// GetLivePID returns the PID of the process running for the instance i.
// It resets the process cache to ensure the check is live. If not found
// then an err of os.ErrProcessDone is returned.
//
// The process is identified by checking the conventions used to start
// Geneos processes. If a component type defines it's own GetPID()
// custom check then that is used instead.
func GetLivePID(i geneos.Instance) (pid int, err error) {
	return process.PID(i.Host(), config.Get[string](i.Config(), "binary"), []string{i.Name()}, process.CustomChecker(i.Type().GetPID, i), process.RefreshCache())
}

// GetProcessInfo returns process information for the instance i. If the
// process is not found then an err of os.ErrProcessDone is returned. A
// process cache is used to avoid repeated calls to the host to get the
// process entries, which can be expensive. The cache is updated every 5
// seconds, or when the cache is empty.
func GetProcessInfo(i geneos.Instance) (pi *ProcessInfo, err error) {
	pid, err := GetPID(i)
	if err != nil {
		return
	}

	return process.GetProcessInfo[*ProcessInfo](i.Host(), pid, process.FetchLazyFields())
}

// GetChildPIDs returns a list of child processes for the instance
// i. If the process is not found then an err of os.ErrProcessDone is
// returned. A process cache is used to avoid repeated calls to the host
// to get the process entries, which can be expensive. The cache is
// updated every 5 seconds, or when the cache is empty.
func GetChildPIDs(i geneos.Instance) (children []int, err error) {
	pid, err := GetPID(i)
	if err != nil {
		return
	}

	h := i.Host()

	pi, err := process.GetProcessInfo[*ProcessInfo](h, pid, process.FetchLazyFields())
	if err != nil {
		return
	}
	return pi.Children, nil
}
