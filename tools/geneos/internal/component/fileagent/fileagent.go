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

package fileagent

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

const Name = "fa"

var FileAgent = geneos.Component{
	Name:          "fileagent",
	Aliases:       []string{"fileagents", "file-agent"},
	LegacyPrefix:  "fa",
	DownloadBase:  geneos.DownloadBases{Default: "Fix+Analyser+File+Agent", Nexus: "geneos-file-agent"},
	DownloadInfix: "file-agent",

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "7030,7100-",
		config.Join(Name, "clean"): strings.Join([]string{}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{}, ":"),
	},
	PortRange: config.Join(Name, "ports"),
	CleanList: config.Join(Name, "clean"),
	PurgeList: config.Join(Name, "purge"),
	ConfigAliases: map[string]string{
		config.Join(Name, "ports"): Name + "portrange",
		config.Join(Name, "clean"): Name + "cleanlist",
		config.Join(Name, "purge"): Name + "purgelist",
	},

	LegacyParameters: map[string]string{
		"binsuffix":  "binary",
		"fahome":     "home",
		"fabins":     "install",
		"fagentbins": "install",
		"fabase":     "version",
		"fagentbase": "version",
		"faexec":     "program",
		"falogd":     "logdir",
		"fagentlogd": "logdir",
		"falogf":     "logfile",
		"fagentlogf": "logfile",
		"faport":     "port",
		"fagentport": "port",
		"falibs":     "libpaths",
		"fagentlibs": "libpaths",
		"facert":     "certificate",
		"fakey":      "privatekey",
		"fauser":     "user",
		"faopts":     "options",
		"fagentopts": "options",
	},
	Defaults: []string{
		`binary=agent.{{ .os }}_64{{if eq .os "windows"}}.exe{{end}}`,
		`home={{join .root "fileagent" "fileagents" .name}}`,
		`install={{join .root "packages" "fileagent"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=fileagent.log`,
		`port=7030`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:{{join "${config:install}" "${config:version}"}}`,
		`autostart=true`,
	},

	Directories: []string{
		"packages/fileagent",
		"fileagent/fileagents",
	},
}

type FileAgents instance.Instance

// ensure that FileAgents satisfies geneos.Instance interface
var _ geneos.Instance = (*FileAgents)(nil)

func init() {
	FileAgent.Register(factory)
}

var instances sync.Map

func factory(name string) (fileagent geneos.Instance) {
	h, _, local := instance.ParseName(name)
	// _, local, h := instance.SplitName(name, geneos.LOCAL)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}
	if f, ok := instances.Load(h.FullName(local)); ok {
		if fa, ok := f.(*FileAgents); ok {
			return fa
		}
	}

	fileagent = &FileAgents{
		Component:    &FileAgent,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(fileagent, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", fileagent)
	}
	// set the home dir based on where it might be, default to one above
	fileagent.Config().Set("home", instance.Home(fileagent))
	instances.Store(h.FullName(local), fileagent)

	return
}

// interface method set

// Return the Component for an Instance
func (n *FileAgents) Type() *geneos.Component {
	return n.Component
}

func (n *FileAgents) Name() string {
	if n.Config() == nil {
		return ""
	}
	return n.Config().GetString("name")
}

func (n *FileAgents) Home() string {
	return instance.Home(n)
}

func (n *FileAgents) Host() *geneos.Host {
	return n.InstanceHost
}

func (n *FileAgents) String() string {
	return instance.DisplayName(n)
}

func (n *FileAgents) Load() (err error) {
	return instance.LoadConfig(n)
}

func (n *FileAgents) Unload() (err error) {
	instances.Delete(n.Name() + "@" + n.Host().String())
	n.ConfigLoaded = time.Time{}
	return
}

func (n *FileAgents) Loaded() time.Time {
	return n.ConfigLoaded
}

func (n *FileAgents) SetLoaded(t time.Time) {
	n.ConfigLoaded = t
}

func (n *FileAgents) Config() *config.Config {
	return n.Conf
}

func (n *FileAgents) Add(tmpl string, port uint16, noCerts bool) (err error) {
	if port == 0 {
		port = instance.NextFreePort(n.Host(), &FileAgent)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	n.Config().Set("port", port)

	if err = instance.SaveConfig(n); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// create certs, report success only
	if !noCerts {
		instance.NewCertificate(n).Report(os.Stdout, responses.StderrWriter(io.Discard))
	}

	// default config XML etc.
	return nil
}

func (i *FileAgents) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	home = i.Home()
	logFile := instance.LogFilePath(i)
	checks = append(checks, filepath.Dir(logFile))
	args = []string{
		i.Name(),
		"-port", i.Config().GetString("port"),
	}
	if instance.CompareVersion(i, "6.6.0") >= 0 {
		// secureArgs := instance.SetSecureArgs(i)
		secureArgs, secureEnv, fileChecks, err := instance.SecureArgs(i)
		if err != nil {
			return nil, nil, "", err
		}
		args = append(args, secureArgs...)
		env = append(env, secureEnv...)
		checks = append(checks, fileChecks...)
		// for _, arg := range secureArgs {
		// 	if !strings.HasPrefix(arg, "-") {
		// 		checks = append(checks, arg)
		// 	}
		// }
	}
	env = append(env, "LOG_FILENAME="+logFile)

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (c *FileAgents) Reload() (err error) {
	return geneos.ErrNotSupported
}

func (c *FileAgents) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
