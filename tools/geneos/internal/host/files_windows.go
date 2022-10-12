package host

import (
	"io/fs"

	"github.com/itrs-group/cordial/pkg/config"
)

func WriteConfigFile(file string, username string, perms fs.FileMode, conf *config.Config) (err error) {
	cf := config.New()
	for k, v := range conf.AllSettings() {
		cf.Set(k, v)
	}
	cf.SetConfigFile(file)
	cf.WriteConfig()

	return
}
