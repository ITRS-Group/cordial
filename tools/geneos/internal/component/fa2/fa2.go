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

package fa2

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var FA2 = geneos.Component{
	Name:          "fa2",
	Aliases:       []string{"fixanalyser", "fixanalyzer", "fixanalyser2-netprobe"},
	LegacyPrefix:  "fa2",
	ParentType:    &netprobe.Netprobe,
	DownloadBase:  geneos.DownloadBases{Resources: "Fix+Analyser+2+Netprobe", Nexus: "geneos-fixanalyser2-netprobe"},
	DownloadInfix: "fixanalyser2-netprobe",
	PortRange:     "FA2PortRange",
	CleanList:     "FA2CleanList",
	PurgeList:     "FA2PurgeList",
	LegacyParameters: map[string]string{
		"binsuffix": "binary",
		"fa2home":   "home",
		"fa2bins":   "install",
		"fa2base":   "version",
		"fa2exec":   "program",
		"fa2logd":   "logdir",
		"fa2logf":   "logfile",
		"fa2port":   "port",
		"fa2libs":   "libpaths",
		"fa2cert":   "certificate",
		"fa2key":    "privatekey",
		"fa2user":   "user",
		"fa2opts":   "options",
	},
	Defaults: []string{
		`binary=fix-analyser2-netprobe.linux_64`,
		`home={{join .root "netprobe" "fa2s" .name}}`,
		`install={{join .root "packages" "fa2"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=fa2.log`,
		`port=7030`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:{{join "${config:install}" "${config:version}"}}`,
		`autostart=true`,
	},
	GlobalSettings: map[string]string{
		"FA2PortRange": "7030,7100-",
		"FA2CleanList": "*.old",
		"FA2PurgeList": "fa2.log:fa2.txt:*.snooze:*.user_assignment",
	},
	Directories: []string{
		"packages/fa2",
		"netprobe/fa2s",
	},
}

type FA2s instance.Instance

// ensure that FA2s satisfies geneos.Instance interface
var _ geneos.Instance = (*FA2s)(nil)

func init() {
	FA2.Register(factory)
}

var fa2s sync.Map

func factory(name string) geneos.Instance {
	_, local, h := instance.SplitName(name, geneos.LOCAL)
	if local == "" || h == geneos.LOCAL && geneos.Root() == "" {
		return nil
	}
	f, ok := fa2s.Load(h.FullName(local))
	if ok {
		fa, ok := f.(*FA2s)
		if ok {
			return fa
		}
	}
	fa2 := &FA2s{}
	fa2.Conf = config.New()
	fa2.InstanceHost = h
	fa2.Component = &FA2
	if err := instance.SetDefaults(fa2, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", fa2)
	}
	// set the home dir based on where it might be, default to one above
	fa2.Config().Set("home", instance.Home(fa2))
	fa2s.Store(h.FullName(local), fa2)
	return fa2
}

// interface method set

// Return the Component for an Instance
func (n *FA2s) Type() *geneos.Component {
	return n.Component
}

func (n *FA2s) Name() string {
	if n.Config() == nil {
		return ""
	}
	return n.Config().GetString("name")
}

func (n *FA2s) Home() string {
	return instance.Home(n)
}

func (n *FA2s) Host() *geneos.Host {
	return n.InstanceHost
}

func (n *FA2s) String() string {
	return instance.DisplayName(n)
}

func (n *FA2s) Load() (err error) {
	return instance.LoadConfig(n)
}

func (n *FA2s) Unload() (err error) {
	fa2s.Delete(n.Name() + "@" + n.Host().String())
	n.ConfigLoaded = time.Time{}
	return
}

func (n *FA2s) Loaded() time.Time {
	return n.ConfigLoaded
}

func (n *FA2s) SetLoaded(t time.Time) {
	n.ConfigLoaded = t
}

func (n *FA2s) Config() *config.Config {
	return n.Conf
}

func (n *FA2s) Add(tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(n.InstanceHost, &FA2)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	n.Config().Set("port", port)

	if err = instance.SaveConfig(n); err != nil {
		return
	}

	// create certs, report success only
	resp := instance.CreateCert(n)
	if resp.Err == nil {
		fmt.Println(resp.Line)
	}

	// default config XML etc.
	return nil
}

func (n *FA2s) Command() (args, env []string, home string) {
	logFile := instance.LogFilePath(n)
	args = []string{
		n.Name(),
		"-port", n.Config().GetString("port"),
	}
	args = append(args, instance.SetSecureArgs(n)...)
	env = append(env, "LOG_FILENAME="+logFile)
	home = n.Home()

	return
}

func (n *FA2s) Reload() (err error) {
	return geneos.ErrNotSupported
}

func (n *FA2s) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
