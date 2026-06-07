package geneos

import (
	"log/slog"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
)

// Instance interfaces contains the method set for an instance of a
// registered Component
type Instance interface {
	// Config returns the configuration for the instance.
	Config() *config.Config

	// SetConfig sets the configuration object for the instance.
	SetConfig(*config.Config)

	// Name returns the base instance name
	Name() string

	// Home returns the path to the instance directory
	Home() string

	// Type returns a Component type for the instance
	Type() *Component

	// Host returns the Host for the instance
	Host() *Host

	// Log returns a logger for the instance
	Log() *slog.Logger

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
	Add(template string, port uint16, noCerts bool) error
	Command(skipFileCheck bool) ([]string, []string, string, error)
	Reload() (err error)
	Rebuild(initial bool) error
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
		config.Global().Default(k, v)
	}
}
