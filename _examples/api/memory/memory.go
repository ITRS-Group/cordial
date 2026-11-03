package memory

import (
	"fmt"
	"runtime"

	"github.com/itrs-group/cordial/pkg/geneos/plugins"
	"github.com/itrs-group/cordial/pkg/geneos/samplers"
)

// MemorySampler TODO: Describe its purpose and intended use.
type MemorySampler struct {
	samplers.Samplers
}

// New TODO: Document its behavior, inputs, and effects.
func New(p *plugins.Connection, name string, group string) (*MemorySampler, error) {
	m := new(MemorySampler)
	m.Plugins = m
	return m, m.New(p, name, group)
}

// InitSampler TODO: Document its behavior, inputs, and effects.
func (p *MemorySampler) InitSampler() (err error) {
	p.Headline("OS", runtime.GOOS)
	p.Headline("SampleInterval", fmt.Sprintf("%v", p.Interval()))
	return
}
