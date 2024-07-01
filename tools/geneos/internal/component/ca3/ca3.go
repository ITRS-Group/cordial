/*
Copyright © 2023 ITRS Group

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

package ca3

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

const Name = "ca3"

var CA3 = geneos.Component{
	Name:         Name,
	Aliases:      []string{"collection-agent", "ca3s", "collector"},
	LegacyPrefix: "",
	ParentType:   &netprobe.Netprobe,
	PackageTypes: []*geneos.Component{&netprobe.Netprobe},
	DownloadBase: geneos.DownloadBases{Default: "Netprobe", Nexus: "geneos-netprobe"},

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "7137-",
		config.Join(Name, "clean"): strings.Join([]string{
			"*.old",
		}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
			"*.log",
		}, ":"),
	},
	PortRange: config.Join(Name, "ports"),
	CleanList: config.Join(Name, "clean"),
	PurgeList: config.Join(Name, "purge"),
	ConfigAliases: map[string]string{
		config.Join(Name, "ports"): Name + "portrange",
		config.Join(Name, "clean"): Name + "cleanlist",
		config.Join(Name, "purge"): Name + "purgelist",
	},

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
	CA3.Register(factory)
}

var ca3s sync.Map

func factory(name string) geneos.Instance {
	_, local, h := instance.SplitName(name, geneos.LOCAL)
	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
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
	ca3.Config().Set("home", instance.Home(ca3))
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
	return instance.Home(n)
}

func (n *CA3s) Host() *geneos.Host {
	return n.InstanceHost
}

func (n *CA3s) String() string {
	return instance.DisplayName(n)
}

func (n *CA3s) Load() (err error) {
	return instance.LoadConfig(n)
}

func (n *CA3s) Unload() (err error) {
	ca3s.Delete(n.Name() + "@" + n.Host().String())
	n.ConfigLoaded = time.Time{}
	return
}

func (n *CA3s) Loaded() time.Time {
	return n.ConfigLoaded
}

func (n *CA3s) SetLoaded(t time.Time) {
	n.ConfigLoaded = t
}

func (n *CA3s) Config() *config.Config {
	return n.Conf
}

func (n *CA3s) Add(tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextFreePort(n.Host(), &CA3)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}

	baseDir := path.Join(instance.BaseVersion(n), "collection_agent")
	n.Config().Set("port", port)

	if err = instance.SaveConfig(n); err != nil {
		return
	}

	// create certs, report success only
	resp := instance.CreateCert(n, 0)
	if resp.Err == nil {
		fmt.Println(resp.Line)
	}

	// copy default configs
	dir, err := os.Getwd()
	defer os.Chdir(dir)
	if err = os.Chdir(baseDir); err != nil {
		return
	}

	for _, source := range ca3Files {
		if _, err = geneos.ImportFile(n.Host(), n.Home(), source); err != nil && err != geneos.ErrExists {
			return
		}
	}
	err = nil

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

	ca3dir, err := n.Host().ReadDir(baseDir)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	latest := ""
	for _, n := range ca3dir {
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

func (n *CA3s) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(arg any, cmdline ...[]byte) bool {
	var jarOK, configOK bool

	c, ok := arg.(*CA3s)
	if !ok {
		return false
	}

	if path.Base(string(cmdline[0])) != "java" {
		return false
	}

	for _, arg := range cmdline[1:] {
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
