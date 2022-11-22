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

package host

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
	"github.com/pkg/sftp"
)

// StatX returns extended Stat info. Needs to be deprecated to support
// non LInux platforms
func (h *Host) StatX(name string) (s FileStat, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		if s.St, err = os.Stat(name); err != nil {
			return
		}
		s.Uid = s.St.Sys().(*syscall.Stat_t).Uid
		s.Gid = s.St.Sys().(*syscall.Stat_t).Gid
		s.Mtime = s.St.Sys().(*syscall.Stat_t).Mtim.Sec
	default:
		var sf *sftp.Client
		if sf, err = h.DialSFTP(); err != nil {
			return
		}
		if s.St, err = sf.Stat(name); err != nil {
			return
		}
		s.Uid = s.St.Sys().(*sftp.FileStat).UID
		s.Gid = s.St.Sys().(*sftp.FileStat).GID
		s.Mtime = int64(s.St.Sys().(*sftp.FileStat).Mtime)
	}
	return
}

// try to be atomic, lots of edge cases, UNIX/Linux only
// we know the size of config structs is typically small, so just marshal
// in memory
func WriteConfigFile(file string, username string, perms fs.FileMode, conf *config.Config) (err error) {
	cf := config.New()
	for k, v := range conf.AllSettings() {
		cf.Set(k, v)
	}
	cf.SetConfigFile(file)
	os.MkdirAll(filepath.Dir(file), 0755)
	cf.WriteConfig()

	uid, gid := -1, -1
	if utils.IsSuperuser() {
		if username == "" {
			// try $SUDO_UID etc.
			sudoUID := os.Getenv("SUDO_UID")
			sudoGID := os.Getenv("SUDO_GID")

			if sudoUID != "" && sudoGID != "" {
				if uid, err = strconv.Atoi(sudoUID); err != nil {
					uid = -1
				}

				if gid, err = strconv.Atoi(sudoGID); err != nil {
					gid = -1
				}
			}
		} else {
			uid, gid, _, _ = utils.GetIDs(username)
		}
		os.Chown(file, uid, gid)
		os.Chmod(file, perms)
	}

	return
}
