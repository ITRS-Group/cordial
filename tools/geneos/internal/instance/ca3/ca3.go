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

package ca3

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
)

var CA3 = geneos.Component{
	Name:             "ca3",
	RelatedTypes:     []*geneos.Component{&netprobe.Netprobe},
	ComponentMatches: []string{"ca3", "collection-agent", "ca3s", "collector"},
	RealComponent:    true,
	DownloadBase:     geneos.DownloadBases{Resources: "Netprobe", Nexus: "geneos-netprobe"},
	PortRange:        "CA3PortRange",
	CleanList:        "CA3CleanList",
	PurgeList:        "CA3PurgeList",
	Aliases:          map[string]string{},
	Defaults: []string{
		`home={{join .root "ca3" "ca3s" .name}}`,
		`install={{join .root "packages" "netprobe"}}`,
		`plugins={{join .install "collection_agent" "plugins"}}`,
		`version=active_prod`,
		`program={{"/usr/bin/java"}}`,
		`logfile=collection-agent.log`,
		`config={{join .home "collection-agent.yml"}}`,
		`minheap=512M`,
		`maxheap=512M`,
	},
	GlobalSettings: map[string]string{
		"CA3PortRange": "7137-",
		"CA3CleanList": "*.old",
		"CA3PurgeList": "*.log",
	},
	Directories: []string{
		"packages/ca3",
		"ca3/ca3s",
	},
	GetPID: ca3getPID,
}

const (
	ca3prefix = "collection-agent-"
	ca3suffix = "-exec.jar"
)

var ca3jarRE = regexp.MustCompile(`^` + ca3prefix + `(.+)` + ca3suffix)

var ca3Files = []string{
	"collection-agent.yml",
	"logback.xml",
}

type CA3s instance.Instance

// ensure that CA3s satisfies geneos.Instance interface
var _ geneos.Instance = (*CA3s)(nil)

func init() {
	CA3.RegisterComponent(New)
}

var ca3s sync.Map

func New(name string) geneos.Instance {
	_, local, r := instance.SplitName(name, geneos.LOCAL)
	n, ok := ca3s.Load(r.FullName(local))
	if ok {
		np, ok := n.(*CA3s)
		if ok {
			return np
		}
	}
	c := &CA3s{}
	c.Conf = config.New()
	c.InstanceHost = r
	c.Component = &CA3
	if err := instance.SetDefaults(c, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", c)
	}
	ca3s.Store(r.FullName(local), c)
	return c
}

// interface method set

// Return the Component for an Instance
func (n *CA3s) Type() *geneos.Component {
	return n.Component
}

func (n *CA3s) Name() string {
	if n.Config() == nil {
		return ""
	}
	return n.Config().GetString("name")
}

func (n *CA3s) Home() string {
	if n.Config() == nil {
		return ""
	}
	return n.Config().GetString("home")
}

func (n *CA3s) Prefix() string {
	return "ca3"
}

func (n *CA3s) Host() *geneos.Host {
	return n.InstanceHost
}

func (n *CA3s) String() string {
	return instance.DisplayName(n)
}

func (n *CA3s) Load() (err error) {
	if n.ConfigLoaded {
		return
	}
	err = instance.LoadConfig(n)
	n.ConfigLoaded = err == nil
	return
}

func (n *CA3s) Unload() (err error) {
	ca3s.Delete(n.Name() + "@" + n.Host().String())
	n.ConfigLoaded = false
	return
}

func (n *CA3s) Loaded() bool {
	return n.ConfigLoaded
}

func (n *CA3s) Config() *config.Config {
	return n.Conf
}

func (n *CA3s) Add(tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(n.Host(), &CA3)
	}

	baseDir := filepath.Join(n.Config().GetString("install"), n.Config().GetString("version"), "collection_agent")
	n.Config().Set("port", port)

	// instance.SetEnvs(n, []string{
	// 	fmt.Sprintf("CA_PLUGIN_DIR=%s", filepath.Join(baseDir, "plugins")),
	// 	fmt.Sprintf("HEALTH_CHECK_PORT=%d", 9136),
	// 	fmt.Sprintf("TCP_REPORTER_PORT=%d", 7137),
	// })

	if err = instance.WriteConfig(n); err != nil {
		return
	}

	// check tls config, create certs if found
	if _, err = instance.ReadSigningCert(); err == nil {
		if err = instance.CreateCert(n); err != nil {
			return
		}
	}

	dir, err := os.Getwd()
	defer os.Chdir(dir)
	if err = os.Chdir(baseDir); err != nil {
		return
	}

	for _, source := range ca3Files {
		if _, err = instance.ImportFile(n.Host(), n.Home(), n.Config().GetString("user"), source); err != nil {
			return
		}
	}

	// default config XML etc.
	return
}

func (n *CA3s) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}

// XXX the is for initial testing - needs cleaning up

func (n *CA3s) Command() (args, env []string) {
	// locate jar file
	baseDir := filepath.Join(n.Config().GetString("install"), n.Config().GetString("version"), "collection_agent")

	d, err := os.ReadDir(baseDir)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	latest := ""
	for _, n := range d {
		parts := ca3jarRE.FindStringSubmatch(n.Name())
		log.Debug().Msgf("found %d parts: %v", len(parts), parts)
		if len(parts) > 1 {
			if geneos.CompareVersion(parts[1], latest) > 0 {
				latest = parts[1]
			}
		}
	}
	log.Debug().Msgf("latest version %s", latest)
	var jar string
	if latest != "" {
		jar = filepath.Join(baseDir, ca3prefix+latest+ca3suffix)
	}

	args = []string{
		"-Xms" + n.Config().GetString("minheap", config.Default("512M")),
		"-Xmx" + n.Config().GetString("maxheap", config.Default("512M")),
		"-Dlogback.configurationFile=" + filepath.Join(baseDir, "logback.xml"),
		"-jar", jar, n.Config().GetString("config"),
	}

	env = []string{
		fmt.Sprintf("CA_PLUGIN_DIR=%s", n.Config().GetString("plugins", config.Default(filepath.Join(baseDir, "plugins")))),
		fmt.Sprintf("HEALTH_CHECK_PORT=%d", n.Config().GetInt("health-check-port", config.Default(9136))),
		fmt.Sprintf("TCP_REPORTER_PORT=%d", n.Config().GetInt("tcp-reporter-port", config.Default(7137))),
	}

	return
}

func (n *CA3s) Reload(params []string) (err error) {
	return geneos.ErrNotSupported
}
