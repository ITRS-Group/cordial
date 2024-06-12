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

package geneos

import (
	"path"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/rs/zerolog/log"
)

var execname string

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
	DownloadBase: DownloadBases{Resources: "", Nexus: ""},
	GlobalSettings: map[string]string{
		// Root directory for all operations
		execname: "",

		// Root URL for all downloads of software archives
		config.Join("download", "url"): "https://resources.itrsgroup.com/download/latest/",

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
	Resources string
	Nexus     string
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

	DownloadBase  DownloadBases
	DownloadInfix string // if set replace this string with component name for download matches

	GlobalSettings map[string]string

	PortRange string
	CleanList string
	PurgeList string

	// Functions

	// Initialise a component. Callback after the component is registered
	Initialise func(*Host, *Component)

	// New is the factory method for the component. It has to be added
	// during initialisation to avoid loops
	New func(string) Instance

	// GetPID returns the process ID of an instance - if nil a standard
	// function is used
	GetPID func(string, interface{}, string, [][]byte) bool // if set, use this to get the PID of an instance
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
	Command() ([]string, []string, string)
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
	geneos := h.GetString(execname) // root for host h
	for _, d := range initDirs[name] {
		dir := path.Join(geneos, d)
		log.Debug().Msgf("%s: mkdirall %s", h, dir)
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

// OrList will return the method receiver, if not nil, or the list of
// component types passed as args. If no arguments are passed then all
// 'real' components (those with the `RealComponent` field set to true)
// are returned.
func (ct *Component) OrList(cts ...*Component) []*Component {
	if ct != nil {
		return []*Component{ct}
	}

	if len(cts) == 0 {
		return RealComponents()
	}

	return cts
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
// not the root
func RealComponents() (cts []*Component) {
	for _, c := range registeredComponents {
		if c != &RootComponent {
			cts = append(cts, c)
		}
	}
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
	for _, ct := range registeredComponents {
		for _, m := range append([]string{ct.Name}, ct.Aliases...) {
			if strings.EqualFold(m, component) {
				return ct
			}
		}
	}
	return nil
}
