package config

import (
	"os"
	"path"
)

// UserConfigPath returns the operating system's user configuration path with
// elem appended.
func UserConfigPath(elem ...string) (p string, err error) {
	confDir, err2 := os.UserConfigDir()
	if err2 != nil {
		return
	}
	p = path.Join(append([]string{confDir}, elem...)...)
	return
}
