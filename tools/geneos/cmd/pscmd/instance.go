package pscmd

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

type psInstance struct {
	psCommon
	Ports     string    `json:"ports"`
	User      string    `json:"user,omitempty"`
	Group     string    `json:"group,omitempty"`
	Starttime time.Time `json:"starttime,omitempty"`
	Version   string    `json:"version,omitempty"`
	Home      string    `json:"home,omitempty"`

	// Extra fields, when `--long` is used
	Extra *instance.ProcessInfo `json:"extra,omitempty"`
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

var instanceCSVHeader = strings.Join(instanceCSVColumns, "\t")

func getInstanceData(i geneos.Instance) (data *psInstance, err error) {
	ct := i.Type()
	h := i.Host()
	name := i.Name()

	pi, base, actual, uptodate, err := psInstanceCommon(i)
	if err != nil {
		return
	}

	psData := &psInstance{
		psCommon: psCommon{
			Type: ct,
			Name: name,
			Host: h,
			PID:  pi.PID,
		},
		Ports:     pi.ListeningPorts,
		User:      pi.Username,
		Group:     pi.Groupname,
		Starttime: pi.StartTime,
		Version:   fmt.Sprintf("%s%s%s", base, uptodate, actual),
		Home:      i.Home(),
	}

	if psCmdLong {
		// psData.Extra, _ = process.ProcessStatus[*process.ProcessInfo](h, pi.PID)
		psData.Extra, _ = process.GetProcessInfo[*instance.ProcessInfo](h, pi.PID, process.FetchLazyFields())
	}

	return psData, nil
}

func checkCA(i geneos.Instance, children []int) (pi *instance.ProcessInfo, ok bool, err error) {
	ct := i.Type()
	h := i.Host()

	if !ct.IsA("netprobe") || len(children) == 0 {
		return
	}

	for _, pid := range children {
		pi, err = process.GetProcessInfo[*instance.ProcessInfo](h, pid, process.FetchLazyFields())
		if err != nil {
			continue
		}

		if len(pi.Cmdline) == 0 {
			err = fmt.Errorf("no cmdline for PID %d", pid)
			continue
		}

		if path.Base(pi.Cmdline[0]) != "java" {
			continue
		}

		for _, arg := range pi.Cmdline[1:] {
			if strings.Contains(arg, "collection-agent") {
				ok = true
				return
			}
		}
	}

	return
}
