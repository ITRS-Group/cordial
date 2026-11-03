//go:build windows

package cpu

import (
	"errors"
	"log/slog"

	"github.com/StackExchange/wmi"
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
	Timestamp_PerfTime    uint64 `column:"-"`
	Frequency_PerfTime    uint64 `column:"-"`
}

// one entry for each CPU row in /proc/stats
type cpustat struct {
	cpus       map[string]Win32_PerfRawData_PerfOS_Processor
	lastsample float64
	frequency  float64
}

var log = slog.Default()

// SampleNow stores the latest Windows processor counters for this sampler.
func (p *CPUPlugin) SampleNow() error {
	var stat cpustat
	if err := getstats(&stat); err != nil {
		return err
	}
	p.cpustats = stat
	return nil
}

func getstats(c *cpustat) (err error) {
	c.cpus = make(map[string]Win32_PerfRawData_PerfOS_Processor)

	var dst []Win32_PerfRawData_PerfOS_Processor
	q := wmi.CreateQuery(&dst, "")
	err = wmi.Query(q, &dst)
	if err != nil {
		log.Error("error querying WMI", slog.Any("error", err))
		return
	}
	if len(dst) == 0 || dst[0].Frequency_PerfTime == 0 {
		return errors.New("WMI returned no usable processor data")
	}

	c.lastsample = float64(dst[0].Timestamp_PerfTime)
	c.frequency = float64(dst[0].Frequency_PerfTime)

	for _, v := range dst {
		c.cpus[v.Name] = v
	}
	return
}
