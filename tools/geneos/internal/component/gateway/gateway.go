/*
Copyright © 2022 ITRS Group

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

package gateway

import (
	_ "embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

const Name = "gateway"

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
	DownloadBase: geneos.DownloadBases{Default: "Gateway+2", Nexus: "geneos-gateway"},

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "7039,7100-",
		config.Join(Name, "clean"): strings.Join([]string{
			"*.history",
			"*.download",
		}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
			"*.snooze",
			"*.user_assignment",
			"stats.xml",
			"persistence.*",
			"licences.cache",
			"cache/",
			"database/",
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

var instances sync.Map

// factory is the factory method for Gateways
func factory(name string) (gateway geneos.Instance) {
	h, _, local := instance.Decompose(name)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}

	if g, ok := instances.Load(h.FullName(local)); ok {
		if gw, ok := g.(*Gateways); ok {
			return gw
		}
	}

	gateway = &Gateways{
		Component:    &Gateway,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(gateway, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", gateway)
	}

	// set the home dir based on where it might be, default to one above
	gateway.Config().Set("home", instance.Home(gateway))
	instances.Store(h.FullName(local), gateway)

	return
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
	instances.Delete(g.Name() + "@" + g.Host().String())
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
		port = instance.NextFreePort(g.InstanceHost, &Gateway)
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
	resp := instance.CreateCert(g, 0)
	if resp.Err == nil {
		fmt.Println(resp.Line)
	}

	// always create a keyfile ?
	if err = instance.CreateAESKeyFile(g); err != nil {
		return
	}

	if instance.CompareVersion(g, "5.14.0") >= 0 {
		// use keyfiles
		log.Debug().Msg("gateway version 5.14.0 or above, using keyfiles on creation")
		cf.Set("usekeyfile", "true")
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
	nextport := instance.NextFreePort(g.Host(), &Gateway)
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

func (g *Gateways) Command(checkExt bool) (args, env []string, home string, err error) {
	var checks []string

	cf := g.Config()
	home = g.Home()

	// first, get the correct name, using the "gatewayname" parameter if
	// it is different to the instance name. It may not be set, hence
	// the test.
	name := g.Name()
	if cf.GetString("gatewayname") != g.Name() {
		name = cf.GetString("gatewayname")
	}

	// if we have a valid version test for additional features
	if instance.CompareVersion(g, "5.10.0") >= 0 {
		args = append(args, g.Name(), "-gateway-name", name)
	} else {
		// fallback to older settings
		args = append(args, name)
	}

	// always required
	resourceDir := path.Join(instance.BaseVersion(g), "resources")
	logDir := filepath.Dir(instance.LogFilePath(g))
	checks = append(checks, resourceDir, logDir)

	args = append(args,
		"-resources-dir",
		resourceDir,
		"-log",
		instance.LogFilePath(g),
	)

	if cf.IsSet("gateway-hub") && cf.IsSet("obcerv") {
		// log.Debug().Msg("only one of 'obcerv' or 'gateway-hub' can be set")
		err = fmt.Errorf("%w: only one of 'obcerv' or 'gateway-hub' can be set", geneos.ErrInvalidArgs)
		return
	}

	for _, k := range []string{"obcerv", "gateway-hub", "app-key", "kerberos-principal", "kerberos-keytab"} {
		if cf.IsSet(k) {
			args = append(args, "-"+k, cf.GetString(k))
		}
		if k == "kerberos-keytab" {
			checks = append(checks, cf.GetString(k))
		}
	}

	if setup := cf.GetString("setup"); !(setup == "" || setup == "none") {
		args = append(args, "-setup", setup)
		checks = append(checks, setup)
	}

	if cf.GetString("licdhost") != "" {
		args = append(args, "-licd-host", cf.GetString("licdhost"))
	}

	if cf.GetInt64("licdport") != 0 {
		args = append(args, "-licd-port", fmt.Sprint(cf.GetString("licdport")))
	}

	secureArgs := instance.SetSecureArgs(g)
	args = append(args, secureArgs...)
	for _, arg := range secureArgs {
		if !strings.HasPrefix(arg, "-") {
			checks = append(checks, arg)
		}
	}

	// 3 options: set, set to false, not set
	if cf.GetBool("licdsecure") || (!cf.IsSet("licdsecure") && instance.FileOf(g, "certificate") != "") {
		args = append(args, "-licd-secure")
	}

	if cf.GetBool("usekeyfile") {
		keyfile := instance.PathOf(g, "keyfile")
		if keyfile != "" {
			args = append(args, "-key-file", keyfile)
			checks = append(checks, keyfile)
		}

		prevkeyfile := instance.PathOf(g, "prevkeyfile")
		if prevkeyfile != "" {
			args = append(args, "-previous-key-file", prevkeyfile)
			checks = append(checks, prevkeyfile)
		}
	}

	if checkExt {
		missing := instance.CheckPaths(g, checks)
		if len(missing) > 0 {
			err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
		}
	}

	return
}
