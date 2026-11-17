//go:build !windows

package ims

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
)

// reloadTest runs a server, as a program would, and lets a test change
// the configuration it loads on reload
type reloadTest struct {
	t      *testing.T
	mu     sync.Mutex
	next   func() (*config.Config, error)
	addrs  chan net.Addr
	cancel context.CancelFunc
	done   chan struct{}
	result error
	logs   *syncBuffer

	checked, applied []string
	release          chan struct{}
	entered          chan string
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}
func (s *syncBuffer) String() string { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

func testServerConfig(marker, listen, token string) *config.Config {
	cf := config.New()
	config.Set(cf, "server.listen", listen)
	config.Set(cf, "server.path", "/api")
	config.Set(cf, "server.authentication.token", token)
	config.Set(cf, "marker", marker)
	return cf
}

func startReloadTest(t *testing.T, cf *config.Config, options func(*ServeOptions)) *reloadTest {
	t.Helper()
	rt := &reloadTest{
		t:       t,
		addrs:   make(chan net.Addr, 4),
		done:    make(chan struct{}),
		release: make(chan struct{}),
		entered: make(chan string, 4),
		logs:    &syncBuffer{},
	}

	orig := *log
	*log = *slog.New(slog.NewTextHandler(rt.logs, nil))
	t.Cleanup(func() { *log = orig })

	endpoints := []Endpoint{
		// who reports which configuration handled the request
		{Method: http.MethodGet, Path: "/who", Handler: func(w http.ResponseWriter, r *http.Request) {
			c, _, _ := RequestContext(w, r)
			_, _ = io.WriteString(w, config.Get[string](c, "marker"))
		}},
		// slow holds the request until the test releases it, and then
		// reports the configuration the request started with
		{Method: http.MethodGet, Path: "/slow", Handler: func(w http.ResponseWriter, r *http.Request) {
			c, _, _ := RequestContext(w, r)
			marker := config.Get[string](c, "marker")
			rt.entered <- marker
			<-rt.release
			_, _ = io.WriteString(w, marker)
		}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	rt.cancel = cancel
	opts := ServeOptions{
		Context: ctx,
		Ready:   func(addr net.Addr) { rt.addrs <- addr },
		Load: func() (*config.Config, error) {
			rt.mu.Lock()
			defer rt.mu.Unlock()
			if rt.next == nil {
				return nil, errors.New("no configuration")
			}
			return rt.next()
		},
		Check: func(cf *config.Config) error {
			rt.mu.Lock()
			rt.checked = append(rt.checked, config.Get[string](cf, "marker"))
			rt.mu.Unlock()
			if config.Get[bool](cf, "reject") {
				return errors.New("rejected by check")
			}
			return nil
		},
		Apply: func(cf *config.Config) {
			rt.mu.Lock()
			rt.applied = append(rt.applied, config.Get[string](cf, "marker"))
			rt.mu.Unlock()
		},
	}
	if options != nil {
		options(&opts)
	}

	go func() { rt.result = ServeWith(cf, endpoints, opts); close(rt.done) }()
	t.Cleanup(func() {
		select {
		case <-rt.release:
		default:
			close(rt.release)
		}
		cancel()
		select {
		case <-rt.done:
		case <-time.After(5 * time.Second):
			t.Error("server did not stop")
		}
	})
	return rt
}

func (rt *reloadTest) addr() string {
	rt.t.Helper()
	select {
	case a := <-rt.addrs:
		return a.String()
	case <-time.After(5 * time.Second):
		rt.t.Fatal("server did not start listening")
		return ""
	}
}

func (rt *reloadTest) setNext(f func() (*config.Config, error)) {
	rt.mu.Lock()
	rt.next = f
	rt.mu.Unlock()
}

func (rt *reloadTest) reload() {
	rt.t.Helper()
	if err := syscall.Kill(os.Getpid(), process.SigReload.(syscall.Signal)); err != nil {
		rt.t.Fatal(err)
	}
}

// waitLog waits for the server to log the text
func (rt *reloadTest) waitLog(text string) {
	rt.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(rt.logs.String(), text) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	rt.t.Fatalf("log does not contain %q:\n%s", text, rt.logs.String())
}

var plainClient = &http.Client{Timeout: 5 * time.Second}

func get(t *testing.T, client *http.Client, url, token string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestServeStopsAndHoldsPIDFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.pid")
	rt := startReloadTest(t, testServerConfig("A", "127.0.0.1:0", "tok"), func(o *ServeOptions) {
		o.PIDFile = path
		o.ConfigFile = "/etc/x.yaml"
	})
	addr := rt.addr()

	if code, body := get(t, plainClient, "http://"+addr+"/api/who", "tok"); code != 200 || body != "A" {
		t.Fatalf("get = %d %q", code, body)
	}

	// the PID file says how to restart, and is held
	st, err := process.StatusOfPIDFile(path)
	if err != nil || !st.Running || st.Info.PID != os.Getpid() || st.Info.ConfigFile != "/etc/x.yaml" || st.Info.Dir == "" || st.Info.Executable == "" {
		t.Errorf("status = %+v, %v", st, err)
	}
	// so a second instance cannot start
	err = ServeWith(testServerConfig("B", "127.0.0.1:0", "tok"), nil, ServeOptions{PIDFile: path})
	if _, ok := errors.AsType[*process.AlreadyRunningError](err); !ok {
		t.Errorf("second server error = %v, want AlreadyRunningError", err)
	}

	rt.cancel()
	select {
	case <-rt.done:
		if rt.result != nil {
			t.Errorf("ServeWith() = %v", rt.result)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
	if st, _ := process.StatusOfPIDFile(path); st.Exists || st.Running {
		t.Errorf("pid file after stop = %+v", st)
	}
	if _, err := plainClient.Get("http://" + addr + "/api/who"); err == nil {
		t.Error("server still accepting connections after it stopped")
	}
}

func TestServeConfigurationErrors(t *testing.T) {
	for name, cf := range map[string]*config.Config{
		"no listen": testServerConfig("A", "", "tok"),
		"no token":  testServerConfig("A", "127.0.0.1:0", ""),
	} {
		if err := ServeWith(cf, nil, ServeOptions{}); err == nil {
			t.Errorf("%s: ServeWith() succeeded", name)
		}
		if err := CheckServer(cf); err == nil {
			t.Errorf("%s: CheckServer() succeeded", name)
		}
	}
	if err := CheckServer(testServerConfig("A", "127.0.0.1:0", "tok")); err != nil {
		t.Errorf("CheckServer() = %v", err)
	}

	// failing to listen is an error, and does not leave the PID file
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	defer l.Close()
	path := filepath.Join(t.TempDir(), "p.pid")
	if err := ServeWith(testServerConfig("A", l.Addr().String(), "tok"), nil, ServeOptions{PIDFile: path}); err == nil {
		t.Error("ServeWith() succeeded on an address in use")
	}
	if st, _ := process.StatusOfPIDFile(path); st.Exists {
		t.Errorf("pid file left behind: %+v", st)
	}
}

func TestReloadUsesNewConfiguration(t *testing.T) {
	rt := startReloadTest(t, testServerConfig("A", "127.0.0.1:0", "tok"), nil)
	addr := rt.addr()
	url := "http://" + addr

	if _, body := get(t, plainClient, url+"/api/who", "tok"); body != "A" {
		t.Fatalf("before reload = %q", body)
	}

	// a new marker, token, and path
	rt.setNext(func() (*config.Config, error) {
		cf := testServerConfig("B", addr, "newtok")
		config.Set(cf, "server.path", "/v2")
		return cf, nil
	})
	rt.reload()
	rt.waitLog("configuration reloaded")

	if code, body := get(t, plainClient, url+"/v2/who", "newtok"); code != 200 || body != "B" {
		t.Errorf("new config: %d %q", code, body)
	}
	if code, _ := get(t, plainClient, url+"/v2/who", "tok"); code != http.StatusUnauthorized {
		t.Errorf("old token after reload: %d, want 401", code)
	}
	if code, _ := get(t, plainClient, url+"/api/who", "newtok"); code != http.StatusNotFound {
		t.Errorf("old path after reload: %d, want 404", code)
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()
	if len(rt.checked) != 1 || rt.checked[0] != "B" || len(rt.applied) != 1 || rt.applied[0] != "B" {
		t.Errorf("checked %v, applied %v", rt.checked, rt.applied)
	}
}

// a request that is being handled when the server reloads completes, using
// the configuration that it started with, while new requests use the new
// one
func TestReloadDoesNotAffectRequestsInProgress(t *testing.T) {
	rt := startReloadTest(t, testServerConfig("A", "127.0.0.1:0", "tok"), nil)
	addr := rt.addr()
	url := "http://" + addr

	type result struct {
		code int
		body string
	}
	slow := make(chan result, 1)
	go func() {
		code, body := get(t, plainClient, url+"/api/slow", "tok")
		slow <- result{code, body}
	}()
	if m := <-rt.entered; m != "A" {
		t.Fatalf("request started with %q", m)
	}

	rt.setNext(func() (*config.Config, error) { return testServerConfig("B", addr, "tok"), nil })
	rt.reload()
	rt.waitLog("configuration reloaded")

	// a new request gets the new configuration while the other waits
	if _, body := get(t, plainClient, url+"/api/who", "tok"); body != "B" {
		t.Errorf("new request = %q, want B", body)
	}
	select {
	case r := <-slow:
		t.Fatalf("request in progress finished early: %+v", r)
	default:
	}

	close(rt.release)
	select {
	case r := <-slow:
		if r.code != 200 || r.body != "A" {
			t.Errorf("request in progress = %+v, want 200 with A", r)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("request in progress did not complete")
	}
}

func TestReloadFailureKeepsCurrentConfiguration(t *testing.T) {
	rt := startReloadTest(t, testServerConfig("A", "127.0.0.1:0", "tok"), nil)
	addr := rt.addr()
	url := "http://" + addr

	tests := map[string]func() (*config.Config, error){
		"load error": func() (*config.Config, error) { return nil, errors.New("no such file") },
		"no token":   func() (*config.Config, error) { return testServerConfig("B", addr, ""), nil },
		"no listen":  func() (*config.Config, error) { return testServerConfig("B", "", "tok"), nil },
		"check fails": func() (*config.Config, error) {
			cf := testServerConfig("B", addr, "tok")
			config.Set(cf, "reject", true)
			return cf, nil
		},
		"tls toggled": func() (*config.Config, error) {
			cf := testServerConfig("B", addr, "tok")
			config.Set(cf, "server.tls.enabled", true)
			return cf, nil
		},
		"address busy": nil, // set below
	}
	busy, _ := net.Listen("tcp", "127.0.0.1:0")
	defer busy.Close()
	tests["address busy"] = func() (*config.Config, error) { return testServerConfig("B", busy.Addr().String(), "tok"), nil }

	for name, load := range tests {
		t.Run(name, func(t *testing.T) {
			before := strings.Count(rt.logs.String(), "configuration not reloaded")
			rt.setNext(load)
			rt.reload()
			deadline := time.Now().Add(5 * time.Second)
			for strings.Count(rt.logs.String(), "configuration not reloaded") == before {
				if time.Now().After(deadline) {
					t.Fatalf("no failure logged:\n%s", rt.logs.String())
				}
				time.Sleep(10 * time.Millisecond)
			}
			if code, body := get(t, plainClient, url+"/api/who", "tok"); code != 200 || body != "A" {
				t.Errorf("server changed after a failed reload: %d %q", code, body)
			}
		})
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()
	for _, m := range rt.applied {
		t.Errorf("Apply was called with %q for a configuration that failed", m)
	}
}

func TestReloadNotSupportedWithoutLoad(t *testing.T) {
	rt := startReloadTest(t, testServerConfig("A", "127.0.0.1:0", "tok"), func(o *ServeOptions) { o.Load = nil })
	addr := rt.addr()
	rt.reload()
	rt.waitLog("reloading is not supported")
	if _, body := get(t, plainClient, "http://"+addr+"/api/who", "tok"); body != "A" {
		t.Errorf("body = %q", body)
	}
}

func TestReloadChangesListenAddress(t *testing.T) {
	rt := startReloadTest(t, testServerConfig("A", "127.0.0.1:0", "tok"), nil)
	oldAddr := rt.addr()

	slow := make(chan string, 1)
	go func() {
		_, body := get(t, plainClient, "http://"+oldAddr+"/api/slow", "tok")
		slow <- body
	}()
	<-rt.entered

	// a different address as configured, which is also on a port chosen by the system
	rt.setNext(func() (*config.Config, error) { return testServerConfig("B", "localhost:0", "tok"), nil })
	rt.reload()
	newAddr := rt.addr() // the server is listening again, at the new address
	if newAddr == oldAddr {
		t.Fatal("address did not change")
	}
	rt.waitLog("configuration reloaded")

	if _, body := get(t, plainClient, "http://"+newAddr+"/api/who", "tok"); body != "B" {
		t.Errorf("new address = %q", body)
	}

	// the old address stops listening, once it has been closed, but the
	// request in progress is allowed to complete
	deadline := time.Now().Add(5 * time.Second)
	for {
		c, err := net.DialTimeout("tcp", oldAddr, 200*time.Millisecond)
		if err != nil {
			break
		}
		c.Close()
		if time.Now().After(deadline) {
			t.Fatal("old address still accepting connections")
		}
		time.Sleep(20 * time.Millisecond)
	}
	select {
	case <-slow:
		t.Fatal("request in progress was cut off")
	default:
	}
	close(rt.release)
	select {
	case body := <-slow:
		if body != "A" {
			t.Errorf("request in progress = %q, want A", body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("request in progress did not complete")
	}
}

func selfSigned(t *testing.T, cn string) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	kb, _ := x509.MarshalECPrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb})
}

func tlsConfig(t *testing.T, marker, listen, cn string) *config.Config {
	cert, key := selfSigned(t, cn)
	cf := testServerConfig(marker, listen, "tok")
	config.Set(cf, "server.tls.enabled", true)
	config.Set(cf, "server.tls.certificate", string(cert))
	config.Set(cf, "server.tls.private-key", string(key))
	return cf
}

// the certificate is replaced by a reload, without restarting
func TestReloadReplacesCertificate(t *testing.T) {
	rt := startReloadTest(t, tlsConfig(t, "A", "127.0.0.1:0", "first"), nil)
	addr := rt.addr()

	client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}}
	commonName := func() string {
		conn, err := tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		return conn.ConnectionState().PeerCertificates[0].Subject.CommonName
	}

	if cn := commonName(); cn != "first" {
		t.Fatalf("certificate = %q", cn)
	}
	if code, body := get(t, client, "https://"+addr+"/api/who", "tok"); code != 200 || body != "A" {
		t.Fatalf("get over TLS = %d %q", code, body)
	}

	rt.setNext(func() (*config.Config, error) { return tlsConfig(t, "B", addr, "second"), nil })
	rt.reload()
	rt.waitLog("configuration reloaded")

	if cn := commonName(); cn != "second" {
		t.Errorf("certificate after reload = %q", cn)
	}
	if _, body := get(t, client, "https://"+addr+"/api/who", "tok"); body != "B" {
		t.Errorf("body after reload = %q", body)
	}

	// a bad certificate is rejected and the current one continues
	rt.setNext(func() (*config.Config, error) {
		cf := tlsConfig(t, "C", addr, "third")
		config.Set(cf, "server.tls.certificate", "not a certificate")
		return cf, nil
	})
	rt.reload()
	rt.waitLog("cannot load TLS certificate")
	if cn := commonName(); cn != "second" {
		t.Errorf("certificate after a bad reload = %q", cn)
	}
}
