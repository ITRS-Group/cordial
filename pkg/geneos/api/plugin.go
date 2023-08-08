package api

import (
	"sync"
	"time"
)

type Plugin struct {
	Name      string
	Interval  time.Duration
	Start     func()
	Stop      func()
	SampleNow func()
}

var plugins sync.Map

func RegisterPlugin(plugin Plugin) {
	plugins.Store(plugin.Name, plugin)
}
