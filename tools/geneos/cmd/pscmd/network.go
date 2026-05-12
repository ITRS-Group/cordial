package pscmd

import (
	"fmt"
	"net"
	"strings"

	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
	"github.com/rs/zerolog/log"
)

type psInstanceNetwork struct {
	psCommon
	FD         int    `json:"fd"`
	Protocol   string `json:"protocol"`
	LocalAddr  net.IP `json:"local_addr"`
	LocalPort  uint16 `json:"local_port"`
	RemoteAddr net.IP `json:"remote_addr"`
	RemotePort uint16 `json:"remote_port"`
	Status     string `json:"status"`
	TxQueue    int64  `json:"tx_queue"`
	RxQueue    int64  `json:"rx_queue"`
}

var netToolkitColumns = []string{
	"ID",
	"type",
	"name",
	"host",
	"pid",
	"fd",
	"protocol",
	"localaddr",
	"localport",
	"remoteaddr",
	"remoteport",
	"status",
	"txqueue",
	"rxqueue",
}

var netCSVColumns = []string{
	"Type",
	"Name",
	"Host",
	"PID",
	"FD",
	"Protocol",
	"Local Addr",
	"Local Port",
	"Remote Addr",
	"Remote Port",
	"Status",
	"TXQueue",
	"RXQueue",
}

var netCSVHeader = strings.Join(netCSVColumns, "\t")

func psNetworkJSON(i geneos.Instance, pid int) (conns []psInstanceNetwork, err error) {
	ct := i.Type()
	h := i.Host()
	name := i.Name()

	// pi, err := process.ProcessStatus[*process.ProcessInfo](h, pid)
	pi, err := process.GetProcessInfo(h, pid, false)
	if err != nil {
		return
	}

	conns = make([]psInstanceNetwork, 0, len(pi.OpenFiles))

	for _, fd := range pi.OpenFiles {
		if fd.Conn != nil {
			// socket
			c := fd.Conn
			if !(strings.HasPrefix(c.Protocol, "tcp") || strings.HasPrefix(c.Protocol, "udp")) {
				continue
			}
			conns = append(conns, psInstanceNetwork{
				psCommon: psCommon{
					Type: ct,
					Name: name,
					Host: h,
					PID:  pid,
				},
				FD:         fd.FD,
				Protocol:   c.Protocol,
				LocalAddr:  c.LocalAddr,
				LocalPort:  c.LocalPort,
				RemoteAddr: c.RemoteAddr,
				RemotePort: c.RemotePort,
				Status:     c.Status,
				TxQueue:    c.TxQueue,
				RxQueue:    c.RxQueue,
			})
		}
	}

	return
}

func psNetworkCSV(i geneos.Instance, pid int, resp *responses.General) (err error) {
	ct := i.Type()
	h := i.Host()
	name := i.Name()

	var row []string
	if psCmdToolkit {
		row = append(row, instance.IDString(i))
	}

	row = append(row,
		ct.String(),
		name,
		h.String(),
		fmt.Sprint(pid),
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
	)
	resp.Dataview.Table = append(resp.Dataview.Table, row)

	for _, fd := range process.OpenFiles(h, pid) {
		if fd.Conn != nil {
			// socket
			c := fd.Conn
			if !(strings.HasPrefix(c.Protocol, "tcp") || strings.HasPrefix(c.Protocol, "udp")) {
				continue
			}
			remAddr := "-"
			if !c.RemoteAddr.Equal(net.IPv4(0, 0, 0, 0)) {
				remAddr = fmt.Sprint(c.RemoteAddr)
			}
			remPort := "-"
			if c.RemotePort != 0 {
				remPort = fmt.Sprint(c.RemotePort)
			}

			var row []string
			if psCmdToolkit {
				row = append(row, instance.IDString(i)+" # "+fmt.Sprint(fd.FD))
			}

			row = append(row,
				ct.String(),
				name,
				h.String(),
				fmt.Sprint(pid),
				fmt.Sprint(fd.FD),
				c.Protocol,
				c.LocalAddr.String(),
				fmt.Sprint(c.LocalPort),
				remAddr,
				remPort,
				c.Status,
				fmt.Sprint(c.TxQueue),
				fmt.Sprint(c.RxQueue),
			)
			resp.Dataview.Table = append(resp.Dataview.Table, row)
		}
	}

	return
}

func psNetworkTable(i geneos.Instance, pid int, resp *responses.General) (err error) {
	ct := i.Type()
	h := i.Host()
	name := i.Name()

	pi, _, _, _, err := psInstanceCommon(i)
	if err != nil {
		return
	}
	for _, fd := range pi.OpenFiles {
		if fd.Conn == nil {
			continue
		}

		// socket
		c := fd.Conn
		if !(strings.HasPrefix(c.Protocol, "tcp") || strings.HasPrefix(c.Protocol, "udp")) {
			continue
		}
		remAddr := "-"
		if !c.RemoteAddr.Equal(net.IPv4(0, 0, 0, 0)) {
			remAddr = fmt.Sprint(c.RemoteAddr)
		}
		remPort := "-"
		if c.RemotePort != 0 {
			remPort = fmt.Sprint(c.RemotePort)
		}
		row := []string{
			ct.String(),
			name,
			h.String(),
			fmt.Sprint(pid),
			fmt.Sprint(fd.FD),
			c.Protocol,
			c.LocalAddr.String(),
			fmt.Sprint(c.LocalPort),
			remAddr,
			remPort,
			c.Status,
			fmt.Sprint(c.TxQueue),
			fmt.Sprint(c.RxQueue),
		}
		resp.Dataview.Table = append(resp.Dataview.Table, row)
	}

	if capi, ok, err := checkCA(h, ct, pi.Children); err == nil && ok {
		log.Debug().Msgf("pid %d has CA child process with pid %d", pi.PID, capi.PID)
		for _, fd := range capi.OpenFiles {
			if fd.Conn == nil {
				continue
			}
			c := fd.Conn
			if !(strings.HasPrefix(c.Protocol, "tcp") || strings.HasPrefix(c.Protocol, "udp")) {
				continue
			}
			remAddr := "-"
			if !c.RemoteAddr.Equal(net.IPv4(0, 0, 0, 0)) {
				remAddr = fmt.Sprint(c.RemoteAddr)
			}
			remPort := "-"
			if c.RemotePort != 0 {
				remPort = fmt.Sprint(c.RemotePort)
			}
			row := []string{
				ct.String() + "/ca",
				name,
				h.String(),
				fmt.Sprint(capi.PID),
				fmt.Sprint(fd.FD),
				c.Protocol,
				c.LocalAddr.String(),
				fmt.Sprint(c.LocalPort),
				remAddr,
				remPort,
				c.Status,
				fmt.Sprint(c.TxQueue),
				fmt.Sprint(c.RxQueue),
			}
			resp.Dataview.Table = append(resp.Dataview.Table, row)
		}
	}

	return
}
