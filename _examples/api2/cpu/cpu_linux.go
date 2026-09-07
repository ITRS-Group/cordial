//go:build linux

package cpu

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// cpustats must be exported, along with all it's fields, so that
// it can be output by methods in the plugins package.
type cpustats struct {
	Name        string `column:"cpuName,sort=+num"`
	Utilisation uint64 `column:"% utilisation,format=%.2f %%"`
	User        uint64 `column:"user,format=%.2f %%"`
	Nice        uint64 `column:"nice,format=%.2f %%"`
	System      uint64 `column:"system,format=%.2f %%"`
	Idle        uint64 `column:"idle,format=%.2f %%"`
	IOWait      uint64 `column:"iowait,format=%.2f %%"`
	IRQ         uint64 `column:"irq,format=%.2f %%"`
	SoftIRQ     uint64 `column:"softirq,format=%.2f %%"`
	Steal       uint64 `column:"steal,format=%.2f %%"`
	Guest       uint64 `column:"guest,format=%.2f %%"`
	GuestNice   uint64 `column:"guest_nice,format=%.2f %%"`
}

// one entry for each CPU row in /proc/stats
type cpustat struct {
	lastsample time.Time
	cpus       map[string]cpustats
}

// SampleNow is the entry point for the example CPU sampler
func (p *CPUPlugin) SampleNow() (err error) {
	// first time through, store initial stats, don't update table and wait for next call
	var stat cpustat
	laststats := p.cpustats
	if laststats.lastsample.IsZero() {
		// first time through, store initial stats, don't update table and wait for next call
		var stat cpustat
		err = parsestats(&stat)
		if err != nil {
			return
		}
		p.cpustats = stat
	} else {
		err = parsestats(&stat)
		if err != nil {
			return
		}
		p.cpustats = stat

		// interval := stat.lastsample.Sub(laststats.lastsample)
		// err = d.UpdateTableFromMapDelta(stat.cpus, laststats.cpus, interval)
		p.cpustats = stat
	}
	return nil
}

func parsestats(c *cpustat) (err error) {
	stats, err := os.Open("/proc/stat")
	if err != nil {
		return
	}
	c.lastsample = time.Now()
	c.cpus = make(map[string]cpustats)
	defer stats.Close()

	lines := bufio.NewScanner(stats)
	for lines.Scan() {
		line := lines.Text()
		if strings.HasPrefix(line, "cpu") {
			var cpu cpustats
			// line = strings.ReplaceAll(line, "  ", " ")
			fmt.Sscanln(line, cpu.Name, cpu.User, cpu.Nice, cpu.System, cpu.Idle, cpu.IOWait, cpu.IRQ, cpu.SoftIRQ, cpu.Steal, cpu.Guest, cpu.GuestNice)
			if cpu.Name == "cpu" {
				cpu.Name = "_Total"
			}
			cpu.Utilisation = cpu.User + cpu.Nice + cpu.System + cpu.IRQ + cpu.SoftIRQ + cpu.Steal + cpu.Guest + cpu.GuestNice
			c.cpus[cpu.Name] = cpu
		}
	}
	return
}
