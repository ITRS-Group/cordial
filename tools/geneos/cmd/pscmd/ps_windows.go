//go:build windows

package pscmd

import (
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
	return []string{}
}
