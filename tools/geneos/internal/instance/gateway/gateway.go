package gateway

import (
	"crypto/rand"
	"crypto/sha1"
	_ "embed"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"golang.org/x/crypto/pbkdf2"
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
		"gateaes":   "aesfile",
		"gatename":  "gatewayname",
		"gatelich":  "licdhost",
		"gatelicp":  "licdport",
		"gatelics":  "licdsecure",
		"gateuser":  "user",
		"gateopts":  "options",
	},
	Defaults: []string{
		"binary=gateway2.linux_64",
		"home={{join .root \"gateway\" \"gateways\" .name}}",
		"install={{join .root \"packages\" \"gateway\"}}",
		"version=active_prod",
		"program={{join .install .version .binary}}",
		"logfile=gateway.log",
		"port=7039",
		"libpaths={{join .install .version \"lib64\"}}:/usr/lib64",
		"gatewayname={{.name}}",
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
		logger.Error.Fatalln(err)
	}
	if err := r.WriteFile(r.GeneosJoinPath("gateway", "templates", GatewayDefaultTemplate), GatewayTemplate, 0664); err != nil {
		logger.Error.Fatalln(err)
	}
	if err := r.WriteFile(r.GeneosJoinPath("gateway", "templates", GatewayInstanceTemplate), InstanceTemplate, 0664); err != nil {
		logger.Error.Fatalln(err)
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
		logger.Error.Fatalln(g, "setDefaults():", err)
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
	return g.V().GetString("name")
}

func (g *Gateways) Home() string {
	return g.V().GetString("home")
}

func (g *Gateways) Prefix() string {
	return "gate"
}

func (g *Gateways) Host() *host.Host {
	return g.InstanceHost
}

func (g *Gateways) String() string {
	return g.Type().String() + ":" + g.Name() + "@" + g.Host().String()
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

func (g *Gateways) V() *config.Config {
	return g.Conf
}

func (g *Gateways) SetConf(v *config.Config) {
	g.Conf = v
}

func (g *Gateways) Add(username string, template string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(g.InstanceHost, &Gateway)
	}
	g.V().Set("port", port)
	g.V().Set("user", username)
	g.V().Set("config.rebuild", "initial")

	g.V().SetDefault("config.template", GatewayDefaultTemplate)
	if template != "" {
		filename, _ := instance.ImportCommons(g.Host(), g.Type(), "templates", []string{template})
		g.V().Set("config.template", filename)
	}

	g.V().Set("includes", make(map[int]string))

	// try to save config early
	if err = instance.WriteConfig(g); err != nil {
		log.Fatalln(err)
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

	configrebuild := g.V().GetString("config.rebuild")

	if configrebuild == "never" {
		return
	}

	if !(configrebuild == "always" || (initial && configrebuild == "initial")) {
		return
	}

	// recheck check certs/keys
	var changed bool
	secure := g.V().GetString("certificate") != "" && g.V().GetString("privatekey") != ""

	// if we have certs then connect to Licd securely
	if secure && g.V().GetString("licdsecure") != "true" {
		g.V().Set("licdsecure", "true")
		changed = true
	} else if !secure && g.V().GetString("licdsecure") == "true" {
		g.V().Set("licdsecure", "false")
		changed = true
	}

	// use getPorts() to check valid change, else go up one
	ports := instance.GetPorts(g.Host())
	nextport := instance.NextPort(g.Host(), &Gateway)
	if secure && g.V().GetInt64("port") == 7039 {
		if _, ok := ports[7038]; !ok {
			g.V().Set("port", 7038)
		} else {
			g.V().Set("port", nextport)
		}
		changed = true
	} else if !secure && g.V().GetInt64("port") == 7038 {
		if _, ok := ports[7039]; !ok {
			g.V().Set("port", 7039)
		} else {
			g.V().Set("port", nextport)
		}
		changed = true
	}

	if changed {
		if err = instance.WriteConfig(g); err != nil {
			return
		}
	}

	return instance.CreateConfigFromTemplate(g, filepath.Join(g.Home(), "gateway.setup.xml"), g.V().GetString("config.template"), GatewayTemplate)
}

func (g *Gateways) Command() (args, env []string) {
	// get opts from
	// from https://docs.itrsgroup.com/docs/geneos/5.10.0/Gateway_Reference_Guide/gateway_installation_guide.html#Gateway_command_line_options
	//
	args = []string{
		g.Name(),
		"-resources-dir",
		filepath.Join(g.V().GetString("install"), g.V().GetString("version"), "resources"),
		"-log",
		instance.LogFile(g),
		"-setup",
		filepath.Join(g.V().GetString("home"), "gateway.setup.xml"),
		// enable stats by default
		"-stats",
	}

	// check version
	// base, underlying, _ := instance.Version(g)
	// if underlying ... { }
	// "-gateway-name",

	if g.V().GetString("gatewayname") != g.Name() {
		args = append([]string{g.V().GetString("gatewayname")}, args...)
	}

	args = append([]string{"-port", fmt.Sprint(g.V().GetString("port"))}, args...)

	if g.V().GetString("licdhost") != "" {
		args = append(args, "-licd-host", g.V().GetString("licdhost"))
	}

	if g.V().GetInt64("licdport") != 0 {
		args = append(args, "-licd-port", fmt.Sprint(g.V().GetString("licdport")))
	}

	if g.V().GetString("certificate") != "" {
		if g.V().GetString("licdsecure") == "" || g.V().GetString("licdsecure") != "false" {
			args = append(args, "-licd-secure")
		}
		args = append(args, "-ssl-certificate", g.V().GetString("certificate"))
		chainfile := g.Host().GeneosJoinPath("tls", "chain.pem")
		args = append(args, "-ssl-certificate-chain", chainfile)
	} else if g.V().GetString("licdsecure") != "" && g.V().GetString("licdsecure") == "true" {
		args = append(args, "-licd-secure")
	}

	if g.V().GetString("privatekey") != "" {
		args = append(args, "-ssl-certificate-key", g.V().GetString("privatekey"))
	}

	// if c.GateAES != "" {
	// 	args = append(args, "-key-file", c.GateAES)
	// }

	return
}

func (g *Gateways) Reload(params []string) (err error) {
	return instance.Signal(g, syscall.SIGUSR1)
}

// create a gateway key file for secure passwords as per
// https://docs.itrsgroup.com/docs/geneos/4.8.0/Gateway_Reference_Guide/gateway_secure_passwords.htm
func createAESKeyFile(c geneos.Instance) (err error) {
	rp := make([]byte, 20)
	salt := make([]byte, 10)
	if _, err = rand.Read(rp); err != nil {
		return
	}
	if _, err = rand.Read(salt); err != nil {
		return
	}

	md := pbkdf2.Key(rp, salt, 10000, 48, sha1.New)
	key := md[:32]
	iv := md[32:]

	if err = c.Host().WriteFile(instance.ComponentFilepath(c, "aes"), []byte(fmt.Sprintf("salt=%X\nkey=%X\niv =%X\n", salt, key, iv)), 0600); err != nil {
		return
	}
	c.V().Set("aesfile", c.Type().String()+".aes")
	return
}
