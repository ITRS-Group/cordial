package host

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/itrs-group/cordial/pkg/config"
)

func WriteConfigFile(file string, username string, perms fs.FileMode, conf *config.Config) (err error) {
	cf := config.New()
	for k, v := range conf.AllSettings() {
		cf.Set(k, v)
	}
	cf.SetConfigFile(file)
	os.MkdirAll(filepath.Dir(file), 0755)
	cf.WriteConfig()

	return
}
