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
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const name = "fa"

var Component = geneos.Component{
	Name:          "fileagent",
	Aliases:       []string{"fileagents", "file-agent"},
	LegacyPrefix:  "fa",
	DownloadBase:  geneos.DownloadBases{Default: "Fix+Analyser+File+Agent", Nexus: "geneos-file-agent"},
	DownloadInfix: "file-agent",

	GlobalSettings: map[string]string{
		config.Join(name, "ports"): "7030,7100-",
		config.Join(name, "clean"): strings.Join([]string{}, ":"),
		config.Join(name, "purge"): strings.Join([]string{}, ":"),
	},
	PortRange: config.Join(name, "ports"),
	CleanList: config.Join(name, "clean"),
	PurgeList: config.Join(name, "purge"),
	ConfigAliases: map[string]string{
		config.Join(name, "ports"): name + "portrange",
		config.Join(name, "clean"): name + "cleanlist",
		config.Join(name, "purge"): name + "purgelist",
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

type FileAgent instance.Instance

// ensure that FileAgents satisfies geneos.Instance interface
var _ geneos.Instance = (*FileAgent)(nil)

func init() {
	Component.Register(factory)
}

var fileagents sync.Map

func factory(name string) (i geneos.Instance) {
	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h.IsLocalhost() && geneos.LocalRoot() == "") {
		return nil
	}
	if f, ok := fileagents.Load(h.FullName(local)); ok {
		if fa, ok := f.(*FileAgent); ok {
			return fa
		}
	}

	i = &FileAgent{
		Component:    &Component,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(i, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", i, err))
	}
	// set the home dir based on where it might be, default to one above
	config.Set(i.Config(), "home", instance.Home(i))
	i.(*FileAgent).Logger = instance.Logger(i)
	i.(*FileAgent).AuditLogger = instance.AuditLogger(i)
	fileagents.Store(h.FullName(local), i)

	return
}

// interface method set

// Return the Component for an Instance
func (i *FileAgent) Type() *geneos.Component {
	return i.Component
}

func (i *FileAgent) Name() string {
	if i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *FileAgent) Home() string {
	return instance.Home(i)
}

func (i *FileAgent) Host() *geneos.Host {
	return i.InstanceHost
}

func (i *FileAgent) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *FileAgent) AuditEvent(event string, args ...any) {
	instance.AuditEvent(i, event, args...)
}

func (i *FileAgent) String() string {
	return instance.DisplayName(i)
}

func (i *FileAgent) Load() (err error) {
	return instance.Read(i)
}

func (i *FileAgent) Unload() (err error) {
	fileagents.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *FileAgent) Loaded() time.Time {
	return i.ConfigLoaded
}

func (i *FileAgent) SetLoaded(t time.Time) {
	i.ConfigLoaded = t
}

func (i *FileAgent) Config() *config.Config {
	return i.Conf
}

func (i *FileAgent) SetConfig(cf *config.Config) {
	i.Conf = cf
}

func (i *FileAgent) Add(tmpl string, port uint16, noCerts bool) (err error) {
	if port == 0 {
		port = instance.NextFreePort(i.Host(), &Component)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	config.Set(i.Config(), "port", port)

	// create certs, report success only
	if !noCerts {
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(io.Discard))
	}

	// default config XML etc.
	return nil
}

func (i *FileAgent) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	home = i.Home()
	logFile := instance.LogFilePath(i)
	checks = append(checks, filepath.Dir(logFile))
	args = []string{
		i.Name(),
		"-port", config.Get[string](i.Config(), "port"),
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

	missing := instance.CheckPaths(i, checks...)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (i *FileAgent) Reload() (err error) {
	return geneos.ErrNotSupported
}

// Rebuild is not supported for FileAgent instances. It always returns false
// and an ErrNotSupported error.
func (i *FileAgent) Rebuild(initial bool) (changed bool, err error) {
	return false, geneos.ErrNotSupported
}
