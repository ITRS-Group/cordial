package pscmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

type psInstance struct {
	psCommon
	Ports     []int     `json:"ports,omitempty"`
	User      string    `json:"user,omitempty"`
	Group     string    `json:"group,omitempty"`
	Starttime time.Time `json:"starttime,omitempty"`
	Version   string    `json:"version,omitempty"`
	Home      string    `json:"home,omitempty"`

	// Extra fields, when `--long` is used
	Extra *process.ProcessStats `json:"extra,omitempty"`
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
		psData.Extra = &process.ProcessStats{}
		process.ProcessStatus(h, pi.PID, psData.Extra)
	}

	resp.Value = psData
	return
}
