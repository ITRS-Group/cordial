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
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var CA3 = geneos.Component{
	Name:             "ca3",
	LegacyPrefix:     "",
	RelatedTypes:     []*geneos.Component{&netprobe.Netprobe},
	Aliases:          []string{"ca3", "collection-agent", "ca3s", "collector"},
	ParentType:       &netprobe.Netprobe,
	RealComponent:    true,
	DownloadBase:     geneos.DownloadBases{Resources: "Netprobe", Nexus: "geneos-netprobe"},
	PortRange:        "CA3PortRange",
	CleanList:        "CA3CleanList",
	PurgeList:        "CA3PurgeList",
	LegacyParameters: map[string]string{},
	Defaults: []string{
		`binary=java`, // needed for 'ps' matching
		`home={{join .root "netprobe" "ca3s" .name}}`,
		`install={{join .root "packages" "netprobe"}}`,
		`version=active_prod`,
		`plugins={{join .install .version "collection_agent"}}`,
		`program={{"/usr/bin/java"}}`,
		`logdir={{join .home "collection_agent"}}`,
		`logfile=collection-agent.log`,
		`config={{join .home "collection-agent.yml"}}`,
		`minheap=512M`,
		`maxheap=512M`,
		`autostart=true`,
	},
	GlobalSettings: map[string]string{
		"CA3PortRange": "7137-",
		"CA3CleanList": "*.old",
		"CA3PurgeList": "*.log",
	},
	Directories: []string{
		"packages/ca3",
		"netprobe/netprobes_shared",
		"netprobe/ca3s",
	},
	GetPID: pidCheckFn,
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
	CA3.RegisterComponent(factory)
}

var ca3s sync.Map

func factory(name string) geneos.Instance {
	_, local, h := instance.SplitName(name, geneos.LOCAL)
	if h == geneos.LOCAL && geneos.Root() == "" {
		return nil
	}
	n, ok := ca3s.Load(h.FullName(local))
	if ok {
		np, ok := n.(*CA3s)
		if ok {
			return np
		}
	}
	ca3 := &CA3s{}
	ca3.Conf = config.New()
	ca3.InstanceHost = h
	ca3.Component = &CA3
	if err := instance.SetDefaults(ca3, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", ca3)
	}
	// set the home dir based on where it might be, default to one above
	ca3.Config().Set("home", instance.HomeDir(ca3))
	ca3s.Store(h.FullName(local), ca3)
	return ca3
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
	return instance.HomeDir(n)
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

	baseDir := path.Join(instance.BaseVersion(n), "collection_agent")
	n.Config().Set("port", port)

	if err = instance.SaveConfig(n); err != nil {
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
		if _, err = instance.ImportFile(n.Host(), n.Home(), source); err != nil {
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

func (n *CA3s) Command() (args, env []string, home string) {
	// locate jar file
	baseDir := path.Join(instance.BaseVersion(n), "collection_agent")

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
		jar = path.Join(baseDir, ca3prefix+latest+ca3suffix)
	}

	args = []string{
		"-Xms" + n.Config().GetString("minheap", config.Default("512M")),
		"-Xmx" + n.Config().GetString("maxheap", config.Default("512M")),
		"-Dlogback.configurationFile=" + path.Join(baseDir, "logback.xml"),
		"-jar", jar, n.Config().GetString("config"),
	}

	env = []string{
		fmt.Sprintf("CA_PLUGIN_DIR=%s", n.Config().GetString("plugins", config.Default(path.Join(baseDir, "plugins")))),
		fmt.Sprintf("HEALTH_CHECK_PORT=%d", n.Config().GetInt("health-check-port", config.Default(9136))),
		fmt.Sprintf("TCP_REPORTER_PORT=%d", n.Config().GetInt("tcp-reporter-port", config.Default(7137))),
	}

	home = n.Home()

	return
}

func (n *CA3s) Reload(params []string) (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(binary string, check interface{}, execfile string, args [][]byte) bool {
	var jarOK, configOK bool

	c, ok := check.(*CA3s)
	if !ok {
		return false
	}
	if execfile != "java" {
		return false
	}
	for _, arg := range args[1:] {
		if strings.Contains(string(arg), "collection-agent") {
			jarOK = true
		}
		if strings.Contains(string(arg), c.Config().GetString("config")) {
			configOK = true
		}
		if jarOK && configOK {
			return true
		}
	}
	return false
}
