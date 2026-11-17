//go:build !windows

package control

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/pkg/process"
)

// TestMain lets the test binary run as the background server that the
// commands control, which is selected by the environment so that the
// arguments, which are recorded in the PID file, do not matter
func TestMain(m *testing.M) {
	if file := os.Getenv("CONTROL_TEST_SERVER"); file != "" {
		runServer(file)
		return
	}
	os.Exit(m.Run())
}

func loadConfig(file string) (*config.Config, error) {
	if file == "" {
		return nil, fmt.Errorf("no configuration file")
	}
	return config.Read("test", config.FilePath(file), config.Format("yaml"), config.MustExist())
}

func runServer(file string) {
	*logger.Logger = *slog.New(slog.DiscardHandler)
	cf, err := loadConfig(file)
	if err != nil {
		os.Exit(2)
	}
	endpoints := []ims.Endpoint{{Method: http.MethodGet, Path: "/who", Handler: func(w http.ResponseWriter, r *http.Request) {
		c, _, _ := ims.RequestContext(w, r)
		fmt.Fprintf(w, "%s %d", config.Get[string](c, "marker"), os.Getpid())
	}}}
	err = ims.ServeWith(cf, endpoints, ims.ServeOptions{
		PIDFile:    config.Get[string](cf, "server.pidfile"),
		ConfigFile: file,
		Load:       func() (*config.Config, error) { return loadConfig(file) },
	})
	if err != nil {
		os.Exit(1)
	}
}

type env struct {
	t       *testing.T
	dir     string
	config  string
	pidfile string
	addr    string
	confArg string
	pidArg  string // the value of the --pidfile flag, set by parsing
	usePID  string // the --pidfile argument to give to commands
	d       *Daemon
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{t: t, dir: t.TempDir()}
	e.config = filepath.Join(e.dir, "server.yaml")
	e.pidfile = filepath.Join(e.dir, "server.pid")

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	e.addr = l.Addr().String()
	l.Close()

	e.write("A", "tok")
	e.usePID = e.pidfile
	t.Setenv("CONTROL_TEST_SERVER", e.config)
	e.d = &Daemon{
		Name:        "test server",
		PIDFileName: "test.pid",
		PIDFile:     &e.pidArg,
		ConfigFile:  func() string { return e.confArg },
		Load:        loadConfig,
	}
	t.Cleanup(func() {
		// make sure nothing is left running
		if st, _ := process.StatusOfPIDFile(e.pidfile); st.Running {
			process.SignalProcess(st.Info.PID, process.SigKill)
			process.WaitStopped(e.pidfile, 5*time.Second)
		}
	})
	return e
}

// write writes the server configuration, which is what the server and
// the commands read
func (e *env) write(marker, token string) {
	e.t.Helper()
	body := fmt.Sprintf("server:\n  listen: %s\n  path: /api\n  pidfile: %s\n  authentication:\n    token: %q\nmarker: %s\n", e.addr, e.pidfile, token, marker)
	if err := os.WriteFile(e.config, []byte(body), 0o600); err != nil {
		e.t.Fatal(err)
	}
}

