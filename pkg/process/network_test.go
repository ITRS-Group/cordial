//go:build !windows

package process

import (
	"net"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/host"
)

// samples as written by the kernel, with the header
const procNetTCP = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 11111 1 0000000000000000 100 0 0 10 0
   1: 0100007F:E2A4 0100007F:1F90 01 00000010:00000020 00:00000000 00000000  1000        0 22222 1 0000000000000000 20 4 30 10 -1
   2: 0100007F:E2A4 0100007F
   3: 0100007F:E2A4 0100007F:1F90 01 00000000:00000000 00:00000000 00000000  1000        0
   4: 0100007F:E2A4 0100007F:1F90 01 00000000:00000000 00:00000000 00000000  1000        0 33333
   5: ZZZZ7F:E2A4 0100007F:1F90 01 00000000:00000000 00:00000000 00000000  1000        0 44444 1 0
   6: 00000000:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 55555 1 0000000000000000 100 0 0 10 0
`

const procNetTCP6 = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000001000000:1F91 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 66666 1 0000000000000000 100 0 0 10 0
   1: 00000000000000000000000000000000:0016 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 77777 1 0000000000000000 100 0 0 10 0
   2: 0000000000000000FFFF00000100007F:0050 0000000000000000FFFF00000200007F:C350 01 00000000:00000000 00:00000000 00000000  1000        0 88888 1 0000000000000000 20 4 30 10 -1
   3: B80D0120000000000000000001000000:01BB 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 99999 1 0000000000000000 100 0 0 10 0
`

