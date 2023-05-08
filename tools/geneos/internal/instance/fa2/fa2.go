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
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var FA2 = geneos.Component{
	Name:             "fa2",
	RelatedTypes:     nil,
	ComponentMatches: []string{"fa2", "fixanalyser", "fixanalyzer", "fixanalyser2-netprobe"},
	RealComponent:    true,
	DownloadBase:     geneos.DownloadBases{Resources: "Fix+Analyser+2+Netprobe", Nexus: "geneos-fixanalyser2-netprobe"},
	PortRange:        "FA2PortRange",
	CleanList:        "FA2CleanList",
	PurgeList:        "FA2PurgeList",
	Aliases: map[string]string{
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
		`home={{join .root "fa2" "fa2s" .name}}`,
		`install={{join .root "packages" "fa2"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=fa2.log`,
		`port=7036`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:{{join "${config:install}" "${config:version}"}}`,
	},
	GlobalSettings: map[string]string{
		"FA2PortRange": "7030,7100-",
		"FA2CleanList": "*.old",
		"FA2PurgeList": "fa2.log:fa2.txt:*.snooze:*.user_assignment",
	},
	Directories: []string{
		"packages/fa2",
		"fa2/fa2s",
	},
}

type FA2s instance.Instance

// ensure that FA2s satisfies geneos.Instance interface
var _ geneos.Instance = (*FA2s)(nil)

func init() {
	FA2.RegisterComponent(New)
}

var fa2s sync.Map

func New(name string) geneos.Instance {
	_, local, r := instance.SplitName(name, geneos.LOCAL)
	f, ok := fa2s.Load(r.FullName(local))
	if ok {
		fa, ok := f.(*FA2s)
		if ok {
			return fa
		}
	}
	c := &FA2s{}
	c.Conf = config.New()
	c.InstanceHost = r
	c.Component = &FA2
	if err := instance.SetDefaults(c, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()")
	}
	fa2s.Store(r.FullName(local), c)
	return c
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
	if n.Config() == nil {
		return ""
	}
	return n.Config().GetString("home")
}

func (n *FA2s) Prefix() string {
	return "fa2"
}

func (n *FA2s) Host() *geneos.Host {
	return n.InstanceHost
}

func (n *FA2s) String() string {
	return instance.DisplayName(n)
}

func (n *FA2s) Load() (err error) {
	if n.ConfigLoaded {
		return
	}
	err = instance.LoadConfig(n)
	n.ConfigLoaded = err == nil
	return
}

func (n *FA2s) Unload() (err error) {
	fa2s.Delete(n.Name() + "@" + n.Host().String())
	n.ConfigLoaded = false
	return
}

func (n *FA2s) Loaded() bool {
	return n.ConfigLoaded
}

func (n *FA2s) Config() *config.Config {
	return n.Conf
}

func (n *FA2s) SetConf(v *config.Config) {
	n.Conf = v
}

func (n *FA2s) Add(username string, tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(n.InstanceHost, &FA2)
	}
	n.Config().Set("port", port)
	n.Config().Set("user", username)

	if err = instance.WriteConfig(n); err != nil {
		return
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

func (n *FA2s) Command() (args, env []string) {
	logFile := instance.LogFile(n)
	args = []string{
		n.Name(),
		"-port", n.Config().GetString("port"),
	}
	args = append(args, instance.SetSecureArgs(n)...)

	env = append(env, "LOG_FILENAME="+logFile)

	return
}

func (n *FA2s) Reload(params []string) (err error) {
	return geneos.ErrNotSupported
}

func (n *FA2s) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
