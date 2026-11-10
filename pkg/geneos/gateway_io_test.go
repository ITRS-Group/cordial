package geneos

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/itrs-group/cordial/pkg/host"
)

const gatewayTestXML = `<gateway compatibility="1"><probes/></gateway>`

type gatewayIOTestHost struct {
	host.Host
	filesystem afero.Fs
	failWrite  string
}

func (h *gatewayIOTestHost) GetFs() afero.Fs   { return h.filesystem }
func (h *gatewayIOTestHost) IsLocalhost() bool { return false }
func (h *gatewayIOTestHost) Open(name string) (io.ReadSeekCloser, error) {
	return h.filesystem.Open(name)
}
func (h *gatewayIOTestHost) Lstat(name string) (fs.FileInfo, error) { return h.filesystem.Stat(name) }
func (h *gatewayIOTestHost) Mkdir(name string, mode os.FileMode) error {
	return h.filesystem.Mkdir(name, mode)
}
func (h *gatewayIOTestHost) Remove(name string) error { return h.filesystem.Remove(name) }
func (h *gatewayIOTestHost) Rename(from, to string) error {
	if to == h.failWrite && strings.Contains(filepath.Base(from), ".tmp-") {
		return fs.ErrPermission
	}
	return h.filesystem.Rename(from, to)
}

func TestGatewayIOUsesSuppliedHost(t *testing.T) {
	h := &gatewayIOTestHost{Host: host.NewLocal(), filesystem: afero.NewMemMapFs()}
	filename := "/gateway/setup.xml"
	if err := h.filesystem.MkdirAll("/gateway", 0755); err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(h.filesystem, filename, []byte(gatewayTestXML), 0640); err != nil {
		t.Fatal(err)
	}
	gateway, err := ReadGateway(filename, WithGatewayReadHost(h))
	if err != nil {
		t.Fatal(err)
	}
	gateway.Compatibility = 2
	backup, err := WriteGateway(filename, gateway, WithGatewayWriteHost(h), WithGatewayBackup())
	if err != nil {
		t.Fatal(err)
	}
	if original, err := afero.ReadFile(h.filesystem, backup); err != nil || string(original) != gatewayTestXML {
		t.Fatalf("host backup = %q, %v; want original XML", original, err)
	}
	updated, err := ReadGateway(filename, WithGatewayReadHost(h))
	if err != nil || updated.Compatibility != 2 {
		t.Fatalf("updated host setup = %#v, %v", updated, err)
	}
	if info, err := h.filesystem.Stat(filename); err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("host setup permissions changed: %v, %v", info, err)
	}
	h.failWrite = filename
	updated.Compatibility = 3
	if _, err := WriteGateway(filename, updated, WithGatewayWriteHost(h), WithGatewayBackup()); err == nil {
		t.Fatal("host installation failure was ignored")
	}
	restored, err := ReadGateway(filename, WithGatewayReadHost(h))
	if err != nil || restored.Compatibility != 2 {
		t.Fatalf("failed host write did not restore original: %#v, %v", restored, err)
	}
}

func TestReadGatewayFromFileAndURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway setup.xml")
	if err := os.WriteFile(path, []byte(gatewayTestXML), 0o600); err != nil {
		t.Fatal(err)
	}

	fromFile, err := ReadGateway(path)
	if err != nil {
		t.Fatalf("ReadGateway(file): %v", err)
	}
	if fromFile.Compatibility != 1 || fromFile.Probes == nil {
		t.Fatalf("ReadGateway(file) = %#v", fromFile)
	}

	fileURL := (&url.URL{Scheme: "file", Path: path}).String()
	fromFileURL, err := ReadGateway(fileURL)
	if err != nil {
		t.Fatalf("ReadGateway(file URL): %v", err)
	}
	if fromFileURL.Compatibility != 1 || fromFileURL.Probes == nil {
		t.Fatalf("ReadGateway(file URL) = %#v", fromFileURL)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = io.WriteString(w, gatewayTestXML)
	}))
	defer server.Close()

	fromURL, err := ReadGateway(server.URL)
	if err != nil {
		t.Fatalf("ReadGateway(URL): %v", err)
	}
	if fromURL.Compatibility != 1 || fromURL.Probes == nil {
		t.Fatalf("ReadGateway(URL) = %#v", fromURL)
	}
}

func TestReadGatewayRejectsRemoteFileURLHost(t *testing.T) {
	if _, err := ReadGateway("file://example.test/gateway.xml"); err == nil {
		t.Fatal("ReadGateway accepted a remote file URL host")
	}
}

func TestReadGatewayInsecureTLSOption(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, gatewayTestXML)
	}))
	defer server.Close()

	if _, err := ReadGateway(server.URL); err == nil {
		t.Fatal("ReadGateway accepted an untrusted HTTPS certificate without the option")
	}
	if _, err := ReadGateway(server.URL, WithGatewayInsecureSkipVerify()); err != nil {
		t.Fatalf("ReadGateway with insecure TLS option: %v", err)
	}
}

func TestReadGatewayRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	if _, err := ReadGateway(server.URL); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("ReadGateway(404) error = %v", err)
	}
}

func TestReadGatewayTimeoutOption(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()

	_, err := ReadGateway(server.URL, WithGatewayTimeout(50*time.Millisecond))
	if err == nil {
		t.Fatal("ReadGateway returned without timing out")
	}
	select {
	case <-started:
	default:
		t.Fatal("HTTP request did not reach test server")
	}
}

