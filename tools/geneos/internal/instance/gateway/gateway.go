package gateway

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var Gateway = geneos.Component{
	Initialise:       Init,
	Name:             "gateway",
	RelatedTypes:     nil,
	ComponentMatches: []string{"gateway", "gateways"},
	RealComponent:    true,
	DownloadBase:     geneos.DownloadBases{Resources: "Gateway+2", Nexus: "geneos-gateway"},
	PortRange:        "GatewayPortRange",
	CleanList:        "GatewayCleanList",
	PurgeList:        "GatewayPurgeList",
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

//go:embed templates/gateway.setup.xml.gotmpl
var GatewayTemplate []byte

//go:embed templates/gateway-instance.setup.xml.gotmpl
var InstanceTemplate []byte

const GatewayDefaultTemplate = "gateway.setup.xml.gotmpl"
const GatewayInstanceTemplate = "gateway-instance.setup.xml.gotmpl"

func init() {
	geneos.RegisterComponent(&Gateway, New)
}

func Init(r *host.Host, ct *geneos.Component) {
	// copy default template to directory
	if err := geneos.MakeComponentDirs(r, ct); err != nil {
		log.Fatal().Err(err).Msg("")
	}
	if err := r.WriteFile(r.Filepath("gateway", "templates", GatewayDefaultTemplate), GatewayTemplate, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
	if err := r.WriteFile(r.Filepath("gateway", "templates", GatewayInstanceTemplate), InstanceTemplate, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

var gateways sync.Map

func New(name string) geneos.Instance {
	_, local, h := instance.SplitName(name, host.LOCAL)
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
		log.Fatal().Err(err).Msgf("%s setDefaults()")
	}
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

func (g *Gateways) Prefix() string {
	return "gate"
}

func (g *Gateways) Host() *host.Host {
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

func (g *Gateways) SetConf(v *config.Config) {
	g.Conf = v
}

func (g *Gateways) Add(username string, template string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(g.InstanceHost, &Gateway)
	}
	g.Config().Set("port", port)
	g.Config().Set("user", username)
	g.Config().Set("config.rebuild", "initial")

	g.Config().SetDefault("config.template", GatewayDefaultTemplate)
	if template != "" {
		filename, _ := instance.ImportCommons(g.Host(), g.Type(), "templates", []string{template})
		g.Config().Set("config.template", filename)
	}

	g.Config().Set("includes", make(map[int]string))

	// try to save config early
	if err = instance.WriteConfig(g); err != nil {
		log.Fatal().Err(err).Msg("")
		return
	}

	// check tls config, create certs if found
	if _, err = instance.ReadSigningCert(); err == nil {
		if err = instance.CreateCert(g); err != nil {
			return
		}
	}

	if err = createAESKeyFile(g); err != nil {
		return
	}

	return nil // g.Rebuild(true)
}

func (g *Gateways) Rebuild(initial bool) (err error) {
	// always rebuild an instance template
	err = instance.CreateConfigFromTemplate(g, filepath.Join(g.Home(), "instance.setup.xml"), GatewayInstanceTemplate, InstanceTemplate)
	if err != nil {
		return
	}

	configrebuild := g.Config().GetString("config.rebuild")

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
	if secure && g.Config().GetString("licdsecure") != "true" {
		g.Config().Set("licdsecure", "true")
		changed = true
	} else if !secure && g.Config().GetString("licdsecure") == "true" {
		g.Config().Set("licdsecure", "false")
		changed = true
	}

	// use getPorts() to check valid change, else go up one
	ports := instance.GetPorts(g.Host())
	nextport := instance.NextPort(g.Host(), &Gateway)
	if secure && g.Config().GetInt64("port") == 7039 {
		if _, ok := ports[7038]; !ok {
			g.Config().Set("port", 7038)
		} else {
			g.Config().Set("port", nextport)
		}
		changed = true
	} else if !secure && g.Config().GetInt64("port") == 7038 {
		if _, ok := ports[7039]; !ok {
			g.Config().Set("port", 7039)
		} else {
			g.Config().Set("port", nextport)
		}
		changed = true
	}

	if changed {
		if err = instance.WriteConfig(g); err != nil {
			return
		}
	}

	return instance.CreateConfigFromTemplate(g, filepath.Join(g.Home(), "gateway.setup.xml"), instance.Filename(g, "config.template"), GatewayTemplate)
}

func (g *Gateways) Command() (args, env []string) {
	// get opts from
	// from https://docs.itrsgroup.com/docs/geneos/5.10.0/Gateway_Reference_Guide/gateway_installation_guide.html#Gateway_command_line_options
	//
	args = []string{
		g.Name(),
		"-resources-dir",
		filepath.Join(g.Config().GetString("install"), g.Config().GetString("version"), "resources"),
		"-log",
		instance.LogFile(g),
		"-setup",
		filepath.Join(g.Config().GetString("home"), "gateway.setup.xml"),
		// enable stats by default
		"-stats",
	}

	// check version
	// base, underlying, _ := instance.Version(g)
	// if underlying ... { }
	// "-gateway-name",

	if g.Config().GetString("gatewayname") != g.Name() {
		args = append([]string{g.Config().GetString("gatewayname")}, args...)
	}

	args = append([]string{"-port", fmt.Sprint(g.Config().GetString("port"))}, args...)

	if g.Config().GetString("licdhost") != "" {
		args = append(args, "-licd-host", g.Config().GetString("licdhost"))
	}

	if g.Config().GetInt64("licdport") != 0 {
		args = append(args, "-licd-port", fmt.Sprint(g.Config().GetString("licdport")))
	}

	args = append(args, instance.SetSecureArgs(g)...)

	licdsecure := g.Config().GetString("licdsecure")
	if instance.Filename(g, "certificate") != "" {
		if licdsecure == "" || licdsecure != "false" {
			args = append(args, "-licd-secure")
		}
	} else if licdsecure != "" && licdsecure == "true" {
		args = append(args, "-licd-secure")
	}

	if g.Config().GetBool("usekeyfile") {
		keyfile := instance.Filepath(g, "keyfile")
		if keyfile != "" {
			args = append(args, "-key-file", keyfile)
		}

		prevkeyfile := instance.Filepath(g, "prevkeyfile")
		if keyfile != "" {
			args = append(args, "-previous-key-file", prevkeyfile)
		}

	}

	return
}

// create a gateway key file for secure passwords as per
// https://docs.itrsgroup.com/docs/geneos/4.8.0/Gateway_Reference_Guide/gateway_secure_passwords.htm
func createAESKeyFile(c geneos.Instance) (err error) {
	a, err := config.NewAESValues()
	if err != nil {
		return
	}

	w, err := c.Host().Create(instance.ComponentFilepath(c, "aes"), 0600)
	if err != nil {
		return
	}
	defer w.Close()
	if err = a.WriteAESValues(w); err != nil {
		return
	}

	c.Config().Set("keyfile", instance.ComponentFilename(c, "aes"))
	return
}
