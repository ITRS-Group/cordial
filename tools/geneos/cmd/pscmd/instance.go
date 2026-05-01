package pscmd

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

type psInstance struct {
	psCommon
	Ports     []int     `json:"ports"`
	User      string    `json:"user,omitempty"`
	Group     string    `json:"group,omitempty"`
	Starttime time.Time `json:"starttime,omitempty"`
	Version   string    `json:"version,omitempty"`
	Home      string    `json:"home,omitempty"`

	// Extra fields, when `--long` is used
	Extra *process.ProcessInfo `json:"extra,omitempty"`
}

var instanceToolkitColumns = []string{
	"ID",
	"type",
	"name",
	"host",
	"pid",
	"ports",
	"user",
	"group",
	"startTime",
	"version",
	"home",
}

var instanceCSVColumns = []string{
	"Type",
	"Name",
	"Host",
	"PID",
	"Ports",
	"User",
	"Group",
	"Starttime",
	"Version",
	"Home",
}

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
var instanceCSVHeader = strings.Join(instanceCSVColumns, "\t")
var instanceCSVLongHeader = strings.Join(append(instanceCSVColumns, instanceCSVExtraColumns...), "\t")

func psInstanceJSON2(i geneos.Instance, resp *responses.Response) (err error) {
	ct := i.Type()
	h := i.Host()
	name := i.Name()

	pi, base, actual, uptodate, ports, err := psInstanceCommon(i)
	if err != nil {
		return
	}

	// ensure empty slice marshals as [] instead of null
	if len(ports) == 0 {
		ports = make([]int, 0)
	}

	psData := psInstance{
		psCommon: psCommon{
			Type: ct,
			Name: name,
			Host: h,
			PID:  pi.PID,
		},
		Ports:     ports,
		User:      pi.Username,
		Group:     pi.Groupname,
		Starttime: pi.CreationTime,
		Version:   fmt.Sprintf("%s%s%s", base, uptodate, actual),
		Home:      i.Home(),
	}

	if psCmdLong {
		psData.Extra = &process.ProcessInfo{}
		process.ProcessStatus(h, pi.PID, psData.Extra)
	}

	resp.Value = psData
	return
}

func checkCA(h *geneos.Host, pid int) (pi *process.ProcessInfo, ok bool, err error) {
	pi, err = process.GetProcessInfo(h, pid, false)
	if err != nil {
		return
	}
	if len(pi.Cmdline) == 0 {
		err = fmt.Errorf("no cmdline for PID %d", pid)
		return
	}

	if path.Base(pi.Cmdline[0]) != "java" {
		return
	}

	for _, arg := range pi.Cmdline[1:] {
		if strings.Contains(arg, "collection-agent") {
			ok = true
			return
		}
	}

	return
}
