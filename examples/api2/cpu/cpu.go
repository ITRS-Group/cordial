package cpu

import (
	"time"

	"github.com/itrs-group/cordial/pkg/geneos/api"
)

type CPUPlugin struct {
	api.Plugin
	cpustats cpustat
}

var _ api.Sampler = (*CPUPlugin)(nil)

func init() {
	cpuPlugin := &CPUPlugin{}
	api.RegisterPlugin("cpu", cpuPlugin)
}

func (c *CPUPlugin) Open() error { return nil }

func (c *CPUPlugin) Interval() time.Duration {
	return c.Plugin.Interval
}

func (c *CPUPlugin) SetInterval(i time.Duration) {
	c.Plugin.Interval = i
}

func (c *CPUPlugin) Start() error { return nil }
func (c *CPUPlugin) Stop() error  { return nil }
func (c *CPUPlugin) Close() error { return nil }
func (c *CPUPlugin) Pause() error { return nil }
