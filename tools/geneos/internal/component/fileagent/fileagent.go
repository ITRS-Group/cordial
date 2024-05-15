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

package fileagent

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var FileAgent = geneos.Component{
	Name:          "fileagent",
	Aliases:       []string{"fileagents", "file-agent"},
	LegacyPrefix:  "fa",
	DownloadBase:  geneos.DownloadBases{Resources: "Fix+Analyser+File+Agent", Nexus: "geneos-file-agent"},
	DownloadInfix: "file-agent",
	PortRange:     "FAPortRange",
	CleanList:     "FACleanList",
	PurgeList:     "FAPurgeList",
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
		`binary=agent.linux_64`,
		`home={{join .root "fileagent" "fileagents" .name}}`,
		`install={{join .root "packages" "fileagent"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=fileagent.log`,
		`port=7030`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:{{join "${config:install}" "${config:version}"}}`,
		`autostart=true`,
	},
	GlobalSettings: map[string]string{
		"FAPortRange": "7030,7100-",
		"FACleanList": "*.old",
		"FAPurgeList": "fileagent.log:fileagent.txt",
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

var fileagents sync.Map

func factory(name string) geneos.Instance {
	_, local, h := instance.SplitName(name, geneos.LOCAL)
	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}
	f, ok := fileagents.Load(h.FullName(local))
	if ok {
		fa, ok := f.(*FileAgents)
		if ok {
			return fa
		}
	}
	fileagent := &FileAgents{}
	fileagent.Conf = config.New()
	fileagent.InstanceHost = h
	fileagent.Component = &FileAgent
	if err := instance.SetDefaults(fileagent, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", fileagent)
	}
	// set the home dir based on where it might be, default to one above
	fileagent.Config().Set("home", instance.Home(fileagent))
	fileagents.Store(h.FullName(local), fileagent)
	return fileagent
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
	fileagents.Delete(n.Name() + "@" + n.Host().String())
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

func (n *FileAgents) Add(tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(n.Host(), &FileAgent)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	n.Config().Set("port", port)

	if err = instance.SaveConfig(n); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// create certs, report success only
	resp := instance.CreateCert(n, 0)
	if resp.Err == nil {
		fmt.Println(resp.Line)
	}

	// default config XML etc.
	return nil
}

func (c *FileAgents) Command() (args, env []string, home string) {
	logFile := instance.LogFilePath(c)
	args = []string{
		c.Name(),
		"-port", c.Config().GetString("port"),
	}
	_, version, err := instance.Version(c)
	if err == nil {
		switch {
		case geneos.CompareVersion(version, "6.6.0") >= 0:
			args = append(args, instance.SetSecureArgs(c)...)
		}
	}
	env = append(env, "LOG_FILENAME="+logFile)
	home = c.Home()
	return
}

func (c *FileAgents) Reload() (err error) {
	return geneos.ErrNotSupported
}

func (c *FileAgents) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
