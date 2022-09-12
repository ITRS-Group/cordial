package geneos

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
)

var (
	ErrInvalidArgs  error = errors.New("invalid arguments")
	ErrNotSupported error = errors.New("not supported")
	ErrDisabled     error = errors.New("disabled")
)

const RootCAFile = "rootCA"
const SigningCertFile = "geneos"
const DisableExtension = "disabled"

var ConfigFileName = "geneos"
var UserConfigFile = "geneos.json"
var ConfigFileType = "json"
var GlobalConfigDir = "/etc/geneos"
var GlobalConfigPath = filepath.Join(GlobalConfigDir, UserConfigFile)

// initialise a Geneos environment.
//
// creates a directory hierarchy and calls the initialisation
// functions for each component, for example to create templates
//
// if the directory is not empty and 'noEmptyOK' is false then
// nothing is changed
func Init(r *host.Host, options ...GeneosOptions) (err error) {
	var uid, gid int

	if r != host.LOCAL && utils.IsSuperuser() {
		err = ErrNotSupported
		return
	}

	opts := EvalOptions(options...)
	if opts.homedir == "" {
		logError.Fatalln("homedir not set")
		// default or error
	}

	// dir must first not exist (or be empty) and then be creatable
	//
	// maybe check that the entire list of registered directories are
	// either directories or do not exist
	if _, err := r.Stat(opts.homedir); err != nil {
		if err = r.MkdirAll(opts.homedir, 0775); err != nil {
			logError.Fatalln(err)
		}
	} else if !opts.overwrite {
		// check empty
		dirs, err := r.ReadDir(opts.homedir)
		if err != nil {
			logError.Fatalln(err)
		}
		for _, entry := range dirs {
			if !strings.HasPrefix(entry.Name(), ".") {
				if r != host.LOCAL {
					logDebug.Println("remote directories exist, exiting init")
					return nil
				}
				logError.Fatalf("target directory %q exists and is not empty", opts.homedir)
			}
		}
	}

	if r == host.LOCAL {
		config.GetConfig().Set("geneos", opts.homedir)
		config.GetConfig().Set("defaultuser", opts.username)

		if utils.IsSuperuser() {
			if err = host.LOCAL.WriteConfigFile(GlobalConfigPath, "root", 0664, config.GetConfig().AllSettings()); err != nil {
				logError.Fatalln("cannot write global config", err)
			}
		} else {
			userConfFile := UserConfigFilePath()

			if err = host.LOCAL.WriteConfigFile(userConfFile, opts.username, 0664, config.GetConfig().AllSettings()); err != nil {
				return err
			}
		}
	}

	// recreate host.LOCAL to load "geneos" and others
	host.LOCAL = nil
	host.LOCAL = host.Get(host.LOCALHOST)

	if utils.IsSuperuser() {
		uid, gid, _, err = utils.GetIDs(opts.username)
		if err != nil {
			// do something
		}
		if err = host.LOCAL.Chown(opts.homedir, uid, gid); err != nil {
			logError.Fatalln(err)
		}
	}

	// it's not an error to try to re-create existing dirs
	if err = MakeComponentDirs(host.LOCAL, nil); err != nil {
		return
	}

	// if we've created directory paths as root, go through and change
	// ownership to the tree
	if utils.IsSuperuser() {
		err = filepath.WalkDir(opts.homedir, func(path string, dir fs.DirEntry, err error) error {
			if err == nil {
				err = host.LOCAL.Chown(path, uid, gid)
			}
			return err
		})
	}

	for _, c := range AllComponents() {
		if c.Initialise != nil {
			c.Initialise(host.LOCAL, c)
		}
	}

	return
}

// read a local configuration file without the need for a host
// connection, primarily for bootstrapping
func ReadLocalConfigFile(file string, config interface{}) (err error) {
	jsonFile, err := os.ReadFile(file)
	if err != nil {
		return
	}
	// dec := json.NewDecoder(jsonFile)
	return json.Unmarshal(jsonFile, &config)
}

func UserConfigFilePath() string {
	userConfDir, err := os.UserConfigDir()
	if err != nil {
		logError.Fatalln(err)
	}
	return filepath.Join(userConfDir, UserConfigFile)
}
