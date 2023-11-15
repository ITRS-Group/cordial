//go:build !windows

/*
Copyright © 2022 ITRS Group

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
