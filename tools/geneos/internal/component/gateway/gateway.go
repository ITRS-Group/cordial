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
	"io"
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
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

const Name = "gateway"

const (
	INSTANCEXML = "instance.setup.xml"
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
	DownloadBase: geneos.DownloadBases{Default: "Gateway+2", Nexus: "geneos-gateway"},

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "7038-7039,7100-",
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
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:/usr/lib64`,
		`gatewayname={{"${config:name}"}}`,
		`setup={{join "${config:home}" "gateway.setup.xml"}}`,
		`autostart=true`,
		`usekeyfile=false`,
	},

	Directories: []string{
		"packages/gateway",
		"gateway/config",
		"gateway/gateways",
		"gateway/includes",
		"gateway/shared",
		"gateway/templates",
	},
	SharedDirectories: []string{
		"gateway/config",
		"gateway/gateway_shared",
		"gateway/gateway_config",
		"gateway/includes",
		"gateway/shared",
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
	h, _, local := instance.ParseName(name)

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
	config.Set(gateway.Config(), "home", instance.Home(gateway))
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
	return config.Get[string](g.Config(), "name")
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
	return instance.Read(g)
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

func (g *Gateways) Add(template string, port uint16, noCerts bool) (err error) {
	cf := g.Config()

	if port == 0 {
		port = instance.NextFreePort(g.InstanceHost, &Gateway)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	config.Set(cf, "port", port)
	config.Set(cf, cf.Join("config", "rebuild"), "initial")

	cf.Default(cf.Join("config", "template"), templateName)
	if template != "" {
		filenames, _ := geneos.ImportCommons(g.Host(), g.Type(), "templates", []string{template})
		config.Set(cf, cf.Join("config", "template"), filenames[0])
	}

	config.Set(cf, "includes", make(map[int]string))

	// try to save config early
	if err = instance.Write(g); err != nil {
		log.Fatal().Err(err).Msg("")
		return
	}

	// create certs, report success only
	if !noCerts {
		instance.NewCertificate(g).Report(os.Stdout, responses.StderrWriter(io.Discard))
	}

	// always create a keyfile ?
	if err = instance.CreateAESKeyFile(g); err != nil {
		return
	}

	if instance.CompareVersion(g, "5.14.0") >= 0 {
		// use keyfiles
		log.Debug().Msg("gateway version 5.14.0 or above, using keyfiles on creation")
		config.Set(cf, "usekeyfile", "true")
	}

	return nil
}

func (g *Gateways) Rebuild(initial bool) (err error) {
	cf := g.Config()

	// always rebuild an instance template
	log.Debug().Msgf("rebuilding %s instance template %q with config %#v", g, instanceTemplateName, cf.AllSettings())
	err = instance.ExecuteTemplate(g, instance.Abs(g, INSTANCEXML), instanceTemplateName, instanceTemplate, 0444)
	if err != nil {
		return
	}
	log.Debug().Msgf("%s instance template %q rebuilt", g, INSTANCEXML)

	configrebuild := config.Get[string](cf, "config::rebuild")

	setup := config.Get[string](cf, "setup")
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

	certPath := instance.PathTo(g, cf.Join("tls", "certificate"))
	if certPath == "" {
		certPath = instance.PathTo(g, "certificate")
	}
	keyPath := instance.PathTo(g, cf.Join("tls", "privatekey"))
	if keyPath == "" {
		keyPath = instance.PathTo(g, "privatekey")
	}

	secure := certPath != "" && keyPath != ""
	log.Debug().Msgf("gateway cert: %q key: %q secure: %v", certPath, keyPath, secure)

	// if we have certs then connect to Licd securely
	if secure && config.Get[string](cf, "licdsecure") != "true" {
		config.Set(cf, "licdsecure", "true")
		changed = true
	} else if !secure && config.Get[string](cf, "licdsecure") == "true" {
		config.Set(cf, "licdsecure", "false")
		changed = true
	}

	// use getPorts() to check valid change, else go up one
	ports := instance.GetAllPorts(g.Host())
	nextport := instance.NextFreePort(g.Host(), &Gateway)
	if nextport == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	if secure && config.Get[uint16](cf, "port") == 7039 {
		if _, ok := ports[7038]; !ok {
			config.Set[uint16](cf, "port", 7038)
		} else {
			config.Set(cf, "port", nextport)
		}
		changed = true
	} else if !secure && config.Get[uint16](cf, "port") == 7038 {
		if _, ok := ports[7039]; !ok {
			config.Set[uint16](cf, "port", 7039)
		} else {
			config.Set(cf, "port", nextport)
		}
		changed = true
	}

	if changed {
		if err = instance.Write(g); err != nil {
			log.Error().Err(err).Msg("Cannot save configuration")
			return
		}
	}

	return instance.ExecuteTemplate(g,
		setup,
		instance.FileOf(g, "config::template"),
		template,
		0664,
	)
}

func (i *Gateways) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	cf := i.Config()
	home = i.Home()

	// first, get the correct name, using the "gatewayname" parameter if
	// it is different to the instance name. It may not be set, hence
	// the test.
	name := i.Name()
	if config.Get[string](cf, "gatewayname") != i.Name() {
		name = config.Get[string](cf, "gatewayname")
	}

	// if we have a valid version test for additional features
	if instance.CompareVersion(i, "5.10.0") >= 0 {
		args = append(args, i.Name(), "-gateway-name", name)
	} else {
		// fallback to older settings
		args = append(args, name)
	}

	// always required
	resourceDir := path.Join(instance.BaseVersion(i), "resources")
	logDir := filepath.Dir(instance.LogFilePath(i))
	checks = append(checks, resourceDir, logDir)

	args = append(args,
		"-resources-dir",
		resourceDir,
		"-log",
		instance.LogFilePath(i),
	)

	if cf.IsSet("gateway-hub") && cf.IsSet("obcerv") {
		// log.Debug().Msg("only one of 'obcerv' or 'gateway-hub' can be set")
		err = fmt.Errorf("%w: only one of 'obcerv' or 'gateway-hub' can be set", geneos.ErrInvalidArgs)
		return
	}

	for _, k := range []string{"obcerv", "gateway-hub", "app-key", "kerberos-principal", "kerberos-keytab"} {
		if v, ok := config.Lookup[string](cf, k); ok {
			args = append(args, "-"+k, v)
		}
		if k == "kerberos-keytab" {
			checks = append(checks, config.Get[string](cf, k))
		}
	}

	if setup := config.Get[string](cf, "setup"); !(setup == "" || setup == "none") {
		args = append(args, "-setup", setup)
		checks = append(checks, setup)
	}

	if config.Get[string](cf, "licdhost") != "" {
		args = append(args, "-licd-host", config.Get[string](cf, "licdhost"))
	}

	if licdport := config.Get[uint16](cf, "licdport"); licdport != 0 {
		args = append(args, "-licd-port", fmt.Sprint(licdport))
	}

	// secureArgs := instance.SetSecureArgs(i)
	secureArgs, secureEnvs, fileChecks, err := instance.SecureArgs(i)
	if err != nil {
		return
	}
	args = append(args, secureArgs...)
	env = append(env, secureEnvs...)
	checks = append(checks, fileChecks...)

	// 3 options: set, set to false, not set
	if config.Get[bool](cf, "licdsecure") || (!cf.IsSet("licdsecure") && (instance.FileOf(i, cf.Join("tls", "certificate")) != "" || instance.FileOf(i, "certificate") != "")) {
		args = append(args, "-licd-secure")
	}

	if config.Get[bool](cf, "usekeyfile") {
		keyfile := instance.PathTo(i, "keyfile")
		if keyfile != "" {
			args = append(args, "-key-file", keyfile)
			checks = append(checks, keyfile)
		}

		prevkeyfile := instance.PathTo(i, "prevkeyfile")
		if prevkeyfile != "" {
			args = append(args, "-previous-key-file", prevkeyfile)
			checks = append(checks, prevkeyfile)
		}
	}

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}
