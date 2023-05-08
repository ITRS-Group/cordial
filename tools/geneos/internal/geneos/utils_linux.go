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

package geneos

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

// FileOwner is only available on Linux localhost
type FileOwner struct {
	Uid uint32
	Gid uint32
}

func (h *Host) GetFileOwner(info fs.FileInfo) (s FileOwner) {
	switch h.GetString("name") {
	case LOCALHOST:
		s.Uid = info.Sys().(*syscall.Stat_t).Uid
		s.Gid = info.Sys().(*syscall.Stat_t).Gid
	default:
		s.Uid = info.Sys().(*sftp.FileStat).UID
		s.Gid = info.Sys().(*sftp.FileStat).GID
	}
	return
}

// WriteConfigFile writes a local configuration file. Tries to be
// atomic, lots of edge cases, UNIX/Linux only. We know the size of
// config structs is typically small, so just marshal in memory
func WriteConfigFile(conf *config.Config, file string, username string, perms fs.FileMode) (err error) {
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
