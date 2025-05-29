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

package geneos

import (
	"iter"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/rs/zerolog/log"
)

// initDirs is a map of component type name to a slice of directories to create
var initDirs = make(map[string][]string)

// slice of registered component types for indirect calls
// this should actually become an Interface
var registeredComponents = make(componentsMap)

const RootComponentName = "root"

const sharedSuffix = "_shared"

var RootComponent = Component{
	Name:         RootComponentName,
	PackageTypes: nil,
	Aliases:      []string{"any"},
	DownloadBase: DownloadBases{Default: "", Nexus: ""},
	GlobalSettings: map[string]string{
		// Root URL for all downloads of software archives
		// hardwire delimiters because config may not be initialised for config.Join()
		"download::url": defaultURL,

		// Path List separated additions to the reserved names list, over and above
		// any words matched by FindComponent()
		"reservednames": "all",

		"privatekeys": "id_rsa,id_ecdsa,id_ecdsa_sk,id_ed25519,id_ed25519_sk,id_dsa",
	},
	Directories: []string{
		"packages/downloads",
	},
}

// definitions and access methods for the generic component types

type componentsMap map[string]*Component

// DownloadBases define the base names for the download archived for
// standard and nexus downloads
type DownloadBases struct {
	Default string
	Nexus   string
}

// Templates define the filename and embedded content for template files
type Templates struct {
	Filename string
	Content  []byte
}

// Component defines a registered component
type Component struct {
	// Name of a component
	Name string

	// Aliases are any names for the component (including the Name, above)
	Aliases []string

	// LegacyPrefix is the three or four letter prefix from legacy `ctl`
	// commands
	LegacyPrefix string

	// LegacyParameters is a map of legacy parameters (including prefix)
	// to new parameter names
	LegacyParameters map[string]string

	// ParentType, if set, is the parent component that the component is
	// under, e.g. 'fa2' has a parent of 'netprobe'. See PackageTypes
	// below.
	ParentType *Component

	// PackageTypes is a list of packages that can be used to support
	// this component. For example, a 'san' could be either a plain
	// 'netprobe' or an 'fa2'. The entries must reference components
	// that do not have PackageTypes set to avoid recursion.
	PackageTypes []*Component

	// DownloadBase lists, for each download source, the path to append
	// to the download URL (without any query string)
	DownloadBase DownloadBases

	// DownloadInfix is the component name in the downloaded file and if
	// set replace this string with component name for download matches,
	// e.g. `geneos-["desktop-activeconsole"]-VERSION...` -> `ac2`
	DownloadInfix string

	// DownloadNameRegexp is a custom regular expression to extract
	// archive information from the download file
	DownloadNameRegexp *regexp.Regexp

	// DownloadParams are custom parameters for the download URL. If nil
	// (not len zero) then os=linux is used
	DownloadParams *[]string

	// DownloadParamsNexus are custom parameters for Nexus downloads. If
	// nil then the defaults are:
	//
	//   maven.classifier=linux-x64
	//   maven.extension=tar.gz
	//   maven.groupId=com.itrsgroup.geneos
	DownloadParamsNexus *[]string

	// StripArchivePrefix, if not nil, is the strings removed from the
	// path of each file in the release archive. This is a pointer so
	// that an empty string can be used (i.e. for webservers)
	// StripArchivePrefix *string

	ArchiveLeaveFirstDir bool

	// Defaults are name=value templates that are "run" for each new
	// instance
	//
	// They are run in order, as later defaults may depend on earlier
	// settings, and so this cannot be a map
	Defaults []string

	UsesKeyfiles bool

	// Directories to be created (under Geneos home) on initialisation
	Directories []string

	// Templates are any templates for the component. Each Template has
	// a filename and default content
	Templates []Templates

	GlobalSettings map[string]string

	PortRange string
	CleanList string
	PurgeList string

	// ConfigAliases maps new configuration parameters to the original
	// names, e.g. "netprobe::ports" -> "netprobeportrange"
	//
	// We can use this to support older geneos.json config while
	// migrating
	ConfigAliases map[string]string

	// Functions

	// Initialise a component. Callback after the component is registered
	Initialise func(*Host, *Component)

	// New is the factory method for the component. It has to be added
	// during initialisation to avoid loops
	New func(string) Instance

	// GetPID returns the process ID of an instance - if nil a standard
	// function is used
	GetPID func(arg any, cmdline ...[]byte) bool // if set, use this to get the PID of an instance
}

// Instance interfaces contains the method set for an instance of a
// registered Component
type Instance interface {
	Config() *config.Config

	// Name returns the base instance name
	Name() string

	// Home returns the path to the instance directory
	Home() string

	// Type returns a Component type for the instance
	Type() *Component

	// Host returns the Host for the instance
	Host() *Host

	// String returns the display name of the instance in the form `TYPE
	// NAME` for local instances and `TYPE NAME@HOST` for those on
	// remote hosts
	String() string

	// config
	Load() error
	Unload() error
	Loaded() time.Time
	SetLoaded(time.Time)

	// actions
	Add(template string, port uint16) error
	Command(bool) ([]string, []string, string, error)
	Reload() (err error)
	Rebuild(bool) error
}

