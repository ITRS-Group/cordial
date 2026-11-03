package instance

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

type rollTestInstance struct {
	geneos.Instance
	host *geneos.Host
	cf   *config.Config
	home string
}

func (i *rollTestInstance) Config() *config.Config { return i.cf }
func (i *rollTestInstance) Host() *geneos.Host     { return i.host }
func (i *rollTestInstance) Home() string           { return i.home }
func (i *rollTestInstance) Log() *slog.Logger      { return slog.Default() }
func (i *rollTestInstance) String() string         { return "gateway roll-test" }

func newRollTestInstance(t *testing.T) *rollTestInstance {
	t.Helper()
	cf := config.New()
	config.Set(cf, "certificate", "gateway.pem")
	config.Set(cf, "privatekey", "gateway.key")
	return &rollTestInstance{
		host: geneos.NewHost("localhost"),
		cf:   cf,
		home: t.TempDir(),
	}
}

func writeRollTestFile(t *testing.T, i *rollTestInstance, name, contents string) string {
	t.Helper()
	filename := filepath.Join(i.home, name)
	if err := os.WriteFile(filename, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return filename
}

func assertRollTestFile(t *testing.T, filename, want string) {
	t.Helper()
	contents, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != want {
		t.Fatalf("%s contains %q, want %q", filename, contents, want)
	}
}

func assertRollTestFileMissing(t *testing.T, filename string) {
	t.Helper()
	if _, err := os.Stat(filename); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %s to be absent, stat error = %v", filename, err)
	}
}

func TestRollFilesRollAndUnroll(t *testing.T) {
	i := newRollTestInstance(t)
	certificate := writeRollTestFile(t, i, "gateway.pem", "old certificate")
	key := writeRollTestFile(t, i, "gateway.key", "old key")
	writeRollTestFile(t, i, "gateway.pem.new", "renewed certificate")
	writeRollTestFile(t, i, "gateway.key.new", "renewed key")

	if err := RollFiles(i, "new", "old", "certificate", "privatekey"); err != nil {
		t.Fatalf("roll files: %v", err)
	}
	assertRollTestFile(t, certificate, "renewed certificate")
	assertRollTestFile(t, certificate+".old", "old certificate")
	assertRollTestFileMissing(t, certificate+".new")
	assertRollTestFile(t, key, "renewed key")
	assertRollTestFile(t, key+".old", "old key")
	assertRollTestFileMissing(t, key+".new")

	if err := RollFiles(i, "old", "new", "certificate", "privatekey"); err != nil {
		t.Fatalf("unroll files: %v", err)
	}
	assertRollTestFile(t, certificate, "old certificate")
	assertRollTestFile(t, certificate+".new", "renewed certificate")
	assertRollTestFileMissing(t, certificate+".old")
	assertRollTestFile(t, key, "old key")
	assertRollTestFile(t, key+".new", "renewed key")
	assertRollTestFileMissing(t, key+".old")
}

func TestRollFilesPartialFailureRestoresCompletedFiles(t *testing.T) {
	i := newRollTestInstance(t)
	certificate := writeRollTestFile(t, i, "gateway.pem", "old certificate")
	key := writeRollTestFile(t, i, "gateway.key", "old key")
	writeRollTestFile(t, i, "gateway.pem.new", "renewed certificate")

	err := RollFiles(i, "new", "old", "certificate", "privatekey")
	if err == nil {
		t.Fatal("expected roll to fail because the staged key is missing")
	}
	if !strings.Contains(err.Error(), "no .\"new\" file found") {
		t.Fatalf("roll error %q does not report the missing staged key", err)
	}
	assertRollTestFile(t, certificate, "old certificate")
	assertRollTestFile(t, certificate+".new", "renewed certificate")
	assertRollTestFileMissing(t, certificate+".old")
	assertRollTestFile(t, key, "old key")
}
