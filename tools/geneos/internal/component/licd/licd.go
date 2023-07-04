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

package licd

import (
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var Licd = geneos.Component{
	Name:          "licd",
	LegacyPrefix:  "licd",
	RelatedTypes:  nil,
	Aliases:       []string{"licd", "licds"},
	RealComponent: true,
	DownloadBase:  geneos.DownloadBases{Resources: "Licence+Daemon", Nexus: "geneos-licd"},
	PortRange:     "LicdPortRange",
	CleanList:     "LicdCleanList",
	PurgeList:     "LicdPurgeList",
	LegacyParameters: map[string]string{
		"binsuffix": "binary",
		"licdhome":  "home",
		"licdbins":  "install",
		"licdbase":  "version",
		"licdexec":  "program",
		"licdlogd":  "logdir",
		"licdlogf":  "logfile",
		"licdport":  "port",
		"licdlibs":  "libpaths",
		"licdcert":  "certificate",
		"licdkey":   "privatekey",
		"licduser":  "user",
		"licdopts":  "options",
	},
	Defaults: []string{
		`binary=licd.linux_64`,
		`home={{join .root "licd" "licds" .name}}`,
		`install={{join .root "packages" "licd"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=licd.log`,
		`port=7041`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}`,
		`autostart=true`,
	},
	GlobalSettings: map[string]string{
		"LicdPortRange": "7041,7100-",
		"LicdCleanList": "*.old",
		"LicdPurgeList": "licd.log:licd.txt",
	},
	Directories: []string{
		"packages/licd",
		"licd/licds",
	},
}

type Licds instance.Instance

// ensure that Licds satisfies geneos.Instance interface
var _ geneos.Instance = (*Licds)(nil)

func init() {
	Licd.RegisterComponent(New)
}

var licds sync.Map

func New(name string) geneos.Instance {
	_, local, r := instance.SplitName(name, geneos.LOCAL)
	l, ok := licds.Load(r.FullName(local))
	if ok {
		lc, ok := l.(*Licds)
		if ok {
			return lc
		}
	}
	c := &Licds{}
	c.Conf = config.New()
	c.InstanceHost = r
	c.Component = &Licd
	if err := instance.SetDefaults(c, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", c)
	}
	// set the home dir based on where it might be, default to one above
	c.Config().Set("home", filepath.Join(instance.ParentDirectory(c), local))
	licds.Store(r.FullName(local), c)
	return c
}

// interface method set

// Return the Component for an Instance
func (l *Licds) Type() *geneos.Component {
	return l.Component
}

func (l *Licds) Name() string {
	if l.Config() == nil {
		return ""
	}
	return l.Config().GetString("name")
}

func (l *Licds) Home() string {
	return instance.HomeDir(l)
}

func (l *Licds) Host() *geneos.Host {
	return l.InstanceHost
}

func (l *Licds) String() string {
	return instance.DisplayName(l)
}

func (l *Licds) Load() (err error) {
	if l.ConfigLoaded {
		return
	}
	err = instance.LoadConfig(l)
	l.ConfigLoaded = err == nil
	return
}

func (l *Licds) Unload() (err error) {
	licds.Delete(l.Name() + "@" + l.Host().String())
	l.ConfigLoaded = false
	return
}

func (l *Licds) Loaded() bool {
	return l.ConfigLoaded
}

func (l *Licds) Config() *config.Config {
	return l.Conf
}

func (l *Licds) Add(tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(l.InstanceHost, &Licd)
	}
	l.Config().Set("port", port)

	if err = l.Config().Save(l.Type().String(),
		config.Host(l.Host()),
		config.SaveDir(instance.ParentDirectory(l)),
		config.SetAppName(l.Name()),
	); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// check tls config, create certs if found
	if _, err = instance.ReadSigningCert(); err == nil {
		if err = instance.CreateCert(l); err != nil {
			return
		}
	}

	// default config XML etc.
	return nil
}

func (l *Licds) Command() (args, env []string, home string) {
	args = []string{
		l.Name(),
		"-port", l.Config().GetString("port"),
		"-log", instance.LogFile(l),
	}

	args = append(args, instance.SetSecureArgs(l)...)
	home = l.Home()
	return
}

func (l *Licds) Reload(params []string) (err error) {
	return geneos.ErrNotSupported
}

func (l *Licds) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
