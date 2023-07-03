package instance

import (
	"os"

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
	return c.Host().Filepath("packages", c.Type().String(), c.Config().GetString("version"))
}

// Version returns the base package name and the underlying package
// version for the instance c. If base is not a link, then base is also
// returned as the symlink. If there are more than 10 levels of symlink
// then return symlink set to "loop-detected" and err set to
// syscall.ELOOP to prevent infinite loops.
func Version(c geneos.Instance) (base string, version string, err error) {
	cf := c.Config()
	base = cf.GetString("version")
	pkgtype := cf.GetString("pkgtype")
	ct := c.Type()
	if pkgtype != "" {
		ct = geneos.FindComponent(pkgtype)
	}
	version, err = geneos.CurrentVersion(c.Host(), ct, cf.GetString("version"))
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
