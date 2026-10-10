/*
Copyright © 2026 ITRS Group

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

package mdgateway

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

const name = "md-gateway"

var Component = geneos.Component{
	Name:         name,
	Aliases:      []string{"mdgateway", "mdgw"},
	LegacyPrefix: "mdgw",

	DownloadParams: &[]string{
		"title=",
	},
	DownloadParamsNexus: &[]string{
		"maven.classifier=linux-x64",
		"maven.extension=tar.gz",
		"maven.groupId=com.itrsgroup.mdgateway",
	},
	DownloadBase:  geneos.DownloadBases{Default: "MD+Gateway", Nexus: name},
	DownloadInfix: name,

	GlobalSettings: map[string]string{
		config.Join(name, "ports"): "18000-",
		config.Join(name, "clean"): strings.Join([]string{}, ":"),
		config.Join(name, "purge"): strings.Join([]string{}, ":"),
	},
	PortRange: config.Join(name, "ports"),
	CleanList: config.Join(name, "clean"),
	PurgeList: config.Join(name, "purge"),
	ConfigAliases: map[string]string{
		config.Join(name, "ports"): name + "portrange",
		config.Join(name, "clean"): name + "cleanlist",
		config.Join(name, "purge"): name + "purgelist",
	},

	LegacyParameters: map[string]string{},
	Defaults: []string{
		`binary=java`, // needed for 'ps' matching
		`home={{join .root "` + name + `" "` + name + `s" .name}}`,
		`install={{join .root "packages" "` + name + `"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "jdk" "bin" "java"}}`,
		`logback={{join "${config:install}" "${config:version}" "config" "logback.xml"}}`,
		`logfile=` + name + `.log`,
		`auditlogfile=` + name + `-audit.log`,
		`setup={{join "${config:home}" "` + name + `.yaml"}}`,
		`jar=lib/` + name + `.jar`,
		`main-class=com.itrsgroup.mdgateway.Main`,
		`autostart=true`,
	},

	Directories: []string{
		filepath.Join("packages", name),
		filepath.Join(name, name+"s"),
	},
	GetPID: pidCheckFn,
}

type MDGateway instance.Instance

// ensure that MDGateway satisfies geneos.Instance interface
var _ geneos.Instance = (*MDGateway)(nil)

var mdgateways sync.Map

func init() {
	Component.Register(factory)
}

func factory(name string) (i geneos.Instance) {
	if name == "" {
		return nil
	}
	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h.IsLocalhost() && geneos.LocalRoot() == "") {
		return nil
	}

	if s, ok := mdgateways.Load(h.FullName(local)); ok {
		if ss, ok := s.(*MDGateway); ok {
			return ss
		}
	}

	i = &MDGateway{
		Component:    &Component,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(i, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", i, err))
	}

	// set the home dir based on where it might be, default to one above
	config.Set(i.Config(), "home", instance.Home(i))
	i.(*MDGateway).Logger = instance.Logger(i)
	i.(*MDGateway).AuditLogger = instance.AuditLogger(i)
	mdgateways.Store(h.FullName(local), i)

	return
}

func (i *MDGateway) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *MDGateway) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *MDGateway) Home() string {
	if i == nil {
		return ""
	}
	return instance.Home(i)
}

func (i *MDGateway) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *MDGateway) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *MDGateway) AuditEvent(event string, args ...any) {
	instance.AuditEvent(i, event, args...)
}

func (i *MDGateway) String() string {
	return instance.DisplayName(i)
}

func (i *MDGateway) Load() error {
	return instance.Read(i)
}

func (i *MDGateway) Unload() error {
	if i == nil {
		return nil
	}
	mdgateways.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return nil
}

func (i *MDGateway) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *MDGateway) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *MDGateway) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *MDGateway) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *MDGateway) Add(_ string, port uint16, noCerts bool) error {
	if i == nil {
		return os.ErrInvalid
	}
	if port == 0 {
		port = instance.NextFreePort(i.InstanceHost, i.Type())
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	config.Set(i.Config(), "port", port)
	seedPackagedYAML(i)
	return nil
}

// seedPackagedYAML copies config/{setup} from the installed package into the
// instance home when the instance does not already have a setup file.
func seedPackagedYAML(i *MDGateway) {
	if i == nil {
		return
	}
	setup := instance.PathTo(i, "setup")
	if setup == "" {
		return
	}
	h := i.Host()
	if _, err := h.Stat(setup); err == nil {
		return
	}
	src := path.Join(instance.BaseVersion(i), "config", path.Base(setup))
	data, err := h.ReadFile(src)
	if err != nil {
		return
	}
	if err := h.WriteFile(setup, data, 0664); err != nil {
		return
	}
	i.AuditEvent("import", slog.Any("file", path.Base(setup)))
}

func (i *MDGateway) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	if i == nil {
		err = os.ErrInvalid
		return
	}

	cf := i.Config()
	home = i.Home()
	base := instance.BaseVersion(i)

	jar := config.Get[string](cf, "jar")
	mainClass := config.Get[string](cf, "main-class", config.PromoteFrom("mainclass"), config.DefaultValue("com.itrsgroup.mdgateway.Main"))
	setup := instance.PathTo(i, "setup")

	jarPath := path.Join(base, jar)
	pluginsDir := path.Join(base, "plugins")
	classpath := jarPath + ":" + path.Join(pluginsDir, "*")

	args = []string{
		"--enable-native-access=ALL-UNNAMED",
		"-Djava.net.preferIPv4Stack=true",
	}

	logback := config.Get[string](cf, "logback")
	if logback != "" {
		args = append(args, "-Dlogback.configurationFile="+logback)
	}

	args = append(args,
		"-Xms"+strings.TrimPrefix(config.Get[string](cf, "xms", config.DefaultValue("512m")), "-Xms"),
		"-Xmx"+strings.TrimPrefix(config.Get[string](cf, "xmx", config.DefaultValue("512m")), "-Xmx"),
		"-XX:+UseG1GC",
		"-Dapp.home="+home,
	)

	args = append(args,
		strings.Fields(config.Get[string](cf, "java-options"))...,
	)

	args = append(args,
		"-cp", classpath,
		mainClass,
		setup,
	)

	// this is overridden if the instance env var is set, as those are
	// added after this function is called and `cmd.Env` only uses the
	// last value of each named var
	env = []string{"JAVA_HOME=" + path.Join(base, "jdk")}

	logFile := instance.LogFilePath(i)
	checks = append(checks, path.Dir(logFile), jarPath, pluginsDir, setup)
	if logback != "" {
		checks = append(checks, logback)
	}

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks...)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (i *MDGateway) Reload() error {
	return geneos.ErrNotSupported
}

// Rebuild is not supported for MDGateways.
func (i *MDGateway) Rebuild(initial bool) (changed bool, err error) {
	return false, geneos.ErrNotSupported
}

func pidCheckFn(arg any, cmdline []string) bool {
	g, ok := arg.(*MDGateway)
	if !ok || g == nil {
		return false
	}
	if path.Base(cmdline[0]) != "java" && path.Base(cmdline[0]) != "java.exe" {
		return false
	}

	home := g.Home()
	mainClass := config.Get[string](g.Config(), "main-class", config.PromoteFrom("mainclass"), config.DefaultValue("com.itrsgroup.mdgateway.Main"))
	setup := instance.PathTo(g, "setup")

	var homeOK, appOK bool
	for _, a := range cmdline[1:] {
		if a == "-Dapp.home="+home || a == setup {
			homeOK = true
		}
		if a == mainClass {
			appOK = true
		}
		if homeOK && appOK {
			return true
		}
	}
	return false
}
