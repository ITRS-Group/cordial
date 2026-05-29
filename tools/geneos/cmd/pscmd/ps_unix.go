//go:build !windows

package pscmd

import (
	"fmt"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var instanceToolkitExtraColumns = []string{
	"state",
	"threads",
	"openfiles",
	"opensockets",
	"residentSetSize",
	"residentSetSizeAnon",
	"residentSetSizeMax",
	"totalUserTime",
	"totalKernelTime",
	"totalChildUserTime",
	"totalChildKernelTime",
}

var instanceCSVExtraColumns = []string{
	"State",
	"Threads",
	"Open Files",
	"Open Sockets",
	"RSS",
	"RSSAnon",
	"RSSMax",
	"TotalUserTime",
	"TotalKernelTime",
	"TotalChildUserTime",
	"TotalChildKernelTime",
}

var instanceCSVLongHeader = strings.Join(append(instanceCSVColumns, instanceCSVExtraColumns...), "\t")

func psInstanceLongColumns(pi *instance.ProcessInfo) []string {
	return []string{
		pi.State,
		fmt.Sprint(pi.Threads),
		fmt.Sprint(len(pi.OpenFiles)),
		fmt.Sprint(pi.OpenSockets),
		fmt.Sprintf("%.2f MiB", float64(pi.VmRSS)/(1024*1024)),
		fmt.Sprintf("%.2f MiB", float64(pi.RssAnon)/(1024*1024)),
		fmt.Sprintf("%.2f MiB", float64(pi.VmHWM)/(1024*1024)),
		fmt.Sprintf("%.2f s", pi.Utime.Seconds()),
		fmt.Sprintf("%.2f s", pi.Stime.Seconds()),
		fmt.Sprintf("%.2f s", pi.CUtime.Seconds()),
		fmt.Sprintf("%.2f s", pi.CStime.Seconds()),
	}
}
