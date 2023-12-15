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
