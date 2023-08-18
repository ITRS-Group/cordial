package api

import (
	"sync"
	"time"
)

// Sampler is the method set for a Plugin instance
type Sampler interface {
	// actions
	Open() error
	Start() error
	Pause() error
	Close() error
	SampleNow() error

	// info
	Interval() time.Duration
	SetInterval(time.Duration)
}

var plugins sync.Map

func RegisterPlugin(name string, plugin Sampler) {
	plugins.Store(name, plugin)
}

func FindPlugin(name string) (plugin Sampler) {
	p, ok := plugins.Load(name)
	if ok {
		plugin = p.(Sampler)
	}
	return
}

// Plugin is an instance of a plugin
type Plugin struct {
	APIClient
	Sampler
	views map[string]Dataview

	Interval time.Duration
}

func NewSampler(c APIClient, name, entity, sampler string) (s *Plugin) {
	s = &Plugin{
		APIClient: c,
		Sampler:   FindPlugin(name),
		views:     make(map[string]Dataview),
	}

	return
}
