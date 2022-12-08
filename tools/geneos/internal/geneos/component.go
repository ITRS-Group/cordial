package geneos

import (
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"

	"github.com/rs/zerolog/log"
)

// definitions and access methods for the generic component types

// type ComponentType string

type DownloadBases struct {
	Resources string
	Nexus     string
}

type Component struct {
	Initialise       func(*host.Host, *Component)
	New              func(string) Instance
	Name             string
	RelatedTypes     []*Component
	ComponentMatches []string
	RealComponent    bool
	DownloadBase     DownloadBases
	PortRange        string
	CleanList        string
	PurgeList        string
	Aliases          map[string]string
	Defaults         []string // ordered list of key=value pairs
	GlobalSettings   map[string]string
	Directories      []string
}

type Instance interface {
	Config() *config.Config

	// getters and setters
	Name() string
	Home() string
	Type() *Component
	Host() *host.Host
	Prefix() string
	String() string

	// config
	Load() error
	Unload() error
	Loaded() bool
	SetConf(*config.Config)

	// actions
	Add(string, string, uint16) error
	Command() ([]string, []string)
	Reload(params []string) (err error)
	Rebuild(bool) error
}

var Root Component = Component{
	Name:             "none",
	RelatedTypes:     nil,
	ComponentMatches: []string{"any"},
	RealComponent:    false,
	DownloadBase:     DownloadBases{Resources: "", Nexus: ""},
	GlobalSettings: map[string]string{
		// Root directory for all operations
		"geneos": "",

		// Root URL for all downloads of software archives
		"download.url": "https://resources.itrsgroup.com/download/latest/",

		// Username to start components if not explicitly defined
		// and we are running with elevated privileges
		//
		// When running as a normal user this is unused and
		// we simply test a defined user against the running user
		//
		// default is owner of Geneos
		"defaultuser": "",

		// Path List seperated additions to the reserved names list, over and above
		// any words matched by ParseComponentName()
		"reservednames": "",

		"privatekeys": "id_rsa,id_ecdsa,id_ecdsa_sk,id_ed25519,id_ed25519_sk,id_dsa",
	},
	Directories: []string{
		"packages/downloads",
		"hosts",
	},
}

func init() {
	Root.RegisterComponent(nil)
}

type ComponentsMap map[string]*Component

// slice of registered component types for indirect calls
// this should actually become an Interface
var components ComponentsMap = make(ComponentsMap)

// AllComponents returns a slice of all registered components
func AllComponents() (cts []*Component) {
	for _, c := range components {
		cts = append(cts, c)
	}
	return
}

// currently supported real component types, for looping
// (go doesn't allow const slices, a function is the workaround)
func RealComponents() (cts []*Component) {
	for _, c := range components {
		if c.RealComponent {
			cts = append(cts, c)
		}
	}
	return
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
	components[ct.Name] = ct
	ct.RegisterDirs(ct.Directories)
	for k, v := range ct.GlobalSettings {
		config.GetConfig().SetDefault(k, v)
	}
}

var initDirs map[string][]string = make(map[string][]string)

// register directories that need to be created in the
// root of the install (by init)
func (ct Component) RegisterDirs(dirs []string) {
	initDirs[ct.Name] = dirs
}

func (ct Component) String() (name string) {
	return ct.Name
}

// ParseComponentName returns the component type by iterating over all
// the names registered by components and returning as soon as any value
// matches. The comparison is case-insensitive. nil is returned if the
// component does not match any known name.
func ParseComponentName(component string) *Component {
	for _, v := range components {
		for _, m := range v.ComponentMatches {
			if strings.EqualFold(m, component) {
				return v
			}
		}
	}
	return nil
}

func (ct *Component) MakeComponentDirs(h *host.Host) (err error) {
	name := "none"
	if h == host.ALL {
		log.Fatal().Msg("called with all hosts")
	}
	if ct != nil {
		name = ct.Name
	}
	geneos := h.GetString("geneos")
	uid, gid := -1, -1
	if utils.IsSuperuser() {
		uid, gid, _, _ = utils.GetIDs("")
	}

	for _, d := range initDirs[name] {
		dir := filepath.Join(geneos, d)
		log.Debug().Msgf("mkdirall %s", dir)
		if err = h.MkdirAll(dir, 0775); err != nil {
			return
		}
		if uid != -1 && gid != -1 {
			h.Chown(dir, uid, gid)
		}
	}
	return
}

// InstancesDir return the base directory for the instances of a
// component
func (ct *Component) InstancesDir(h *host.Host) string {
	if ct == nil {
		return ""
	}
	p := h.Filepath(ct, ct.String()+"s")
	return p
}

// SharedDir return the shared directory for the component
func (ct *Component) SharedDir(h *host.Host) string {
	if ct == nil {
		return ""
	}
	p := h.Filepath(ct, ct.String()+"_shared")
	return p
}

// Range will either return just the specific component it is called on,
// or if that is nil than the list of component types passed as args. If
// no arguments are passed then all real components types are returned.
//
// This is a convenience to avoid a double layer of if and range in
// callers than want to work on specific component types.
func (ct *Component) Range(cts ...*Component) []*Component {
	if ct != nil {
		return []*Component{ct}
	}

	if len(cts) == 0 {
		return RealComponents()
	}

	return cts
}
