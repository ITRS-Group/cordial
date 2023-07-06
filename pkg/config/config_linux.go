package config

import (
	"os"
	"os/user"
	"path"
)

func UserConfigDir(username ...string) (p string, err error) {
	if len(username) == 0 {
		return os.UserConfigDir()
	}
	u, err := user.Lookup(username[0])
	if err != nil {
		return
	}
	p = path.Join(u.HomeDir, ".config")
	return
}
