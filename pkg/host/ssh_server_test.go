//go:build !windows

package host

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/sftp"
	"github.com/skeema/knownhosts"
	"golang.org/x/crypto/ssh"
)

// testSSHServer is an in-process SSH server, listening on an ephemeral
// port, that runs `exec` requests with the local shell and serves the
// local filesystem over SFTP, with no authentication.
type testSSHServer struct {
	t        *testing.T
	ln       net.Listener
	hostKey  ssh.Signer
	accepted atomic.Int32

	// behaviours for tests of timeouts
	muteHandshake atomic.Bool
	muteSFTP      bool

	mu    sync.Mutex
	conns []net.Conn
}

func newTestSSHServer(t *testing.T) *testSSHServer {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen on loopback: %v", err)
	}

	s := &testSSHServer{t: t, ln: ln, hostKey: signer}
	t.Cleanup(func() {
		ln.Close()
		s.closeConns()
	})
	return s
}

func (s *testSSHServer) start() {
	go func() {
		for {
			conn, err := s.ln.Accept()
			if err != nil {
				return
			}
			s.accepted.Add(1)
			s.mu.Lock()
			s.conns = append(s.conns, conn)
			s.mu.Unlock()
			go s.serve(conn)
		}
	}()
}

func (s *testSSHServer) closeConns() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.conns {
		c.Close()
	}
	s.conns = nil
}

func (s *testSSHServer) serve(conn net.Conn) {
	if s.muteHandshake.Load() {
		// never say anything, until the connection is closed
		buf := make([]byte, 1)
		conn.Read(buf)
		return
	}

	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(s.hostKey)
	sconn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		return
	}
	defer sconn.Close()
	go ssh.DiscardRequests(reqs)

	for nc := range chans {
		if nc.ChannelType() != "session" {
			nc.Reject(ssh.UnknownChannelType, "unsupported")
			continue
		}
		ch, requests, err := nc.Accept()
		if err != nil {
			return
		}
		go s.session(ch, requests)
	}
}

func (s *testSSHServer) session(ch ssh.Channel, requests <-chan *ssh.Request) {
	defer ch.Close()
	for req := range requests {
		switch req.Type {
		case "subsystem":
			var payload struct{ Name string }
			ssh.Unmarshal(req.Payload, &payload)
			if payload.Name != "sftp" {
				req.Reply(false, nil)
				continue
			}
			req.Reply(true, nil)
			if s.muteSFTP {
				// accepted, but never answers
				for range requests {
				}
				return
			}
			srv, err := sftp.NewServer(ch)
			if err != nil {
				return
			}
			srv.Serve()
			return

		case "exec":
			var payload struct{ Command string }
			ssh.Unmarshal(req.Payload, &payload)
			req.Reply(true, nil)

			cmd := exec.Command("/bin/sh", "-c", payload.Command)
			cmd.Stdout = ch
			cmd.Stderr = ch.Stderr()
			status := 0
			if err := cmd.Run(); err != nil {
				if ee, ok := errors.AsType[*exec.ExitError](err); ok {
					status = ee.ExitCode()
				} else {
					status = 127
				}
			}
			ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{uint32(status)}))
			return

		default:
			req.Reply(false, nil)
		}
	}
}

// remote sets up an isolated home with a known_hosts entry for the
// server, so that connections work and no real configuration is used,
// and returns an SSHRemote for it with a name unique to the test.
func (s *testSSHServer) remote(t *testing.T, options ...any) *SSHRemote {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SSH_AUTH_SOCK", "")
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	addr := s.ln.Addr().String()
	line := knownhosts.Line([]string{addr}, s.hostKey.PublicKey()) + "\n"
	if err := os.WriteFile(filepath.Join(home, ".ssh", "known_hosts"), []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}

	_, port, _ := net.SplitHostPort(addr)
	p, _ := strconv.Atoi(port)
	opts := slices.Concat([]any{Hostname("127.0.0.1"), Port(uint16(p)), Username("test")}, options)
	h := NewSSHRemote("test-"+t.Name(), opts...).(*SSHRemote)
	t.Cleanup(h.Close)
	return h
}

