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

// Package appgw implements a shared Java application-gateway component
// used by md-gateway and tr-gateway.
package appgw

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/audit"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// archiveNameRegexp matches released md-gateway / tr-gateway archives, including
// Maven unique snapshots and an optional OS classifier, for example:
//
//	md-gateway-1.0.0.tar.gz
//	md-gateway-1.0.0-SNAPSHOT-linux-x64.tar.gz
//	md-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz
func archiveNameRegexp(component string) *regexp.Regexp {
	return regexp.MustCompile(`^(?<component>` + regexp.QuoteMeta(component) +
		`)-(?<version>\d+(?:\.\d+){0,2}(?:-SNAPSHOT)?(?:-\d{8}\.\d+(?:-\d+)?)?)(?:-(?<os>linux|windows)(?:-\w+)?)?\.(?<suffix>tar\.gz|tgz|gz|zip)$`)
}

// Spec describes a Java application-gateway component type.
type Spec struct {
	Name      string
	Aliases   []string
	Jar       string
	MainClass string
	SetupFile string
	LogFile   string
	Ports     string
}

// AppGW is a runnable instance of an application-gateway component.
type AppGW struct {
	instance.Instance
}

var _ geneos.Instance = (*AppGW)(nil)

var instances sync.Map

func instanceKey(ct *geneos.Component, h *geneos.Host, local string) string {
	if ct == nil || h == nil {
		return local
	}
	return ct.Name + ":" + h.FullName(local)
}

// Register creates and registers a component from spec.
func Register(spec Spec) *geneos.Component {
	if spec.Ports == "" {
		spec.Ports = "18000-"
	}
	if spec.LogFile == "" {
		spec.LogFile = spec.Name + ".log"
	}

	plural := spec.Name + "s"
	nameKey := spec.Name

	ct := &geneos.Component{
		Name:    spec.Name,
		Aliases: spec.Aliases,
		// Released archives wrap contents in `{name}-{version}[-linux-x64]/`.
		ArchiveLeaveFirstDir: false,
		DownloadNameRegexp:   archiveNameRegexp(spec.Name),

		GlobalSettings: map[string]string{
			config.Join(nameKey, "ports"):          spec.Ports,
			config.Join(nameKey, "clean"):          "",
			config.Join(nameKey, "purge"):          strings.Join([]string{"*.log", audit.LogFile(spec.Name)}, ":"),
			config.Join(nameKey, "audit-log-file"): audit.LogFile(spec.Name),
		},
		PortRange: config.Join(nameKey, "ports"),
		CleanList: config.Join(nameKey, "clean"),
		PurgeList: config.Join(nameKey, "purge"),
		ConfigAliases: map[string]string{
			config.Join(nameKey, "ports"): nameKey + "portrange",
			config.Join(nameKey, "clean"): nameKey + "cleanlist",
			config.Join(nameKey, "purge"): nameKey + "purgelist",
		},
		Defaults: []string{
			`binary=java`,
			`home={{join .root "` + spec.Name + `" "` + plural + `" .name}}`,
			`install={{join .root "packages" "` + spec.Name + `"}}`,
			`version=active_prod`,
			`program={{join "${config:install}" "${config:version}" "jdk" "bin" "java"}}`,
			`logback={{join "${config:install}" "${config:version}" "config" "logback.xml"}}`,
			`logfile=` + spec.LogFile,
			`setup={{join "${config:home}" "` + spec.SetupFile + `"}}`,
			`jar=` + spec.Jar,
			`mainclass=` + spec.MainClass,
			`xms=512m`,
			`xmx=512m`,
			`autostart=true`,
			`audit-log=` + audit.LogFile(spec.Name),
		},
		Directories: []string{
			"packages/" + spec.Name,
			spec.Name + "/" + plural,
		},
		GetPID:   pidCheckFn,
		OnImport: audit.AuditImport,
		Audit:    audit.AuditEvent,
	}

	ct.Register(factory(ct))
	return ct
}

func factory(ct *geneos.Component) func(string) geneos.Instance {
	return func(name string) geneos.Instance {
		if name == "" {
			return nil
		}
		h, _, local := instance.ParseName(name)

		if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
			return nil
		}

		key := instanceKey(ct, h, local)
		if existing, ok := instances.Load(key); ok {
			if g, ok := existing.(*AppGW); ok {
				return g
			}
		}

		g := &AppGW{
			Instance: instance.Instance{
				Component:    ct,
				Conf:         config.New(),
				InstanceHost: h,
			},
		}

		if err := instance.SetDefaults(g, local); err != nil {
			panic(fmt.Sprintf("%s setDefaults(): %v", g, err))
		}

		config.Set(g.Config(), "home", instance.Home(g))
		g.Logger = instance.NewLogger(g)
		instances.Store(key, g)
		return g
	}
}

func (i *AppGW) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *AppGW) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *AppGW) Home() string {
	if i == nil {
		return ""
	}
	return instance.Home(i)
}

func (i *AppGW) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *AppGW) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *AppGW) String() string {
	return instance.DisplayName(i)
}

func (i *AppGW) Load() error {
	return instance.Read(i)
}

func (i *AppGW) Unload() error {
	if i == nil {
		return nil
	}
	instances.Delete(instanceKey(i.Type(), i.Host(), i.Name()))
	i.ConfigLoaded = time.Time{}
	return nil
}

func (i *AppGW) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *AppGW) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *AppGW) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *AppGW) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *AppGW) Add(_ string, port uint16, _ bool) error {
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
func seedPackagedYAML(i *AppGW) {
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
	_ = audit.AuditImport(i, path.Base(setup))
}

func (i *AppGW) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	if i == nil {
		err = os.ErrInvalid
		return
	}

	cf := i.Config()
	home = i.Home()
	base := instance.BaseVersion(i)

	jar := config.Get[string](cf, "jar")
	mainClass := config.Get[string](cf, "mainclass")
	setup := instance.PathTo(i, "setup")
	xms := config.Get[string](cf, "xms", config.DefaultValue("512m"))
	xmx := config.Get[string](cf, "xmx", config.DefaultValue("512m"))

	jarPath := path.Join(base, jar)
	pluginsDir := path.Join(base, "plugins")
	classpath := jarPath + ":" + path.Join(pluginsDir, "*")
	javaHome := path.Join(base, "jdk")
	logback := config.Get[string](cf, "logback")

	args = []string{
		"--enable-native-access=ALL-UNNAMED",
		"-Djava.net.preferIPv4Stack=true",
	}
	if logback != "" {
		args = append(args, "-Dlogback.configurationFile="+logback)
	}
	args = append(args,
		"-Xms"+strings.TrimPrefix(xms, "-Xms"),
		"-Xmx"+strings.TrimPrefix(xmx, "-Xmx"),
		"-XX:+UseG1GC",
		"-Dapp.home="+home,
	)

	javaopts := strings.Fields(config.Get[string](cf, "java-options"))
	args = append(args, javaopts...)

	args = append(args,
		"-cp", classpath,
		mainClass,
		setup,
	)

	env = []string{"JAVA_HOME=" + javaHome}

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

func (i *AppGW) Reload() error {
	return geneos.ErrNotSupported
}

func (i *AppGW) Rebuild(bool) error {
	return nil
}

func pidCheckFn(arg any, cmdline []string) bool {
	g, ok := arg.(*AppGW)
	if !ok || g == nil {
		return false
	}
	if path.Base(cmdline[0]) != "java" && path.Base(cmdline[0]) != "java.exe" {
		return false
	}

	home := g.Home()
	mainClass := config.Get[string](g.Config(), "mainclass")
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
