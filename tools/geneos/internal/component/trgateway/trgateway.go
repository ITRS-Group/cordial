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

package trgateway

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const name = "tr-gateway"

// Component TODO: Describe its purpose and intended use.
var Component = geneos.Component{
	ApplyProfile: profile.ApplyCommonProfile,
	Name:         name,
	Aliases:      []string{"trgateway", "trgw"},
	LegacyPrefix: "trgw",

	// Archive names are COMPONENT[-EXCHANGE]-VERSION[-OS-ARCH].SUFFIX,
	// e.g. `tr-gateway-idx-1.0.0-20261030.041812-1-linux-x64.tar.gz`.
	// The optional exchange is returned as the platform, which is
	// appended to the version (as `+idx`) so that each exchange installs
	// into its own package directory.
	DownloadNameRegexp: regexp.MustCompile(`^(?<component>` + name + `)(?:-(?<platform>[a-z][a-z0-9]*))?-(?<version>\d+(?:\.\d+){0,2}(?:-SNAPSHOT)?(?:-\d{8}\.\d+(?:-\d+)?)?)(?:-(?<os>linux|windows)(?:-\w+)?)?\.(?<suffix>tar\.gz|zip)$`),

	// accept `NAME-EXCHANGE` in package commands, e.g. `package install tr-gateway-idx`
	PackageVariants: true,

	DownloadParams: &[]string{
		"title=",
	},
	DownloadParamsNexus: &[]string{
		"maven.classifier=linux-x64",
		"maven.extension=tar.gz",
		"maven.groupId=com.itrsgroup.trgateway",
	},
	DownloadBase:  geneos.DownloadBases{Default: "TR+Gateway", Nexus: name},
	DownloadInfix: name,

	GlobalSettings: map[string]string{
		config.Join(name, "ports"): "19000-",
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
		`main-class=com.itrsgroup.trgateway.Main`,
		`autostart=true`,
	},

	Directories: []string{
		path.Join("packages", name),
		path.Join(name, name+"s"),
	},
	GetPID: pidCheckFn,
}

// TRGateway TODO: Describe its purpose and intended use.
type TRGateway instance.Instance

// ensure that Trgateway satisfies geneos.Instance interface
var _ geneos.Instance = (*TRGateway)(nil)

var trgateways sync.Map

func init() {
	Component.Register(factory)
}

func factory(name string) (i geneos.Instance) {
	return instance.Factory(name, &Component, &trgateways,
		func(h *geneos.Host, _ *geneos.Component) *TRGateway {
			return &TRGateway{Component: &Component, Conf: config.New(), InstanceHost: h}
		},
		nil,
		func(i *TRGateway) {
			i.Logger = instance.Logger(i)
			i.AuditLogger = instance.AuditLogger(i)
		},
	)
}

// Type TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

// Name TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

// Home TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) Home() string {
	if i == nil {
		return ""
	}
	return instance.Home(i)
}

// Host TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

// Log TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

// AuditEvent TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) AuditEvent(event string, result error, args ...any) {
	instance.AuditEvent(i, result, event, args...)
}

// String TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) String() string {
	return instance.DisplayName(i)
}

// Load TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) Load() error {
	return instance.Read(i)
}

// Unload TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) Unload() error {
	if i == nil {
		return nil
	}
	trgateways.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return nil
}

// Loaded TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

// SetLoaded TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

// Config TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

// SetConfig TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

// Add TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) Add(options ...any) error {
	if i == nil {
		return os.ErrInvalid
	}

	opts := instance.EvalAddOptions(options...)

	if opts.Port == 0 {
		opts.Port = instance.NextFreePort(i.InstanceHost, i.Type())
	}
	if opts.Port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	config.Set(i.Config(), "port", opts.Port)
	seedPackagedYAML(i)

	// create certs, report success only
	if !(opts.Insecure || opts.CertBundle != "") {
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(io.Discard))
	}
	return nil
}

// seedPackagedYAML copies config/{setup} from the installed package into the
// instance home when the instance does not already have a setup file.
func seedPackagedYAML(i *TRGateway) {
	instance.SeedPackagedSetup(i)
}

// Command TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	return instance.JavaGatewayCommand(i, skipFileCheck, "com.itrsgroup.trgateway.Main")
}

// Reload TODO: Document its behavior, inputs, and effects.
func (i *TRGateway) Reload() error {
	return geneos.ErrNotSupported
}

// Rebuild is not implemented for TRGateways and always returns false,
// geneos.ErrNotSupported.
func (i *TRGateway) Rebuild(initial bool) (changed bool, err error) {
	return false, geneos.ErrNotSupported
}

func pidCheckFn(arg any, cmdline []string) bool {
	g, ok := arg.(*TRGateway)
	return ok && instance.JavaPIDCheck(g, cmdline, "com.itrsgroup.trgateway.Main")
}