func TestSSHDialIsSharedByConcurrentCallers(t *testing.T) {
	srv := newTestSSHServer(t)
	srv.start()
	h := srv.remote(t)

	const callers = 16
	var wg sync.WaitGroup
	start := make(chan struct{})
	sshClients := make([]*ssh.Client, callers)
	sftpClients := make([]*sftp.Client, callers)
	errs := make([]error, callers)
	for i := range callers {
		wg.Go(func() {
			<-start
			if i%2 == 0 {
				sshClients[i], errs[i] = h.DialSSH()
				return
			}
			sftpClients[i], errs[i] = h.DialSFTP()
		})
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("caller %d: %v", i, err)
		}
	}
	if n := srv.accepted.Load(); n != 1 {
		t.Errorf("server accepted %d connections, want 1", n)
	}
	for i := 2; i < callers; i += 2 {
		if sshClients[i] != sshClients[0] {
			t.Errorf("caller %d got a different ssh client", i)
		}
	}
	for i := 3; i < callers; i += 2 {
		if sftpClients[i] != sftpClients[1] {
			t.Errorf("caller %d got a different sftp client", i)
		}
	}
	if err := h.LastError(); err != nil {
		t.Errorf("LastError() = %v after success", err)
	}
}

func TestSSHDeadConnectionIsReplaced(t *testing.T) {
	srv := newTestSSHServer(t)
	srv.start()
	h := srv.remote(t)

	first, err := h.DialSFTP()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = first.Stat("/"); err != nil {
		t.Fatal(err)
	}

	srv.closeConns()

	deadline := time.Now().Add(5 * time.Second)
	for {
		second, err := h.DialSFTP()
		if err == nil && second != first {
			if _, err = second.Stat("/"); err != nil {
				t.Fatalf("replacement client is not usable: %v", err)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("dead connection was not replaced (err = %v)", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestSSHHandshakeTimeout(t *testing.T) {
	srv := newTestSSHServer(t)
	srv.muteHandshake.Store(true)
	srv.start()
	h := srv.remote(t)

	old := sshHandshakeTimeout
	sshHandshakeTimeout = 300 * time.Millisecond
	t.Cleanup(func() { sshHandshakeTimeout = old })

	done := make(chan error, 1)
	go func() {
		_, err := h.DialSSH()
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("DialSSH() succeeded against a server that never speaks")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("DialSSH() did not time out")
	}
	if h.LastError() == nil {
		t.Error("LastError() is nil after a failed dial")
	}
}

func TestSSHSFTPTimeout(t *testing.T) {
	srv := newTestSSHServer(t)
	srv.muteSFTP = true
	srv.start()
	h := srv.remote(t)

	old := sshSFTPTimeout
	sshSFTPTimeout = 300 * time.Millisecond
	t.Cleanup(func() { sshSFTPTimeout = old })

	done := make(chan error, 1)
	go func() {
		_, err := h.DialSFTP()
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("DialSFTP() succeeded against a server that never answers")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("DialSFTP() did not time out")
	}
}

func TestSSHFailureBackoffAndRecovery(t *testing.T) {
	srv := newTestSSHServer(t)
	srv.muteHandshake.Store(true)
	srv.start()
	h := srv.remote(t)

	oldHandshake := sshHandshakeTimeout
	sshHandshakeTimeout = 200 * time.Millisecond
	oldRetry := sshRetryInterval
	t.Cleanup(func() {
		sshHandshakeTimeout = oldHandshake
		sshRetryInterval = oldRetry
	})

	if _, err := h.DialSSH(); err == nil {
		t.Fatal("DialSSH() succeeded with a server that never speaks")
	}
	if h.LastError() == nil {
		t.Fatal("LastError() = nil after a failure")
	}
	if ok, err := h.IsAvailable(); ok || err == nil || !strings.Contains(err.Error(), "ago") {
		t.Errorf("IsAvailable() = %v, %v within the retry interval, want the last error", ok, err)
	}
	if n := srv.accepted.Load(); n != 1 {
		t.Fatalf("server accepted %d connections, want 1", n)
	}

	// LastError only reports, it must not connect again, however old
	// the failure is
	sshRetryInterval = 0
	srv.muteHandshake.Store(false)
	if err := h.LastError(); err == nil {
		t.Error("LastError() = nil without a successful connection")
	}
	if n := srv.accepted.Load(); n != 1 {
		t.Errorf("LastError() connected, server accepted %d connections", n)
	}

	// and the failure is forgotten once it works
	if _, err := h.DialSSH(); err != nil {
		t.Fatalf("DialSSH() after recovery error = %v", err)
	}
	if err := h.LastError(); err != nil {
		t.Errorf("LastError() = %v after a successful connection", err)
	}
}

func TestSSHRemoteRunQuoting(t *testing.T) {
	srv := newTestSSHServer(t)
	srv.start()
	h := srv.remote(t)

	// none of these may be expanded or split by the remote shell
	args := []string{"a b", "$HOME", "`id`", "it's", "", `"dq"`, `\n`, "$(id)", "a;b", "*"}
	cmd := exec.Command("printf", append([]string{"[%s]"}, args...)...)
	out, err := h.Run(cmd)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	var want strings.Builder
	for _, a := range args {
		want.WriteString("[" + a + "]")
	}
	if string(out) != want.String() {
		t.Errorf("Run() output = %q, want %q", out, want.String())
	}

	// environment values are used, and quoted
	cmd = exec.Command("sh", "-c", `printf %s "$QV"`)
	cmd.Env = []string{"QV=a b $HOME 'q' `id`; echo bad"}
	out, err = h.Run(cmd)
	if err != nil {
		t.Fatalf("Run() with env error = %v", err)
	}
	if string(out) != "a b $HOME 'q' `id`; echo bad" {
		t.Errorf("env value = %q", out)
	}

	// the working directory is quoted too
	dir := filepath.Join(t.TempDir(), "d i'r")
	if err = os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("pwd")
	cmd.Dir = dir
	if out, err = h.Run(cmd); err != nil || strings.TrimSpace(string(out)) != dir {
		t.Errorf("Run() in dir = %q, %v, want %q", out, err, dir)
	}

	// an invalid variable name is an error, not an injection
	cmd = exec.Command("true")
	cmd.Env = []string{"A;touch x=1"}
	if _, err = h.Run(cmd); !errors.Is(err, ErrInvalidArgs) {
		t.Errorf("Run() with bad env name error = %v", err)
	}
}

func TestSSHRemoteStart(t *testing.T) {
	srv := newTestSSHServer(t)
	srv.start()
	h := srv.remote(t)

	dir := filepath.Join(t.TempDir(), "work dir")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	waitFor := func(file string) string {
		t.Helper()
		var b []byte
		for range 100 {
			var err error
			if b, err = os.ReadFile(file); err == nil && len(b) > 0 {
				return string(b)
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("%s was not written", file)
		return ""
	}

	// the argument and environment value are not expanded, and the
	// output of the process goes to the (relative to dir) errfile
	cmd := exec.Command("sh", "-c", `printf '%s %s' "$1" "$SV"`, "sh", "$HOME 'x'")
	cmd.Dir = dir
	cmd.Env = []string{"SV=env value"}
	pid, err := h.Start(cmd, ProcessErrfile("out.log"))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if pid <= 0 {
		t.Errorf("Start() pid = %d", pid)
	}
	if got := waitFor(filepath.Join(dir, "out.log")); got != "$HOME 'x' env value" {
		t.Errorf("output = %q", got)
	}

	// with no errfile the output is discarded rather than the command
	// failing on an empty redirection
	cmd = exec.Command("sh", "-c", "echo hi; touch started")
	cmd.Dir = dir
	if _, err = h.Start(cmd); err != nil {
		t.Fatalf("Start() with no errfile error = %v", err)
	}
	for range 100 {
		if _, err := os.Stat(filepath.Join(dir, "started")); err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Error("command with no errfile did not run")
}

func TestSSHRemoteSignal(t *testing.T) {
	srv := newTestSSHServer(t)
	srv.start()
	h := srv.remote(t)

	for _, pid := range []int{0, -1} {
		if err := h.Signal(pid, 0); !errors.Is(err, ErrInvalidArgs) {
			t.Errorf("Signal(%d) error = %v, want ErrInvalidArgs", pid, err)
		}
	}

	// the exit status differs between shells, the message is what counts
	if err := h.Signal(1<<22+12345, 0); !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("Signal() of missing process error = %v, want ErrProcessDone", err)
	}
	if os.Getuid() != 0 {
		if err := h.Signal(1, 0); !errors.Is(err, os.ErrPermission) {
			t.Errorf("Signal() without permission error = %v, want ErrPermission", err)
		}
	}
	if err := h.Signal(os.Getpid(), 0); err != nil {
		t.Errorf("Signal() of own process error = %v", err)
	}
}

func TestSSHRemoteFileOperations(t *testing.T) {
	srv := newTestSSHServer(t)
	srv.start()
	h := srv.remote(t)
	root := t.TempDir()

	t.Run("Mkdir keeps perm", func(t *testing.T) {
		dir := filepath.Join(root, "m")
		if err := h.Mkdir(dir, 0o770); err != nil {
			t.Fatal(err)
		}
		if fi, err := os.Stat(dir); err != nil || fi.Mode().Perm() != 0o770 {
			t.Errorf("mode = %v, %v, want 0770", fi, err)
		}
	})

	t.Run("MkdirAll keeps perm", func(t *testing.T) {
		if err := os.Mkdir(filepath.Join(root, "existing"), 0o755); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(root, "existing", "x", "y")
		if err := h.MkdirAll(dir, 0o770); err != nil {
			t.Fatal(err)
		}
		for p, want := range map[string]os.FileMode{
			filepath.Join(root, "existing"):      0o755,
			filepath.Join(root, "existing", "x"): 0o770,
			dir:                                  0o770,
		} {
			if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != want {
				t.Errorf("%s mode = %v, %v, want %v", p, fi, err, want)
			}
		}
		// again is fine, but not over a file
		if err := h.MkdirAll(dir, 0o770); err != nil {
			t.Errorf("MkdirAll() of existing error = %v", err)
		}
		file := filepath.Join(root, "file")
		os.WriteFile(file, nil, 0o644)
		if err := h.MkdirAll(filepath.Join(file, "sub"), 0o770); err == nil {
			t.Error("MkdirAll() below a file succeeded")
		}
	})

	t.Run("RemoveAll", func(t *testing.T) {
		outside := filepath.Join(root, "outside")
		os.Mkdir(outside, 0o755)
		os.WriteFile(filepath.Join(outside, "keep"), []byte("keep"), 0o644)

		tree := filepath.Join(root, "tree")
		os.MkdirAll(filepath.Join(tree, "a", "b"), 0o755)
		os.WriteFile(filepath.Join(tree, "a", "b", "f"), []byte("f"), 0o644)
		os.WriteFile(filepath.Join(tree, "g"), []byte("g"), 0o644)
		os.Symlink(outside, filepath.Join(tree, "link"))

		if err := h.RemoveAll(tree); err != nil {
			t.Fatalf("RemoveAll() error = %v", err)
		}
		if _, err := os.Stat(tree); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("tree still exists: %v", err)
		}
		if _, err := os.Stat(filepath.Join(outside, "keep")); err != nil {
			t.Errorf("RemoveAll followed a symlink: %v", err)
		}
		if err := h.RemoveAll(filepath.Join(root, "missing")); err != nil {
			t.Errorf("RemoveAll() of missing path error = %v", err)
		}
	})

	t.Run("write, read, rename", func(t *testing.T) {
		a, b := filepath.Join(root, "a.txt"), filepath.Join(root, "b.txt")
		if err := h.WriteFile(a, []byte("alpha"), 0o640); err != nil {
			t.Fatal(err)
		}
		if err := h.WriteFile(b, []byte("beta"), 0o640); err != nil {
			t.Fatal(err)
		}
		if got, err := h.ReadFile(a); err != nil || string(got) != "alpha" {
			t.Errorf("ReadFile() = %q, %v", got, err)
		}
		if err := h.Rename(a, b); err != nil {
			t.Fatalf("Rename() over existing error = %v", err)
		}
		if got, _ := os.ReadFile(b); string(got) != "alpha" {
			t.Errorf("renamed contents = %q", got)
		}
		if err := h.WriteFile(filepath.Join(root, "nodir", "x"), nil, 0o640); err == nil {
			t.Error("WriteFile() into a missing directory succeeded")
		}
		if w, err := h.Create(filepath.Join(root, "nodir", "x"), 0o640); err == nil || w != nil {
			t.Errorf("Create() = %v, %v, want nil and an error", w, err)
		}
	})
}

func TestSSHRemoteWalkDirMatchesLocal(t *testing.T) {
	srv := newTestSSHServer(t)
	srv.start()
	h := srv.remote(t)

	root := t.TempDir()
	for _, d := range []string{"a/b", "a/c", "d", "e"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"a/b/f1", "a/c/f2", "a/f3", "d/f4", "e/f5", "z"} {
		if err := os.WriteFile(filepath.Join(root, f), []byte(f), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name string
		fn   func(path string, d fs.DirEntry) error
	}{
		{"all", func(string, fs.DirEntry) error { return nil }},
		{"skip dir a/b", func(p string, d fs.DirEntry) error {
			if p == "a/b" {
				return fs.SkipDir
			}
			return nil
		}},
		{"skip dir d", func(p string, d fs.DirEntry) error {
			if p == "d" {
				return fs.SkipDir
			}
			return nil
		}},
		{"skip all at a/c", func(p string, d fs.DirEntry) error {
			if p == "a/c" {
				return fs.SkipAll
			}
			return nil
		}},
		{"skip dir from a file skips the rest of its directory", func(p string, d fs.DirEntry) error {
			if p == "a/c/f2" {
				return fs.SkipDir
			}
			return nil
		}},
		{"error", func(p string, d fs.DirEntry) error {
			if p == "d/f4" {
				return os.ErrDeadlineExceeded
			}
			return nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walk := func(h Host) (visited []string, err error) {
				err = h.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					visited = append(visited, p+":"+strconv.FormatBool(d.IsDir()))
					return tt.fn(p, d)
				})
				return
			}
			want, wantErr := walk(NewLocal())
			got, gotErr := walk(h)
			if !slices.Equal(got, want) {
				t.Errorf("visited = %v, want %v", got, want)
			}
			if !errors.Is(gotErr, wantErr) && gotErr != wantErr {
				t.Errorf("error = %v, want %v", gotErr, wantErr)
			}
		})
	}

	t.Run("missing", func(t *testing.T) {
		var gotErr error
		h.WalkDir(filepath.Join(root, "missing"), func(p string, d fs.DirEntry, err error) error {
			gotErr = err
			return err
		})
		if !errors.Is(gotErr, fs.ErrNotExist) {
			t.Errorf("walk of missing dir error = %v, want ErrNotExist", gotErr)
		}
	})
}

// wrappedHost embeds a Host, as the geneos package does, so is neither
// a *Local nor a *SSHRemote
type wrappedHost struct{ Host }

func TestCopyAllFromWrappedRemote(t *testing.T) {
	srv := newTestSSHServer(t)
	srv.start()
	h := srv.remote(t)

	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "top.txt"), []byte("top"), 0o644)
	os.WriteFile(filepath.Join(src, "sub", "nested.txt"), []byte("nested"), 0o640)
	os.Symlink("top.txt", filepath.Join(src, "link"))

	if err := CopyAll(wrappedHost{h}, src, NewLocal(), dst); err != nil {
		t.Fatalf("CopyAll() error = %v", err)
	}
	for file, want := range map[string]string{"top.txt": "top", "sub/nested.txt": "nested"} {
		if got, err := os.ReadFile(filepath.Join(dst, file)); err != nil || string(got) != want {
			t.Errorf("%s = %q, %v, want %q", file, got, err, want)
		}
	}
	if link, err := os.Readlink(filepath.Join(dst, "link")); err != nil || link != "top.txt" {
		t.Errorf("link = %q, %v", link, err)
	}
}

func TestSSHRemoteGetFsIsNeverNil(t *testing.T) {
	// a remote that cannot be reached, as there is no known_hosts entry
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SSH_AUTH_SOCK", "")
	h := NewSSHRemote("unreachable-"+t.Name(), Hostname("127.0.0.1"), Port(1))
	fsys := h.GetFs()
	if fsys == nil {
		t.Fatal("GetFs() = nil")
	}
	if _, err := fsys.Stat("/"); err == nil {
		t.Error("Stat() on an unreachable host succeeded")
	}
	if _, err := fsys.Open("/"); err == nil {
		t.Error("Open() on an unreachable host succeeded")
	}
}
