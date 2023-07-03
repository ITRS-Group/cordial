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
	"path/filepath"
	"strings"

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
	Name:             RootComponentName,
	RelatedTypes:     nil,
	ComponentMatches: []string{"any"},
	RealComponent:    false,
	DownloadBase:     DownloadBases{Resources: "", Nexus: ""},
	GlobalSettings: map[string]string{
		// Root directory for all operations
		execname: "",

		// Root URL for all downloads of software archives
		config.Join("download", "url"): "https://resources.itrsgroup.com/download/latest/",

		// Path List separated additions to the reserved names list, over and above
		// any words matched by FindComponent()
		"reservednames": "",

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

// Component defines a register component
type Component struct {
	Initialise       func(*Host, *Component)
	New              func(string) Instance
	Name             string
	LegacyPrefix     string
	RelatedTypes     []*Component
	ComponentMatches []string
	ParentType       *Component
	RealComponent    bool
	UsesKeyfiles     bool
	Templates        []Templates
	DownloadBase     DownloadBases
	DownloadInfix    string // if set replace this string with component name for download matches
	PortRange        string
	CleanList        string
	PurgeList        string
	Aliases          map[string]string
	Defaults         []string // ordered list of key=value pairs
	GlobalSettings   map[string]string
	Directories      []string
	GetPID           func(interface{}) (int, error) // if set, use this to get the PID of an instance
}

// Instance interfaces contains the method set for an instance of a
// registered Component
type Instance interface {
	Config() *config.Config

	// getters and setters
	Name() string
	Home() string
	Type() *Component
	Host() *Host
	String() string

	// config
	Load() error
	Unload() error
	Loaded() bool

	// actions
	Add(template string, port uint16) error
	Command() ([]string, []string, string)
	Reload(params []string) (err error)
	Rebuild(bool) error
}

// RegisterComponent adds the given Component ct to the internal list of
// component types. The factory parameter is the Component's New()
// function, which has to be passed in to avoid initialisation loops as
// the function refers to the type being registered.
func (ct *Component) RegisterComponent(factory func(string) Instance) {
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

func (ct Component) String() (name string) {
	return ct.Name
}

// IsA returns true is any of the names match the any of the names
// defined in ComponentMatches.
func (ct Component) IsA(name ...string) bool {
	for _, a := range ct.ComponentMatches {
		for _, b := range name {
			if strings.EqualFold(a, b) {
				return true
			}
		}
	}
	return false
}

// MakeComponentDirs creates the directory structure for the component.
// If ct is nil then the Root component type is used. If there is an
// error creating the directory then the error is immediately returned
// and the list of directories may only be partially created.
func (ct *Component) MakeComponentDirs(h *Host) (err error) {
	name := RootComponentName
	if h == ALL {
		log.Fatal().Msg("called with all hosts")
	}
	if ct != nil {
		name = ct.Name
	}
	geneos := h.GetString(execname) // root for host h
	for _, d := range initDirs[name] {
		dir := filepath.Join(geneos, d)
		log.Debug().Msgf("mkdirall %s", dir)
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
	dir = h.Filepath(ct, ct.String()+"s")
	return
}

// InstancesDirs (plural) returns a list of possible instance
// directories to look for an instance.
func (ct *Component) InstancesDirs(h *Host) (dirs []string) {
	if ct == nil {
		return
	}
	// this check is to support older installations
	if ct.ParentType != nil {
		dirs = append(dirs, h.Filepath(ct.ParentType, ct.String()+"s"))
	}
	dirs = append(dirs, h.Filepath(ct.Name, ct.String()+"s"))
	return
}

// SharedPath return the shared directory for the component on host h
// joined to subdirectories and file given as subs.
func (ct *Component) SharedPath(h *Host, subs ...interface{}) string {
	if ct == nil {
		return ""
	}
	parts := []interface{}{}
	if ct.ParentType == nil {
		parts = append(parts, ct, ct.String()+sharedSuffix, subs)
	} else {
		parts = append(parts, ct.ParentType, ct.String()+sharedSuffix, subs)
	}
	return h.Filepath(parts...)
}

// OrList will return receiver, if not nil, or the list of component types
// passed as args. If no arguments are passed then all 'real' components (those
// with the `RealComponent` field set to true) are returned.
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

// RealComponents returns a slice of all registered components that have
// their `RealComponent` field set to true.
func RealComponents() (cts []*Component) {
	for _, c := range registeredComponents {
		if c.RealComponent {
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

// FindComponent returns the component type by iterating over all
// the names registered by components and returning as soon as any value
// matches. The comparison is case-insensitive. nil is returned if the
// component does not match any known name.
func FindComponent(component string) *Component {
	for _, v := range registeredComponents {
		for _, m := range v.ComponentMatches {
			if strings.EqualFold(m, component) {
				return v
			}
		}
	}
	return nil
}
