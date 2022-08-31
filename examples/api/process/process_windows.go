//go:build windows
// +build windows

package process

import (
	"log"
	"time"

	"github.com/StackExchange/wmi"
	"github.com/itrs-group/cordial/pkg/samplers"
)

type Win32_Process struct {
	ProcessID    uint32 `column:"PID,sort=+num"`
	Name         string `column:"processName`
	CommandLine  string `column:"commandLine"`
	CreationDate time.Time
}

func (p *ProcessSampler) DoSample() (err error) {
	var dst []Win32_Process
	q := wmi.CreateQuery(&dst, "")
	err = wmi.Query(q, &dst)
	if err != nil {
		log.Fatal(err)
	}

	data := make(map[string]Win32_Process, len(dst))
	for _, d := range dst {
		data[d.Name] = d
	}

	return p.UpdateTableFromMap(data)
}

func (p ProcessSampler) initColumns() (cols samplers.Columns, columnnames []string, sortcol string, err error) {
	return p.ColumnInfo(Win32_Process{})
}
