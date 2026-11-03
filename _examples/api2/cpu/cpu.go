package cpu

import (
	"time"

	"github.com/itrs-group/cordial/pkg/geneos/api"
)

// CPUPlugin TODO: Describe its purpose and intended use.
type CPUPlugin struct {
	api.Plugin
	cpustats cpustat
}

var _ api.Sampler = (*CPUPlugin)(nil)

func init() {
	cpuPlugin := &CPUPlugin{}
	api.RegisterPlugin("cpu", cpuPlugin)
}

// Open TODO: Document its behavior, inputs, and effects.
func (c *CPUPlugin) Open() error { return nil }

// Interval TODO: Document its behavior, inputs, and effects.
func (c *CPUPlugin) Interval() time.Duration {
	return c.Plugin.Interval
}

// SetInterval TODO: Document its behavior, inputs, and effects.
func (c *CPUPlugin) SetInterval(i time.Duration) {
	c.Plugin.Interval = i
}

// Start TODO: Document its behavior, inputs, and effects.
func (c *CPUPlugin) Start() error { return nil }
// Stop TODO: Document its behavior, inputs, and effects.
func (c *CPUPlugin) Stop() error  { return nil }
// Close TODO: Document its behavior, inputs, and effects.
func (c *CPUPlugin) Close() error { return nil }
// Pause TODO: Document its behavior, inputs, and effects.
func (c *CPUPlugin) Pause() error { return nil }
