package config

import (
	"os"
	"path"
)

func UserConfigPath(elem ...string) (p string, err error) {
	confDir, err2 := os.UserConfigDir()
	if err2 != nil {
		return
	}
	p = path.Join(append([]string{confDir}, elem...)...)
	return
}
