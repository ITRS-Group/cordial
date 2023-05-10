package geneos

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"
)

var credentials sync.Map

const UserCredsFile = "credentials"

func LoadCredentials() {
	// note that SetAppName only matters when PromoteFile returns an empty path
	cr, _ := config.Load("credentials",
		config.SetAppName("geneos"),
		config.UseDefaults(false),
		config.IgnoreWorkingDir(),
	)

	credentials = sync.Map{}
	for _, cred := range cr.GetStringMap("credentials") {
		cf := config.New()
		switch m := cred.(type) {
		case map[string]interface{}:
			cf.MergeConfigMap(m)
		default:
			log.Debug().Msgf("credentials value not a map[string]interface{} but a %T", cred)
			continue
		}
		credentials.Store(cf.GetString("name"), cf)
	}
}

func SaveCredentials() (err error) {
	c := config.New()

	credentials.Range(func(k, v interface{}) bool {
		name := k.(string)
		switch v := v.(type) {
		case *config.Config:
			name = strings.ReplaceAll(name, ".", "-")
			c.Set("credentials."+name, v.AllSettings())
		}
		return true
	})

	userhostfile := UserHostsFilePath()

	if err := os.MkdirAll(filepath.Dir(userhostfile), 0775); err != nil {
		return err
	}
	if err := c.WriteConfigAs(userhostfile); err != nil {
		return err
	}
	return nil
}