// Register adds the given Component ct to the internal list of
// component types. The factory parameter is the Component's New()
// function, which has to be passed in to avoid initialisation loops as
// the function refers to the type being registered.
func (ct *Component) Register(factory func(string) Instance) {
	if ct == nil {
		return
	}
	ct.New = factory
	registeredComponents[ct.Name] = ct
	initDirs[ct.Name] = ct.Directories
	for k, v := range ct.GlobalSettings {
		config.GetConfig().SetDefault(k, v)
	}
}

func (ct *Component) String() (name string) {
	if ct == nil {
		return ""
	}
	return ct.Name
}

// IsA returns true is any of the names match the any of the names
// defined in ComponentMatches. The check is case-insensitive.
func (ct *Component) IsA(names ...string) bool {
	for _, a := range append([]string{ct.Name}, ct.Aliases...) {
		for _, b := range names {
			if strings.EqualFold(a, b) {
				return true
			}
		}
	}
	return false
}

// MakeDirs creates the directory structure for the component ct.
// If ct is nil then the Root component type is used. If there is an
// error creating the directory then the error is immediately returned
// and the list of directories may only be partially created.
func (ct *Component) MakeDirs(h *Host) (err error) {
	name := RootComponentName
	if h == ALL {
		log.Fatal().Msg("called with all hosts")
	}
	if ct != nil {
		name = ct.Name
	}
	geneos := h.GetString(cordial.ExecutableName()) // root for host h
	for _, d := range initDirs[name] {
		dir := path.Join(geneos, d)
		if err = h.MkdirAll(dir, 0775); err != nil {
			return
		}
	}
	return
}

// InstancesDir return the parent directory for the instances of a
// component
func (ct *Component) InstancesDir(h *Host) (dir string) {
	if ct == nil {
		return ""
	}
	dir = h.PathTo(ct, ct.Name+"s")
	return
}

// InstancesBaseDirs returns a list of possible instance directories to
// look for an instance.
func (ct *Component) InstancesBaseDirs(h *Host) (dirs []string) {
	if ct == nil {
		return
	}
	// this check is to support older installations
	if ct.ParentType != nil {
		dirs = append(dirs, h.PathTo(ct.ParentType, ct.Name+"s"))
	}
	dirs = append(dirs, h.PathTo(ct.Name, ct.Name+"s"))
	return
}

// Shared return a path to a location in the shared directory for the
// component on host h joined to subdirectories and file given as subs.
func (ct *Component) Shared(h *Host, subs ...interface{}) string {
	if ct == nil {
		return ""
	}
	parts := []interface{}{}
	if ct.ParentType == nil {
		parts = append(parts, ct, ct.Name+sharedSuffix, subs)
	} else {
		parts = append(parts, ct.ParentType, ct.Name+sharedSuffix, subs)
	}
	return h.PathTo(parts...)
}

// OrList returns and iterator for component types for the first non-nil
// value of: the method receiver or the list of component types passed as
// args. If no arguments are passed then all non-root components are
// returned.
func (ct *Component) OrList(cts ...*Component) iter.Seq[*Component] {
	return func(yield func(*Component) bool) {
		if ct != nil {
			yield(ct)
			return
		}

		if len(cts) == 0 {
			for _, c := range registeredComponents {
				if c == &RootComponent {
					continue
				}
				if !yield(c) {
					return
				}
			}
			return
		}

		for _, c := range cts {
			if c == &RootComponent {
				continue
			}
			if !yield(c) {
				return
			}
		}
	}
}

// AllComponents returns a slice of all registered components, include
// the Root component type
func AllComponents() (cts []*Component) {
	for _, c := range registeredComponents {
		cts = append(cts, c)
	}
	return
}

// RealComponents returns a slice of all registered components that are
// not the root, sorted by name
func RealComponents() (cts []*Component) {
	for _, c := range registeredComponents {
		if c != &RootComponent {
			cts = append(cts, c)
		}
	}
	slices.SortFunc(cts, func(i, j *Component) int { return strings.Compare(i.Name, j.Name) })

	return
}

// UsesKeyFiles returns a slice of registered components that use key
// files
func UsesKeyFiles() (cts []*Component) {
	for _, c := range registeredComponents {
		if c.UsesKeyfiles {
			cts = append(cts, c)
		}
	}
	return
}

// ParseComponent returns the component type by iterating over all
// the names registered by components and returning as soon as any value
// matches. The comparison is case-insensitive. nil is returned if the
// component does not match any known name.
func ParseComponent(component string) *Component {
	if component == "" {
		return nil
	}
	for _, ct := range registeredComponents {
		names := []string{ct.Name}
		if ct.DownloadInfix != "" {
			names = append(names, ct.DownloadInfix)
		}
		for _, m := range append(names, ct.Aliases...) {
			if strings.EqualFold(m, component) {
				return ct
			}
		}
	}
	return nil
}
