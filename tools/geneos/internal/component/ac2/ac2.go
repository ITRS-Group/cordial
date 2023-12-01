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
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var AC2 = geneos.Component{
	Name:             "ac2",
	Aliases:          []string{"active-console", "activeconsole", "desktop-activeconsole"},
	LegacyPrefix:     "",
	DownloadBase:     geneos.DownloadBases{Resources: "Active+Console", Nexus: "geneos-desktop-activeconsole"},
	DownloadInfix:    "desktop-activeconsole",
	PortRange:        "AC2PortRange",
	CleanList:        "AC2CleanList",
	PurgeList:        "AC2PurgeList",
	LegacyParameters: map[string]string{},
	Defaults: []string{
		`binary=ActiveConsole`,
		`home={{join .root "ac2" "ac2s" .name}}`,
		`install={{join .root "packages" "ac2"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=ActiveConsole.log`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}`,
		`config={{join .home "ActiveConsol.gci"}}`,
		`autostart=false`,
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
	GetPID: pidCheckFn,
}

const (
	ac2prefix = "collection-agent-"
	ac2suffix = "-exec.jar"
)

var ac2jarRE = regexp.MustCompile(`^` + ac2prefix + `(.+)` + ac2suffix)

var ac2Files = []string{
	// "ActiveConsole.gci",
	// "log4j2.properties",
	// "defaultws.dwx",
}

type AC2s instance.Instance

// ensure that AC2s satisfies geneos.Instance interface
var _ geneos.Instance = (*AC2s)(nil)

func init() {
	AC2.Register(factory)
}

var ac2s sync.Map

func factory(name string) geneos.Instance {
	_, local, h := instance.SplitName(name, geneos.LOCAL)
	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}
	n, ok := ac2s.Load(h.FullName(local))
	if ok {
		np, ok := n.(*AC2s)
		if ok {
			return np
		}
	}
	ac2 := &AC2s{}
	ac2.Conf = config.New()
	ac2.InstanceHost = h
	ac2.Component = &AC2
	if err := instance.SetDefaults(ac2, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", ac2)
	}
	// set the home dir based on where it might be, default to one above
	ac2.Config().Set("home", instance.Home(ac2))
	ac2s.Store(h.FullName(local), ac2)
	return ac2
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
	return instance.Home(n)
}

func (n *AC2s) Host() *geneos.Host {
	return n.InstanceHost
}

func (n *AC2s) String() string {
	return instance.DisplayName(n)
}

func (n *AC2s) Load() (err error) {
	return instance.LoadConfig(n)
}

func (n *AC2s) Unload() (err error) {
	ac2s.Delete(n.Name() + "@" + n.Host().String())
	n.ConfigLoaded = time.Time{}
	return
}

func (n *AC2s) Loaded() time.Time {
	return n.ConfigLoaded
}

func (n *AC2s) SetLoaded(t time.Time) {
	n.ConfigLoaded = t
}

func (n *AC2s) Config() *config.Config {
	return n.Conf
}

// Add created a new instance of AC2
func (n *AC2s) Add(tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(n.Host(), &AC2)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}

	baseDir := instance.BaseVersion(n)
	n.Config().Set("port", port)

	if err = instance.SaveConfig(n); err != nil {
		return
	}

	// create certs, report success only
	resp := instance.CreateCert(n)
	if resp.Err == nil {
		fmt.Println(resp.Line)
	}

	dir, err := os.Getwd()
	defer os.Chdir(dir)
	if err = os.Chdir(baseDir); err != nil {
		return
	}

	for _, source := range ac2Files {
		if _, err = geneos.ImportFile(n.Host(), n.Home(), source); err != nil && err != geneos.ErrExists {
			return
		}
	}
	err = nil

	return
}

func (n *AC2s) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}

// Command returns the command, args and environment for the instance
func (n *AC2s) Command() (args, env []string, home string) {
	home = instance.BaseVersion(n)

	args = []string{}

	env = []string{
		"_JAVA_OPTIONS=-Dawt.useSystemAAFontSettings=lcd",
	}

	return
}

func (n *AC2s) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(binary string, check interface{}, execfile string, args [][]byte) bool {
	c, ok := check.(*AC2s)
	if !ok {
		return false
	}
	if execfile == c.Config().GetString("program") {
		return true
	}
	return false
}
