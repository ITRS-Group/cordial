//go:build !windows

/*
Copyright © 2022 ITRS Group

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

package config

import (
	"os"
	"os/user"
	"path"
)

// UserConfigDir returns the configuration directory for username, or is
// none given then the current user. If os.UserConfigDir() fails then we
// lookup the user and return a path relative to the homedir (which
// works around empty environments)
func UserConfigDir(username ...string) (confdir string, err error) {
	if len(username) == 0 {
		if confdir, err = os.UserConfigDir(); err == nil {
			return
		}
		u, err := user.Current()
		if err != nil {
			return confdir, err
		}
		confdir = path.Join(u.HomeDir, ".config")
		return confdir, nil
	}
	u, err := user.Lookup(username[0])
	if err != nil {
		return
	}
	confdir = path.Join(u.HomeDir, ".config")
	return
}
