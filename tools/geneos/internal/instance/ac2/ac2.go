/*
Copyright © 2023 ITRS Group

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

// Package ac2 supports installation and control of the Active Console
package ac2

import (
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var AC2 = geneos.Component{
	Name:             "ac2",
	LegacyPrefix:     "",
	RelatedTypes:     []*geneos.Component{},
	ComponentMatches: []string{"ac2", "active-console", "activeconsole"},
	RealComponent:    true,
	//https://resources.itrsgroup.com/download/latest/Active+Console?title=geneos-desktop-activeconsole-6.3.0-linux-x64.tar.gz
	DownloadBase: geneos.DownloadBases{Resources: "Active+Console", Nexus: "geneos-desktop-activeconsole"},
	PortRange:    "AC2PortRange",
	CleanList:    "AC2CleanList",
	PurgeList:    "AC2PurgeList",
	Aliases:      map[string]string{},
	Defaults: []string{
		`home={{join .root "ac2" "ac2s" .name}}`,
		`install={{join .root "packages" "ac2"}}`,
		`version=active_prod`,
		`program={{"ActiveConsole"}}`,
		`logfile=ActiveConsole.log`,
		`config={{join .home "collection-agent.yml"}}`,
	},
	GlobalSettings: map[string]string{
		"AC2PortRange": "7040-",
		"AC2CleanList": "*.old",
		"AC2PurgeList": "*.log",
	},
	Directories: []string{
		"packages/ac2",
		"ac2/ac2s",
	},
	GetPID: getPID,
}

const (
	ac2prefix = "collection-agent-"
	ac2suffix = "-exec.jar"
)

var ac2jarRE = regexp.MustCompile(`^` + ac2prefix + `(.+)` + ac2suffix)

var ac2Files = []string{
	"collection-agent.yml",
	"logback.xml",
}

type AC2s instance.Instance

// ensure that AC2s satisfies geneos.Instance interface
var _ geneos.Instance = (*AC2s)(nil)

func init() {
	AC2.RegisterComponent(New)
}

var ac2s sync.Map

func New(name string) geneos.Instance {
	_, local, r := instance.SplitName(name, geneos.LOCAL)
	n, ok := ac2s.Load(r.FullName(local))
	if ok {
		np, ok := n.(*AC2s)
		if ok {
			return np
		}
	}
	c := &AC2s{}
	c.Conf = config.New()
	c.InstanceHost = r
	c.Component = &AC2
	if err := instance.SetDefaults(c, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", c)
	}
	ac2s.Store(r.FullName(local), c)
	return c
}

// interface method set

// Return the Component for an Instance
func (n *AC2s) Type() *geneos.Component {
	return n.Component
}

func (n *AC2s) Name() string {
	if n.Config() == nil {
		return ""
	}
	return n.Config().GetString("name")
}

func (n *AC2s) Home() string {
	if n.Config() == nil {
		return ""
	}
	return n.Config().GetString("home")
}

func (n *AC2s) Host() *geneos.Host {
	return n.InstanceHost
}

func (n *AC2s) String() string {
	return instance.DisplayName(n)
}

func (n *AC2s) Load() (err error) {
	if n.ConfigLoaded {
		return
	}
	err = instance.LoadConfig(n)
	n.ConfigLoaded = err == nil
	return
}

func (n *AC2s) Unload() (err error) {
	ac2s.Delete(n.Name() + "@" + n.Host().String())
	n.ConfigLoaded = false
	return
}

func (n *AC2s) Loaded() bool {
	return n.ConfigLoaded
}

func (n *AC2s) Config() *config.Config {
	return n.Conf
}

func (n *AC2s) Add(tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(n.Host(), &AC2)
	}

	baseDir := filepath.Join(n.Config().GetString("install"), n.Config().GetString("version"), "collection_agent")
	n.Config().Set("port", port)

	if err = n.Config().Save(n.Type().String(),
		config.Host(n.Host()),
		config.SaveDir(n.Type().InstancesDir(n.Host())),
		config.SetAppName(n.Name()),
	); err != nil {
		return
	}

	// check tls config, create certs if found
	if _, err = instance.ReadSigningCert(filepath.Join(geneos.Root(), "tls")); err == nil {
		if err = instance.CreateCert(n); err != nil {
			return
		}
	}

	dir, err := os.Getwd()
	defer os.Chdir(dir)
	if err = os.Chdir(baseDir); err != nil {
		return
	}

	for _, source := range ac2Files {
		if _, err = instance.ImportFile(n.Host(), n.Home(), source); err != nil {
			return
		}
	}

	// default config XML etc.
	return
}

func (n *AC2s) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}

// XXX the is for initial testing - needs cleaning up

func (n *AC2s) Command() (args, env []string) {
	// locate jar file
	baseDir := filepath.Join(n.Config().GetString("install"), n.Config().GetString("version"), "collection_agent")

	d, err := os.ReadDir(baseDir)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	latest := ""
	for _, n := range d {
		parts := ac2jarRE.FindStringSubmatch(n.Name())
		log.Debug().Msgf("found %d parts: %v", len(parts), parts)
		if len(parts) > 1 {
			if geneos.CompareVersion(parts[1], latest) > 0 {
				latest = parts[1]
			}
		}
	}

	args = []string{}

	env = []string{}

	return
}

func (n *AC2s) Reload(params []string) (err error) {
	return geneos.ErrNotSupported
}