// run runs a command and returns its output and error
func (e *env) run(args ...string) (string, error) {
	e.t.Helper()
	root := &cobra.Command{Use: "test", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(e.d.Commands()...)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if e.usePID != "" {
		args = append(args, "--pidfile", e.usePID)
	}
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

// start starts the server in the background as a program does with -D
func (e *env) start() int {
	e.t.Helper()
	pid, err := Daemonize(e.pidfile)
	if err != nil {
		e.t.Fatal(err)
	}
	return pid
}

// who returns what the server reports, which is the marker from the
// configuration it is using and its PID
func (e *env) who() (marker string, pid int, err error) {
	req, _ := http.NewRequest(http.MethodGet, "http://"+e.addr+"/api/who", nil)
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("status %d", resp.StatusCode)
	}
	_, err = fmt.Sscanf(string(b), "%s %d", &marker, &pid)
	return
}

func (e *env) waitMarker(want string) int {
	e.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		m, pid, err := e.who()
		if err == nil && m == want {
			return pid
		}
		if time.Now().After(deadline) {
			e.t.Fatalf("server did not report %q: got %q, %v", want, m, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestStatusAndStopWhenNotRunning(t *testing.T) {
	e := newEnv(t)

	out, err := e.run("status")
	if err == nil || !strings.Contains(out, "not running") {
		t.Errorf("status: %q, %v", out, err)
	}
	if out, err = e.run("stop"); err != nil || !strings.Contains(out, "not running") {
		t.Errorf("stop of a server that is not running: %q, %v (should succeed)", out, err)
	}
	if _, err = e.run("reload"); err == nil || !strings.Contains(err.Error(), "not running") {
		t.Errorf("reload: %v", err)
	}
	if _, err = e.run("restart"); err == nil || !strings.Contains(err.Error(), "no record") {
		t.Errorf("restart with no record of how it was started: %v", err)
	}
}

func TestDaemonLifecycle(t *testing.T) {
	e := newEnv(t)

	pid := e.start()
	if m, p, err := e.who(); err != nil || m != "A" || p != pid {
		t.Fatalf("who = %q, %d, %v (daemon pid %d)", m, p, err, pid)
	}

	// status
	out, err := e.run("status")
	if err != nil || !strings.Contains(out, fmt.Sprintf("running (pid %d", pid)) || !strings.Contains(out, e.config) {
		t.Errorf("status: %q, %v", out, err)
	}

	// the PID file can also be found using the configuration file
	e.usePID, e.confArg = "", e.config
	if out, err := e.run("status"); err != nil || !strings.Contains(out, "running") {
		t.Errorf("status using the configuration to find the PID file: %q, %v", out, err)
	}
	e.usePID, e.confArg = e.pidfile, ""

	// a second one cannot be started with the same PID file
	if _, err := Daemonize(e.pidfile); err == nil {
		t.Error("a second daemon was started with the same PID file")
	}

	// reload picks up the new configuration, in the same process
	e.write("B", "tok")
	if out, err = e.run("reload"); err != nil || !strings.Contains(out, "reload requested") {
		t.Fatalf("reload: %q, %v", out, err)
	}
	if p := e.waitMarker("B"); p != pid {
		t.Errorf("reload changed the process: %d, was %d", p, pid)
	}

	// a configuration that is not usable is refused by the command, without
	// telling the server, which carries on
	e.write("C", "")
	if _, err = e.run("reload"); err == nil || !strings.Contains(err.Error(), "not reloading") {
		t.Errorf("reload with an unusable configuration: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if m, _, _ := e.who(); m != "B" {
		t.Errorf("server changed after an unusable configuration: %q", m)
	}

	// a configuration file can be given to check, but the server was
	// started with its own
	other := filepath.Join(e.dir, "other.yaml")
	os.WriteFile(other, []byte("server:\n  listen: x\n"), 0o600)
	e.confArg = other
	if _, err = e.run("reload"); err == nil {
		t.Error("reload checked the wrong configuration file")
	}
	e.confArg = ""

	// restart starts a new process, with the same arguments and so the
	// same configuration file, which is read again
	e.write("D", "tok")
	if out, err = e.run("restart"); err != nil || !strings.Contains(out, "started") {
		t.Fatalf("restart: %q, %v", out, err)
	}
	newPid := e.waitMarker("D")
	if newPid == pid {
		t.Errorf("restart did not start a new process")
	}
	if st, _ := process.StatusOfPIDFile(e.pidfile); !st.Running || st.Info.PID != newPid {
		t.Errorf("pid file after restart = %+v", st)
	}

	// stop
	if out, err = e.run("stop"); err != nil || !strings.Contains(out, "stopped") {
		t.Fatalf("stop: %q, %v", out, err)
	}
	if _, _, err := e.who(); err == nil {
		t.Error("server still answering after stop")
	}
	if _, err := os.Stat(e.pidfile); !os.IsNotExist(err) {
		t.Errorf("pid file left behind: %v", err)
	}
	if out, err = e.run("stop"); err != nil || !strings.Contains(out, "not running") {
		t.Errorf("second stop: %q, %v", out, err)
	}
}

// a server that was killed leaves its PID file, which is not running, but
// the details allow it to be restarted
func TestRestartAfterKill(t *testing.T) {
	e := newEnv(t)
	pid := e.start()
	process.SignalProcess(pid, process.SigKill)
	if !process.WaitStopped(e.pidfile, 5*time.Second) {
		t.Fatal("server did not die")
	}
	if out, err := e.run("status"); err == nil || !strings.Contains(out, "not running") {
		t.Errorf("status of a dead server: %q, %v", out, err)
	}

	if out, err := e.run("restart"); err != nil || !strings.Contains(out, "started") {
		t.Fatalf("restart: %q, %v", out, err)
	}
	if p := e.waitMarker("A"); p == pid {
		t.Error("the new server has the old pid")
	}
}

func TestPIDFileLocation(t *testing.T) {
	cf := config.New()
	if got := ConfiguredPIDFile(cf, ""); got != "" {
		t.Errorf("unset = %q", got)
	}
	config.Set(cf, "server.pidfile", "/from/config.pid")
	if got := ConfiguredPIDFile(cf, ""); got != "/from/config.pid" {
		t.Errorf("config = %q", got)
	}
	if got := ConfiguredPIDFile(cf, "/from/flag.pid"); got != "/from/flag.pid" {
		t.Errorf("flag = %q", got)
	}
	if got := ConfiguredPIDFile(nil, ""); got != "" {
		t.Errorf("nil config = %q", got)
	}
	if got := DefaultPIDFile("x.pid"); filepath.Base(got) != "x.pid" || filepath.Base(filepath.Dir(got)) != "geneos" {
		t.Errorf("default = %q", got)
	}

	// the commands use the flag, then the configuration, then the default
	flag := ""
	file := ""
	d := &Daemon{
		PIDFileName: "d.pid", PIDFile: &flag, ConfigFile: func() string { return file },
		Load: func(f string) (*config.Config, error) {
			if f == "" {
				return nil, fmt.Errorf("none")
			}
			c := config.New()
			config.Set(c, "server.pidfile", "/configured.pid")
			return c, nil
		},
	}
	if got := d.pidFile(); got != DefaultPIDFile("d.pid") {
		t.Errorf("default = %q", got)
	}
	file = "x"
	if got := d.pidFile(); got != "/configured.pid" {
		t.Errorf("configured = %q", got)
	}
	flag = "/flag.pid"
	if got := d.pidFile(); got != "/flag.pid" {
		t.Errorf("flag = %q", got)
	}
}
