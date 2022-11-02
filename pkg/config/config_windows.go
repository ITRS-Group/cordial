package config

import (
	"os"
	"os/user"
	"path/filepath"
)

func UserConfigDir(username ...string) (path string, err error) {
	if len(username) == 0 {
		return os.UserConfigDir()
	}
	// maybe try looking up appdata instead?
	// https://learn.microsoft.com/en-us/windows/win32/api/shlobj_core/nf-shlobj_core-shgetknownfolderpath
	u, err := user.Lookup(username[0])
	if err != nil {
		return
	}
	path = filepath.Join(u.HomeDir, ".config")
	return
}
