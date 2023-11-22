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
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var Gateway = geneos.Component{
	Initialise:   initialise,
	Name:         "gateway",
	Aliases:      []string{"gateways"},
	LegacyPrefix: "gate",
	UsesKeyfiles: true,
	Templates: []geneos.Templates{
		{Filename: templateName, Content: template},
		{Filename: instanceTemplateName, Content: instanceTemplate},
	},
	DownloadBase: geneos.DownloadBases{Resources: "Gateway+2", Nexus: "geneos-gateway"},
	PortRange:    "GatewayPortRange",
	CleanList:    "GatewayCleanList",
	PurgeList:    "GatewayPurgeList",
	LegacyParameters: map[string]string{
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
		// order is important, do not change
		`binary=gateway2.linux_64`,
		`home={{join .root "gateway" "gateways" .name}}`,
		`install={{join .root "packages" "gateway"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=gateway.log`,
		`port=7039`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:/usr/lib64`,
		`gatewayname={{"${config:name}"}}`,
		`setup={{join "${config:home}" "gateway.setup.xml"}}`,
		`autostart=true`,
		`usekeyfile=false`,
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

const templateName = "gateway.setup.xml.gotmpl"

//go:embed templates/gateway-instance.setup.xml.gotmpl
var instanceTemplate []byte

const instanceTemplateName = "gateway-instance.setup.xml.gotmpl"

func init() {
	Gateway.Register(factory)
}

func initialise(r *geneos.Host, ct *geneos.Component) {
	// copy default template to directory
	if err := r.WriteFile(r.PathTo("gateway", "templates", templateName), template, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
	if err := r.WriteFile(r.PathTo("gateway", "templates", instanceTemplateName), instanceTemplate, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

var gateways sync.Map

// factory is the factory method for Gateways
func factory(name string) geneos.Instance {
	_, local, h := instance.SplitName(name, geneos.LOCAL)
	if local == "" || h == nil || (h == geneos.LOCAL && geneos.Root() == "") {
		return nil
	}
	if i, ok := gateways.Load(h.FullName(local)); ok {
		if g, ok := i.(*Gateways); ok {
			return g
		}
	}
	gateway := &Gateways{}
	gateway.Conf = config.New()
	gateway.Component = &Gateway
	gateway.InstanceHost = h
	if err := instance.SetDefaults(gateway, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", gateway)
	}
	// set the home dir based on where it might be, default to one above
	gateway.Config().Set("home", instance.Home(gateway))
	gateways.Store(h.FullName(local), gateway)
	return gateway
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
	return instance.Home(g)
}

func (g *Gateways) Host() *geneos.Host {
	return g.InstanceHost
}

func (g *Gateways) String() string {
	return instance.DisplayName(g)
}

func (g *Gateways) Load() (err error) {
	return instance.LoadConfig(g)
}

func (g *Gateways) Unload() (err error) {
	gateways.Delete(g.Name() + "@" + g.Host().String())
	g.ConfigLoaded = time.Time{}
	return
}

func (g *Gateways) Loaded() time.Time {
	return g.ConfigLoaded
}

func (g *Gateways) SetLoaded(t time.Time) {
	g.ConfigLoaded = t
}

func (g *Gateways) Config() *config.Config {
	return g.Conf
}

func (g *Gateways) Add(template string, port uint16) (err error) {
	cf := g.Config()

	if port == 0 {
		port = instance.NextPort(g.InstanceHost, &Gateway)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	cf.Set("port", port)
	cf.Set(cf.Join("config", "rebuild"), "initial")

	cf.SetDefault(cf.Join("config", "template"), templateName)
	if template != "" {
		filenames, _ := geneos.ImportCommons(g.Host(), g.Type(), "templates", []string{template})
		cf.Set(cf.Join("config", "template"), filenames[0])
	}

	cf.Set("includes", make(map[int]string))

	// try to save config early
	if err = instance.SaveConfig(g); err != nil {
		log.Fatal().Err(err).Msg("")
		return
	}

	// create certs, report success only
	resp := instance.CreateCert(g)
	if resp.Err == nil {
		fmt.Println(resp.Line)
	}

	// always create a keyfile ?
	if err = createAESKeyFile(g); err != nil {
		return
	}

	return nil
}

func (g *Gateways) Rebuild(initial bool) (err error) {
	cf := g.Config()

	// always rebuild an instance template
	err = instance.ExecuteTemplate(g, instance.Abs(g, "instance.setup.xml"), instanceTemplateName, instanceTemplate)
	if err != nil {
		return
	}
	log.Debug().Msgf("%s instance template %q rebuilt", g, "instance.setup.xml")

	configrebuild := cf.GetString("config::rebuild")

	setup := cf.GetString("setup")
	if configrebuild == "never" || setup == "" || setup == "none" {
		return
	}

	if !(configrebuild == "always" || (initial && configrebuild == "initial")) {
		return
	}

	if strings.HasPrefix(setup, "http:") || strings.HasPrefix(setup, "https:") {
		log.Debug().Msg("not rebuilding URL bases setup")
		return
	}

	// recheck check certs/keys
	var changed bool
	secure := instance.FileOf(g, "certificate") != "" && instance.FileOf(g, "privatekey") != ""

	// if we have certs then connect to Licd securely
	if secure && cf.GetString("licdsecure") != "true" {
		cf.Set("licdsecure", "true")
		changed = true
	} else if !secure && cf.GetString("licdsecure") == "true" {
		cf.Set("licdsecure", "false")
		changed = true
	}

	// use getPorts() to check valid change, else go up one
	ports := instance.GetAllPorts(g.Host())
	nextport := instance.NextPort(g.Host(), &Gateway)
	if nextport == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
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
		if err = instance.SaveConfig(g); err != nil {
			return
		}
	}

	return instance.ExecuteTemplate(g,
		setup,
		instance.FileOf(g, "config::template"),
		template)
}

func (g *Gateways) Command() (args, env []string, home string) {
	cf := g.Config()

	// first, get the correct name, using the "gatewayname" parameter if
	// it is different to the instance name. It may not be set, hence
	// the test.
	name := g.Name()
	if cf.GetString("gatewayname") != g.Name() {
		name = cf.GetString("gatewayname")
	}

	// if we have a valid version test for additional features
	_, version, err := instance.Version(g)
	if err == nil {
		switch {
		case geneos.CompareVersion(version, "5.10.0") >= 0:
			args = append(args, "-gateway-name", name)
		}
	} else {
		// fallback to older settings
		args = append(args,
			name,
		)
	}

	// always required
	args = append(args,
		"-resources-dir",
		path.Join(instance.BaseVersion(g), "resources"),
		"-log",
		instance.LogFilePath(g),
	)

	if cf.IsSet("gateway-hub") && cf.IsSet("obcerv") {
		err = errors.New("only one of 'obcerv' or 'gateway-hub' can be set")
		return
	}

	for _, k := range []string{"obcerv", "gateway-hub", "app-key", "kerberos-principal", "kerberos-keytab"} {
		if cf.IsSet(k) {
			args = append(args, "-"+k, cf.GetString(k))
		}
	}

	if setup := cf.GetString("setup"); !(setup == "" || setup == "none") {
		args = append(args, "-setup", setup)
	}

	if cf.GetString("licdhost") != "" {
		args = append(args, "-licd-host", cf.GetString("licdhost"))
	}

	if cf.GetInt64("licdport") != 0 {
		args = append(args, "-licd-port", fmt.Sprint(cf.GetString("licdport")))
	}

	args = append(args, instance.SetSecureArgs(g)...)

	// 3 options: set, set to false, not set
	if cf.GetBool("licdsecure") || (!cf.IsSet("licdsecure") && instance.FileOf(g, "certificate") != "") {
		args = append(args, "-licd-secure")
	}

	if cf.GetBool("usekeyfile") {
		keyfile := instance.PathOf(g, "keyfile")
		if keyfile != "" {
			args = append(args, "-key-file", keyfile)
		}

		prevkeyfile := instance.PathOf(g, "prevkeyfile")
		if prevkeyfile != "" {
			args = append(args, "-previous-key-file", prevkeyfile)
		}

	}

	home = g.Home()

	return
}

// create a gateway key file for secure passwords as per
// https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm
func createAESKeyFile(i geneos.Instance) (err error) {
	a := config.NewRandomKeyValues()

	w, err := i.Host().Create(instance.ComponentFilepath(i, "aes"), 0600)
	if err != nil {
		return
	}
	defer w.Close()
	if err = a.Write(w); err != nil {
		return
	}

	i.Config().Set("keyfile", instance.ComponentFilename(i, "aes"))
	return
}
