package api

import (
	"sync"
	"time"
)

type Plugin interface {
	Init()
	Interval() time.Duration
	SetInterval(time.Duration)
	Start() error
	Stop() error
	SampleNow() error
}

var plugins sync.Map

func RegisterPlugin(name string, plugin Plugin) {
	plugins.Store(name, plugin)
}

func FindPlugin(name string) (plugin Plugin) {
	p, ok := plugins.Load(name)
	if ok {
		plugin = p.(Plugin)
	}
	return
}

func Attach(c APIClient, name, entity, sampler, group string) {
	p := FindPlugin(name)
	if p == nil {
		return
	}
	p.Init()
}
