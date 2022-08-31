//go:build linux

package cpu

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/samplers"
)

// CPUStats must be exported, along with all it's fields, so that
// it can be output by methods in the plugins package.
type CPUStats struct {
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
	cpus       map[string]CPUStats
	lastsample time.Time
}

func (p *CPUSampler) DoSample() (err error) {
	logDebug.Print("called")
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
		// calculate diff and publish
		var stat cpustat

		err = parsestats(&stat)
		if err != nil {
			return
		}

		interval := stat.lastsample.Sub(laststats.lastsample)
		err = p.UpdateTableFromMapDelta(stat.cpus, laststats.cpus, interval)
		p.cpustats = stat
	}

	return
}

func parsestats(c *cpustat) (err error) {
	stats, err := os.Open("/proc/stat")
	if err != nil {
		return
	}
	c.lastsample = time.Now()
	c.cpus = make(map[string]CPUStats)
	defer stats.Close()

	lines := bufio.NewScanner(stats)
	for lines.Scan() {
		line := lines.Text()
		if strings.HasPrefix(line, "cpu") {
			line = strings.ReplaceAll(line, "  ", " ")
			fields := strings.Split(line, " ")
			var cpu CPUStats
			cpuname := fields[0]
			cpu.Name = cpuname
			if cpuname == "cpu" {
				cpu.Name = "_Total"
			}
			cpu.User, _ = strconv.ParseUint(fields[1], 10, 0)
			cpu.Nice, _ = strconv.ParseUint(fields[2], 10, 0)
			cpu.System, _ = strconv.ParseUint(fields[3], 10, 0)
			cpu.Idle, _ = strconv.ParseUint(fields[4], 10, 0)
			cpu.IOWait, _ = strconv.ParseUint(fields[5], 10, 0)
			cpu.IRQ, _ = strconv.ParseUint(fields[6], 10, 0)
			cpu.SoftIRQ, _ = strconv.ParseUint(fields[7], 10, 0)
			cpu.Steal, _ = strconv.ParseUint(fields[8], 10, 0)
			cpu.Guest, _ = strconv.ParseUint(fields[9], 10, 0)
			cpu.GuestNice, _ = strconv.ParseUint(fields[10], 10, 0)

			cpu.Utilisation = cpu.User + cpu.Nice + cpu.System + cpu.IRQ + cpu.SoftIRQ + cpu.Steal + cpu.Guest + cpu.GuestNice
			c.cpus[cpuname] = cpu
		}
	}
	return
}

func (p CPUSampler) initColumns() (cols samplers.Columns, columnnames []string, sortcol string, err error) {
	return p.ColumnInfo(CPUStats{})
}