func TestReadGatewayContextCancellation(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := ReadGatewayContext(ctx, server.URL)
		result <- err
	}()

	select {
	case <-started:
		cancel()
	case <-time.After(time.Second):
		cancel()
		t.Fatal("HTTP request did not reach test server")
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ReadGatewayContext error = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ReadGatewayContext did not return after cancellation")
	}
}

func TestReadGatewayRejectsNegativeTimeout(t *testing.T) {
	if _, err := ReadGateway("https://example.test/gateway.xml", WithGatewayTimeout(-time.Second)); err == nil {
		t.Fatal("ReadGateway accepted a negative timeout")
	}
}

func TestWriteGatewayBackupAndOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.setup.xml")
	if err := os.WriteFile(path, []byte("old config"), 0o640); err != nil {
		t.Fatal(err)
	}

	first := &Gateway{Compatibility: 2, Probes: &Probes{}}
	if _, err := WriteGateway(path, first); !errors.Is(err, os.ErrExist) {
		t.Fatalf("WriteGateway without overwrite policy error = %v", err)
	}
	if contents, err := os.ReadFile(path); err != nil || string(contents) != "old config" {
		t.Fatalf("existing file changed without policy: %q, %v", contents, err)
	}

	backup, err := WriteGateway(path, first, WithGatewayBackup())
	if err != nil {
		t.Fatalf("WriteGateway with backup: %v", err)
	}
	if want := path + "_0001.history"; backup != want {
		t.Fatalf("backup path = %q, want %q", backup, want)
	}
	if contents, err := os.ReadFile(backup); err != nil || string(contents) != "old config" {
		t.Fatalf("backup contents = %q, %v", contents, err)
	}
	read, err := ReadGateway(path)
	if err != nil || read.Compatibility != 2 {
		t.Fatalf("ReadGateway(written file) = %#v, %v", read, err)
	}

	second := &Gateway{Compatibility: 3, Probes: &Probes{}}
	backup, err = WriteGateway(path, second, WithGatewayBackup())
	if err != nil || backup != path+"_0001.history" {
		t.Fatalf("second backup = %q, %v", backup, err)
	}
	latestBackup, err := ReadGateway(backup)
	if err != nil || latestBackup.Compatibility != 2 {
		t.Fatalf("latest backup = %#v, %v", latestBackup, err)
	}
	olderBackup, err := os.ReadFile(path + "_0002.history")
	if err != nil || string(olderBackup) != "old config" {
		t.Fatalf("older backup contents = %q, %v", olderBackup, err)
	}

	third := &Gateway{Compatibility: 4, Probes: &Probes{}}
	if backup, err = WriteGateway(path, third, WithGatewayOverwrite()); err != nil || backup != "" {
		t.Fatalf("explicit overwrite = %q, %v", backup, err)
	}
	read, err = ReadGateway(path)
	if err != nil || read.Compatibility != 4 {
		t.Fatalf("ReadGateway(overwritten file) = %#v, %v", read, err)
	}
}

func TestWriteGatewayCapsHistoryByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.setup.xml")
	current := &Gateway{Compatibility: 0, Probes: &Probes{}}
	if _, err := WriteGateway(path, current); err != nil {
		t.Fatalf("write initial gateway: %v", err)
	}

	for version := 1; version <= DefaultGatewayHistoryLimit+2; version++ {
		current = &Gateway{Compatibility: version, Probes: &Probes{}}
		if _, err := WriteGateway(path, current, WithGatewayBackup()); err != nil {
			t.Fatalf("write gateway version %d: %v", version, err)
		}
	}

	for sequence := 1; sequence <= DefaultGatewayHistoryLimit; sequence++ {
		history := gatewayHistoryPath(path, sequence)
		if _, err := os.Stat(history); err != nil {
			t.Fatalf("history %d missing: %v", sequence, err)
		}
	}
	if _, err := os.Stat(gatewayHistoryPath(path, DefaultGatewayHistoryLimit+1)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("history beyond limit exists: %v", err)
	}

	newest, err := ReadGateway(gatewayHistoryPath(path, 1))
	if err != nil || newest.Compatibility != DefaultGatewayHistoryLimit+1 {
		t.Fatalf("newest history = %#v, %v", newest, err)
	}
	oldest, err := ReadGateway(gatewayHistoryPath(path, DefaultGatewayHistoryLimit))
	if err != nil || oldest.Compatibility != 2 {
		t.Fatalf("oldest retained history = %#v, %v", oldest, err)
	}
}

func TestWriteGatewayRejectsConflictingOptions(t *testing.T) {
	_, err := WriteGateway("unused.xml", &Gateway{}, WithGatewayBackup(), WithGatewayOverwrite())
	if err == nil {
		t.Fatal("WriteGateway accepted both backup and overwrite")
	}
}

func TestWriteGatewayHistoryLimitOption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.setup.xml")
	current := &Gateway{Compatibility: 0, Probes: &Probes{}}
	if _, err := WriteGateway(path, current); err != nil {
		t.Fatal(err)
	}
	for version := 1; version <= 3; version++ {
		current = &Gateway{Compatibility: version, Probes: &Probes{}}
		if _, err := WriteGateway(path, current, WithGatewayBackup(), WithGatewayHistoryLimit(2)); err != nil {
			t.Fatalf("write version %d: %v", version, err)
		}
	}
	if _, err := os.Stat(gatewayHistoryPath(path, 3)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("history beyond custom limit exists: %v", err)
	}
}
