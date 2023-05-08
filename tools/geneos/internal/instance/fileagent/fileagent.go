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
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var FileAgent = geneos.Component{
	Name:             "fileagent",
	RelatedTypes:     nil,
	ComponentMatches: []string{"fileagent", "fileagents", "file-agent"},
	RealComponent:    true,
	DownloadBase:     geneos.DownloadBases{Resources: "Fix+Analyser+File+Agent", Nexus: "geneos-file-agent"},
	PortRange:        "FAPortRange",
	CleanList:        "FACleanList",
	PurgeList:        "FAPurgeList",
	Aliases: map[string]string{
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
	FileAgent.RegisterComponent(New)
}

var fileagents sync.Map

func New(name string) geneos.Instance {
	_, local, r := instance.SplitName(name, geneos.LOCAL)
	f, ok := fileagents.Load(r.FullName(local))
	if ok {
		fa, ok := f.(*FileAgents)
		if ok {
			return fa
		}
	}
	c := &FileAgents{}
	c.Conf = config.New()
	c.InstanceHost = r
	c.Component = &FileAgent
	if err := instance.SetDefaults(c, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()")
	}
	fileagents.Store(r.FullName(local), c)
	return c
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
	if n.Config() == nil {
		return ""
	}
	return n.Config().GetString("home")
}

func (n *FileAgents) Prefix() string {
	return "fa"
}

func (n *FileAgents) Host() *geneos.Host {
	return n.InstanceHost
}

func (n *FileAgents) String() string {
	return instance.DisplayName(n)
}

func (n *FileAgents) Load() (err error) {
	if n.ConfigLoaded {
		return
	}
	err = instance.LoadConfig(n)
	n.ConfigLoaded = err == nil
	return
}

func (n *FileAgents) Unload() (err error) {
	fileagents.Delete(n.Name() + "@" + n.Host().String())
	n.ConfigLoaded = false
	return
}

func (n *FileAgents) Loaded() bool {
	return n.ConfigLoaded
}

func (n *FileAgents) Config() *config.Config {
	return n.Conf
}

func (n *FileAgents) SetConf(v *config.Config) {
	n.Conf = v
}

func (n *FileAgents) Add(username string, tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(n.Host(), &FileAgent)
	}
	n.Config().Set("port", port)
	n.Config().Set("user", username)

	if err = instance.WriteConfig(n); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// check tls config, create certs if found
	if _, err = instance.ReadSigningCert(); err == nil {
		if err = instance.CreateCert(n); err != nil {
			return
		}
	}

	// default config XML etc.
	return nil
}

func (c *FileAgents) Command() (args, env []string) {
	logFile := instance.LogFile(c)
	args = []string{
		c.Name(),
		"-port", c.Config().GetString("port"),
	}
	env = append(env, "LOG_FILENAME="+logFile)

	return
}

func (c *FileAgents) Reload(params []string) (err error) {
	return geneos.ErrNotSupported
}

func (c *FileAgents) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
