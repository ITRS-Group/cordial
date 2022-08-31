//go:build windows

package cpu

import (
	"log"
	"time"

	"github.com/StackExchange/wmi"
	"github.com/itrs-group/cordial/pkg/samplers"
)

// Win32_PerfRawData_PerfOS_Processor must be exported along with all it's
// fields so that methods in plugins package can output the results
type Win32_PerfRawData_PerfOS_Processor struct {
	Name                  string `column:"cpuName"`
	PercentUserTime       uint64 `column:"% User Time,format=%.2f %%"`
	PercentPrivilegedTime uint64 `column:"% Priv Time,format=%.2f %%"`
	PercentIdleTime       uint64 `column:"% Idle Time,format=%.2f %%"`
	PercentProcessorTime  uint64 `column:"% Proc Time,format=%.2f %%"`
	PercentInterruptTime  uint64 `column:"% Intr Time,format=%.2f %%"`
	PercentDPCTime        uint64 `column:"% DPC Time,format=%.2f %%"`
	Timestamp_PerfTime    uint64 `column:"OMIT"`
	Frequency_PerfTime    uint64 `column:"OMIT"`
}

// one entry for each CPU row in /proc/stats
type cpustat struct {
	cpus       map[string]Win32_PerfRawData_PerfOS_Processor
	lastsample float64
	frequency  float64
}

func (p *CPUSampler) DoSample() (err error) {
	DebugLogger.Print("called")
	laststats := p.cpustats
	if laststats.lastsample == 0 {
		// first time through, store initial stats, don't update table and wait for next call
		var stat cpustat
		err = getstats(&stat)
		if err != nil {
			return
		}
		p.cpustats = stat
	} else {
		// calculate diff and publish
		var stat cpustat

		err = getstats(&stat)
		if err != nil {
			return
		}
		interval := stat.lastsample - laststats.lastsample

		err = p.UpdateTableFromMapDelta(stat.cpus, laststats.cpus, time.Duration(interval)*10*time.Millisecond)
		p.cpustats = stat
	}

	return
}

func getstats(c *cpustat) (err error) {
	c.cpus = make(map[string]Win32_PerfRawData_PerfOS_Processor)

	var dst []Win32_PerfRawData_PerfOS_Processor
	q := wmi.CreateQuery(&dst, "")
	err = wmi.Query(q, &dst)
	if err != nil {
		log.Fatal(err)
	}

	c.lastsample = float64(dst[0].Timestamp_PerfTime)
	c.frequency = float64(dst[0].Frequency_PerfTime)

	for k, v := range dst {
		c.cpus[v.Name] = Win32_PerfRawData_PerfOS_Processor(dst[k])
	}
	return
}

func (p CPUSampler) initColumns() (cols samplers.Columns, columnnames []string, sortcol string, err error) {
	return p.ColumnInfo(Win32_PerfRawData_PerfOS_Processor{})
}
