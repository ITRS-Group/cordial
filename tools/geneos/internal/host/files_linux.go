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
