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

package gateway

import (
	_ "embed"
	"fmt"
	"path"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var Gateway = geneos.Component{
	Initialise:       Init,
	Name:             "gateway",
	LegacyPrefix:     "gate",
	RelatedTypes:     nil,
	ComponentMatches: []string{"gateway", "gateways"},
	RealComponent:    true,
	UsesKeyfiles:     true,
	Templates: []geneos.Templates{
		{Filename: templateName, Content: template},
		{Filename: instanceTemplateName, Content: instanceTemplate},
	},
	DownloadBase: geneos.DownloadBases{Resources: "Gateway+2", Nexus: "geneos-gateway"},
	PortRange:    "GatewayPortRange",
	CleanList:    "GatewayCleanList",
	PurgeList:    "GatewayPurgeList",
	Aliases: map[string]string{
		"binsuffix": "binary",
		"gatehome":  "home",
		"gatebins":  "install",
		"gatebase":  "version",
		"gateexec":  "program",
		"gatelogd":  "logdir",
		"gatelogf":  "logfile",
		"gateport":  "port",
		"gatelibs":  "libpaths",
		"gatecert":  "certificate",
		"gatekey":   "privatekey",
		"gateaes":   "keyfile",
		"aesfile":   "keyfile",
		"gatename":  "gatewayname",
		"gatelich":  "licdhost",
		"gatelicp":  "licdport",
		"gatelics":  "licdsecure",
		"gateuser":  "user",
		"gateopts":  "options",
	},
	Defaults: []string{
		`binary=gateway2.linux_64`,
		`home={{join .root "gateway" "gateways" .name}}`,
		`install={{join .root "packages" "gateway"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=gateway.log`,
		`port=7039`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:/usr/lib64`,
		`gatewayname={{.name}}`,
		`setup={{join "${config:home}" "gateway.setup.xml"}}`,
	},
	GlobalSettings: map[string]string{
		"GatewayPortRange": "7039,7100-",
		"GatewayCleanList": "*.old:*.history",
		"GatewayPurgeList": "gateway.log:gateway.txt:gateway.snooze:gateway.user_assignment:licences.cache:cache/:database/",
	},
	Directories: []string{
		"packages/gateway",
		"gateway/gateways",
		"gateway/gateway_shared",
		"gateway/gateway_config",
		"gateway/templates",
	},
}

type Gateways instance.Instance

// ensure that Gateways satisfies geneos.Instance interface
var _ geneos.Instance = (*Gateways)(nil)

//go:embed templates/gateway.setup.xml.gotmpl
var template []byte

//go:embed templates/gateway-instance.setup.xml.gotmpl
var instanceTemplate []byte

const templateName = "gateway.setup.xml.gotmpl"
const instanceTemplateName = "gateway-instance.setup.xml.gotmpl"

func init() {
	Gateway.RegisterComponent(New)
}

func Init(r *geneos.Host, ct *geneos.Component) {
	// copy default template to directory
	if err := r.WriteFile(r.Filepath("gateway", "templates", templateName), template, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
	if err := r.WriteFile(r.Filepath("gateway", "templates", instanceTemplateName), instanceTemplate, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

var gateways sync.Map

func New(name string) geneos.Instance {
	_, local, h := instance.SplitName(name, geneos.LOCAL)
	if i, ok := gateways.Load(h.FullName(local)); ok {
		if g, ok := i.(*Gateways); ok {
			return g
		}
	}
	g := &Gateways{}
	g.Conf = config.New()
	g.Component = &Gateway
	g.InstanceHost = h
	if err := instance.SetDefaults(g, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", g)
	}
	// set the home dir based on where it might be, default to one above
	g.Config().Set("home", filepath.Join(instance.ParentDirectory(g), local))
	gateways.Store(h.FullName(local), g)
	return g
}

// interface method set

// Return the Component for an Instance
func (g *Gateways) Type() *geneos.Component {
	return g.Component
}

func (g *Gateways) Name() string {
	if g.Config() == nil {
		return ""
	}
	return g.Config().GetString("name")
}

func (g *Gateways) Home() string {
	if g.Config() == nil {
		return ""
	}
	return g.Config().GetString("home")
}

func (g *Gateways) Host() *geneos.Host {
	return g.InstanceHost
}

func (g *Gateways) String() string {
	return instance.DisplayName(g)
}

func (g *Gateways) Load() (err error) {
	if g.ConfigLoaded {
		return
	}
	err = instance.LoadConfig(g)
	g.ConfigLoaded = err == nil
	return
}

func (g *Gateways) Unload() (err error) {
	gateways.Delete(g.Name() + "@" + g.Host().String())
	g.ConfigLoaded = false
	return
}

func (g *Gateways) Loaded() bool {
	return g.ConfigLoaded
}

func (g *Gateways) Config() *config.Config {
	return g.Conf
}

func (g *Gateways) Add(template string, port uint16) (err error) {
	cf := g.Config()

	if port == 0 {
		port = instance.NextPort(g.InstanceHost, &Gateway)
	}
	cf.Set("port", port)
	cf.Set(cf.Join("config", "rebuild"), "initial")

	cf.SetDefault(cf.Join("config", "template"), templateName)
	if template != "" {
		filename, _ := instance.ImportCommons(g.Host(), g.Type(), "templates", []string{template})
		cf.Set(cf.Join("config", "template"), filename)
	}

	cf.Set("includes", make(map[int]string))

	// try to save config early
	log.Debug().Msgf("dir: %s", instance.ParentDirectory(g))
	if err = g.Config().Save(g.Type().String(),
		config.Host(g.Host()),
		config.SaveDir(instance.ParentDirectory(g)),
		config.SetAppName(g.Name()),
	); err != nil {
		log.Fatal().Err(err).Msg("")
		return
	}

	// check tls config, create certs if found
	if _, err = instance.ReadSigningCert(); err == nil {
		if err = instance.CreateCert(g); err != nil {
			return
		}
	}

	// always create a keyfile ?
	if err = createAESKeyFile(g); err != nil {
		return
	}

	return nil // g.Rebuild(true)
}

func (g *Gateways) Rebuild(initial bool) (err error) {
	cf := g.Config()

	// always rebuild an instance template
	err = instance.CreateConfigFromTemplate(g, filepath.Join(g.Home(), "instance.setup.xml"), instanceTemplateName, instanceTemplate)
	if err != nil {
		return
	}
	fmt.Printf("%s instance template %q rebuilt\n", g, "instance.setup.xml")

	configrebuild := cf.GetString("config::rebuild")

	if configrebuild == "never" {
		return
	}

	if !(configrebuild == "always" || (initial && configrebuild == "initial")) {
		return
	}

	// recheck check certs/keys
	var changed bool
	secure := instance.Filename(g, "certificate") != "" && instance.Filename(g, "privatekey") != ""

	// if we have certs then connect to Licd securely
	if secure && cf.GetString("licdsecure") != "true" {
		cf.Set("licdsecure", "true")
		changed = true
	} else if !secure && cf.GetString("licdsecure") == "true" {
		cf.Set("licdsecure", "false")
		changed = true
	}

	// use getPorts() to check valid change, else go up one
	ports := instance.GetPorts(g.Host())
	nextport := instance.NextPort(g.Host(), &Gateway)
	if secure && cf.GetInt64("port") == 7039 {
		if _, ok := ports[7038]; !ok {
			cf.Set("port", 7038)
		} else {
			cf.Set("port", nextport)
		}
		changed = true
	} else if !secure && cf.GetInt64("port") == 7038 {
		if _, ok := ports[7039]; !ok {
			cf.Set("port", 7039)
		} else {
			cf.Set("port", nextport)
		}
		changed = true
	}

	if changed {
		if err = g.Config().Save(g.Type().String(),
			config.Host(g.Host()),
			config.SaveDir(instance.ParentDirectory(g)),
			config.SetAppName(g.Name()),
		); err != nil {
			return
		}
	}

	return instance.CreateConfigFromTemplate(g, filepath.Join(g.Home(), "gateway.setup.xml"), instance.Filename(g, "config::template"), template)
}

func (g *Gateways) Command() (args, env []string, home string) {
	cf := g.Config()

	// get opts from
	// from https://docs.itrsgroup.com/docs/geneos/5.10.0/Gateway_Reference_Guide/gateway_installation_guide.html#Gateway_command_line_options
	//
	args = []string{
		g.Name(),
		"-resources-dir",
		path.Join(instance.BaseVersion(g), "resources"),
		"-log",
		instance.LogFile(g),
		"-setup",
		path.Join(cf.GetString("home"), "gateway.setup.xml"),
		// enable stats by default
		"-stats",
	}

	_, version, err := instance.Version(g)
	if err == nil { // if we have a valid version test for additional features
		switch {
		case geneos.CompareVersion(version, "6.0.0") >= 0:
			log.Debug().Msg("version 6.0.0 or above, doing stuff")
			// use keyfiles etc.
		}

	}
	// check version
	// base, underlying, _ := instance.Version(g)
	// if underlying ... { }
	// "-gateway-name",

	if cf.GetString("gatewayname") != g.Name() {
		args = append([]string{cf.GetString("gatewayname")}, args...)
	}

	// We should not set port on command line. This is now done in the
	// instance template.

	// if cf.IsSet("port") {
	//  args = append([]string{"-port", fmt.Sprint(cf.GetString("port"))}, args...)
	// }

	if cf.GetString("licdhost") != "" {
		args = append(args, "-licd-host", cf.GetString("licdhost"))
	}

	if cf.GetInt64("licdport") != 0 {
		args = append(args, "-licd-port", fmt.Sprint(cf.GetString("licdport")))
	}

	args = append(args, instance.SetSecureArgs(g)...)

	// 3 options: set, set to false, not set
	if cf.GetBool("licdsecure") || (!cf.IsSet("licdsecure") && instance.Filename(g, "certificate") != "") {
		args = append(args, "-licd-secure")
	}

	if cf.GetBool("usekeyfile") {
		keyfile := instance.Filepath(g, "keyfile")
		if keyfile != "" {
			args = append(args, "-key-file", keyfile)
		}

		prevkeyfile := instance.Filepath(g, "prevkeyfile")
		if keyfile != "" {
			args = append(args, "-previous-key-file", prevkeyfile)
		}

	}

	home = g.Home()

	return
}

// create a gateway key file for secure passwords as per
// https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm
func createAESKeyFile(c geneos.Instance) (err error) {
	a := config.NewRandomKeyValues()

	w, err := c.Host().Create(instance.ComponentFilepath(c, "aes"), 0600)
	if err != nil {
		return
	}
	defer w.Close()
	if err = a.Write(w); err != nil {
		return
	}

	c.Config().Set("keyfile", instance.ComponentFilename(c, "aes"))
	return
}