func TestDecodeProcNetAddr(t *testing.T) {
	tests := []struct {
		in       string
		wantIP   string
		wantPort uint16
		wantErr  bool
	}{
		{in: "0100007F:1F90", wantIP: "127.0.0.1", wantPort: 8080},
		{in: "00000000:0000", wantIP: "0.0.0.0", wantPort: 0},
		{in: "0F02000A:0016", wantIP: "10.0.2.15", wantPort: 22},
		{in: "00000000000000000000000001000000:1F91", wantIP: "::1", wantPort: 8081},
		{in: "00000000000000000000000000000000:0016", wantIP: "::", wantPort: 22},
		{in: "0000000000000000FFFF00000100007F:0050", wantIP: "127.0.0.1", wantPort: 80}, // IPv4-mapped
		{in: "B80D0120000000000000000001000000:01BB", wantIP: "2001:db8::1", wantPort: 443},
		{in: "0100007F", wantErr: true},
		{in: "0100007F:ZZ", wantErr: true},
		{in: "0100007F:10000", wantErr: true},
		{in: "01007F:0050", wantErr: true},
		{in: "XX00007F:0050", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		ip, port, err := decodeProcNetAddr(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("decodeProcNetAddr(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if tt.wantErr {
			continue
		}
		if want := net.ParseIP(tt.wantIP); !ip.Equal(want) || port != tt.wantPort {
			t.Errorf("decodeProcNetAddr(%q) = %v, %d, want %v, %d", tt.in, ip, port, want, tt.wantPort)
		}
	}
}

func TestParseProcNet(t *testing.T) {
	entries, err := parseProcNet(strings.NewReader(procNetTCP), "tcp")
	if err != nil {
		t.Fatal(err)
	}

	// the short, truncated and garbled rows (2, 3 and 5) are skipped
	// without preventing later rows from being read
	var inodes []string
	for _, e := range entries {
		inodes = append(inodes, e.Inode)
	}
	if want := []string{"11111", "22222", "33333", "55555"}; !slices.Equal(inodes, want) {
		t.Fatalf("inodes = %v, want %v", inodes, want)
	}

	listen, est := entries[0], entries[1]
	if listen.State != TCP_LISTEN || listen.Status != "LISTEN" || listen.Protocol != "tcp" ||
		!listen.LocalAddr.Equal(net.ParseIP("127.0.0.1")) || listen.LocalPort != 8080 || listen.RemotePort != 0 {
		t.Errorf("listen entry = %+v", listen)
	}
	if est.State != TCP_ESTABLISHED || est.Status != "ESTABLISHED" || est.LocalPort != 0xE2A4 || est.RemotePort != 8080 ||
		est.TxQueue != 0x10 || est.RxQueue != 0x20 {
		t.Errorf("established entry = %+v", est)
	}

	entries6, err := parseProcNet(strings.NewReader(procNetTCP6), "tcp6")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries6) != 4 {
		t.Fatalf("got %d tcp6 entries, want 4", len(entries6))
	}
	want := []struct {
		local, remote string
		state         int
	}{
		{"::1", "::", TCP_LISTEN},
		{"::", "::", TCP_LISTEN},
		{"127.0.0.1", "127.0.0.2", TCP_ESTABLISHED},
		{"2001:db8::1", "::", TCP_LISTEN},
	}
	for i, w := range want {
		e := entries6[i]
		if !e.LocalAddr.Equal(net.ParseIP(w.local)) || !e.RemoteAddr.Equal(net.ParseIP(w.remote)) || e.State != w.state {
			t.Errorf("tcp6 entry %d = %v -> %v state %d, want %s -> %s state %d", i, e.LocalAddr, e.RemoteAddr, e.State, w.local, w.remote, w.state)
		}
	}

	// the header alone, and nothing at all
	for _, in := range []string{"  sl  local_address\n", ""} {
		if entries, err := parseProcNet(strings.NewReader(in), "tcp"); err != nil || len(entries) != 0 {
			t.Errorf("parseProcNet(%q) = %v, %v", in, entries, err)
		}
	}
}

func fakeNetHost() *fakeHost {
	h := newFakeHost()
	h.files["/proc/net/tcp"] = procNetTCP
	h.files["/proc/net/tcp6"] = procNetTCP6
	return h
}

func TestAllTCPListenPorts(t *testing.T) {
	h := fakeNetHost()
	ports := map[int]int{}
	if err := AllTCPListenPorts(h, ports); err != nil {
		t.Fatal(err)
	}

	// listeners after connections, in both files, are all found
	want := map[int]int{11111: 8080, 55555: 80, 66666: 8081, 77777: 22, 99999: 443}
	if len(ports) != len(want) {
		t.Errorf("ports = %v, want %v", ports, want)
	}
	for inode, port := range want {
		if ports[inode] != port {
			t.Errorf("port for inode %d = %d, want %d", inode, ports[inode], port)
		}
	}

	if opened, closed := h.totalOpens(), h.closed(); opened != 2 || closed != opened {
		t.Errorf("opened %d files and closed %d, want 2 and all closed", opened, closed)
	}

	// without IPv6 there is no tcp6, which is not a problem
	delete(h.files, "/proc/net/tcp6")
	ports = map[int]int{}
	if err := AllTCPListenPorts(h, ports); err != nil || ports[11111] != 8080 {
		t.Errorf("AllTCPListenPorts() without tcp6 = %v, %v", ports, err)
	}

	// but not being able to read any is
	delete(h.files, "/proc/net/tcp")
	if err := AllTCPListenPorts(h, map[int]int{}); err == nil {
		t.Error("AllTCPListenPorts() with no files succeeded")
	}
}

func TestSocketToConn(t *testing.T) {
	h := fakeNetHost()

	sc, err := SocketToConn(h, "socket:[22222]")
	if err != nil || sc == nil {
		t.Fatalf("SocketToConn() = %v, %v", sc, err)
	}
	if sc.Protocol != "tcp" || sc.Status != "ESTABLISHED" || sc.LocalPort != 0xE2A4 || sc.RemotePort != 8080 || sc.TxQueue != 0x10 {
		t.Errorf("connection = %+v", sc)
	}

	sc, err = SocketToConn(h, "socket:[88888]")
	if err != nil || sc == nil || sc.Protocol != "tcp6" || !sc.LocalAddr.Equal(net.ParseIP("127.0.0.1")) || sc.RemotePort != 50000 {
		t.Errorf("tcp6 connection = %+v, %v", sc, err)
	}

	// not found, and not sockets, are nil not empty
	for _, link := range []string{"socket:[1]", "pipe:[22222]", "/etc/passwd", "socket:[]", ""} {
		if sc, err := SocketToConn(h, link); sc != nil || err != nil {
			t.Errorf("SocketToConn(%q) = %v, %v, want nil, nil", link, sc, err)
		}
	}

	if opened, closed := h.totalOpens(), h.closed(); closed != opened {
		t.Errorf("opened %d files and closed only %d", opened, closed)
	}
}

func TestOpenFilesReadsConnectionsOnce(t *testing.T) {
	h := fakeNetHost()
	h.lists["/proc/42/fd"] = []string{"10", "3", "4", "5", "6", "0"}
	for fd, target := range map[string]string{
		"0":  "/dev/null",
		"3":  "socket:[11111]",
		"4":  "socket:[22222]",
		"5":  "pipe:[1234]",
		"6":  "anon_inode:[eventpoll]",
		"10": "socket:[99]", // not a TCP socket
	} {
		h.links["/proc/42/fd/"+fd] = target
	}

	files := OpenFiles(h, 42)
	if len(files) != 6 {
		t.Fatalf("OpenFiles() returned %d files, want 6: %+v", len(files), files)
	}

	var fds []int
	for _, f := range files {
		fds = append(fds, f.FD)
		if f.Lstat == nil || f.Stat == nil {
			t.Errorf("fd %d (%s) has Lstat %v and Stat %v, want both", f.FD, f.Path, f.Lstat, f.Stat)
		}
		wantConn := f.FD == 3 || f.FD == 4
		if (f.Conn != nil) != wantConn {
			t.Errorf("fd %d (%s) Conn = %+v", f.FD, f.Path, f.Conn)
		}
	}
	if !slices.IsSorted(fds) {
		t.Errorf("file descriptors = %v, want sorted numerically", fds)
	}
	if files[1].Conn == nil || files[1].Conn.Status != "LISTEN" || files[1].Conn.LocalPort != 8080 {
		t.Errorf("fd 3 connection = %+v", files[1].Conn)
	}

	// one read of each file, whatever the number of sockets
	if n4, n6 := h.opened("/proc/net/tcp"), h.opened("/proc/net/tcp6"); n4 != 1 || n6 != 1 {
		t.Errorf("/proc/net/tcp opened %d times and tcp6 %d times, want once each", n4, n6)
	}
	if opened, closed := h.totalOpens(), h.closed(); closed != opened {
		t.Errorf("opened %d files and closed only %d", opened, closed)
	}

	// none at all if no sockets
	h2 := fakeNetHost()
	h2.lists["/proc/43/fd"] = []string{"0"}
	h2.links["/proc/43/fd/0"] = "/dev/null"
	if files := OpenFiles(h2, 43); len(files) != 1 || h2.totalOpens() != 0 {
		t.Errorf("OpenFiles() = %d files, with %d opens, want 1 and none", len(files), h2.totalOpens())
	}
}

func TestOpenFilesAndListeningPortsOnLocalhost(t *testing.T) {
	if _, err := os.Stat("/proc/net/tcp"); err != nil {
		t.Skip("no /proc/net/tcp")
	}

	for _, network := range []string{"tcp4", "tcp6"} {
		t.Run(network, func(t *testing.T) {
			addr := "127.0.0.1:0"
			if network == "tcp6" {
				addr = "[::1]:0"
			}
			ln, err := net.Listen(network, addr)
			if err != nil {
				t.Skipf("cannot listen on %s: %v", network, err)
			}
			defer ln.Close()
			tcpAddr := ln.Addr().(*net.TCPAddr)

			var found *SocketConnection
			for _, f := range OpenFiles(host.Localhost, os.Getpid()) {
				if f.Conn != nil && f.Conn.LocalPort == uint16(tcpAddr.Port) && f.Conn.Status == "LISTEN" {
					found = f.Conn
				}
			}
			if found == nil {
				t.Fatalf("listener on %s was not found in the open files", tcpAddr)
			}
			if !found.LocalAddr.Equal(tcpAddr.IP) {
				t.Errorf("local address = %v, want %v", found.LocalAddr, tcpAddr.IP)
			}
			if ports := ListeningPorts(host.Localhost, os.Getpid()); !slices.Contains(ports, tcpAddr.Port) {
				t.Errorf("ListeningPorts() = %v, want to contain %d", ports, tcpAddr.Port)
			}
		})
	}
}
