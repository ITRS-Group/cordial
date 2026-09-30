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

// UserConfigPath returns the configuration path for the specified
// subdirectory and/or file made up of joining the elements of [elem]
// using [path.Join]. If [os.UserConfigPath] fails then return a path
// relative to the homedir (which works around empty environments). If
// [user.Current] - used to get the home directory - fails then an empty
// path and an error are returned.
func UserConfigPath(elem ...string) (confdir string, err error) {
	if confdir, err = os.UserConfigDir(); err == nil {
		confdir = path.Join(append([]string{confdir}, elem...)...)
		return
	}
	// fallback to home directory if os.UserConfigDir fails
	u, err := user.Current()
	if err != nil {
		return confdir, err
	}

	confdir = path.Join(append([]string{u.HomeDir, ".config"}, elem...)...)
	return confdir, nil
}
