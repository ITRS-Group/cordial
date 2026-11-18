package instance_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func TestSecureArgsMinimumTLSVersion(t *testing.T) {
	newTestRoot(t)

	i, err := instance.Add(geneos.LOCAL, &licd.Component, "tls-version-test", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i.Unload() })

	cf := i.Config()
	config.Set(cf, cf.Join(instance.TLSBASE, instance.CERTIFICATE), "cert.pem")
	config.Set(cf, cf.Join(instance.TLSBASE, instance.PRIVATEKEY), "key.pem")

	globalKey := config.Join(instance.TLSBASE, instance.MINVERSION)
	t.Cleanup(func() { config.Delete(config.Global(), globalKey) })

	minVersion := func() string {
		t.Helper()
		args, _, _, err := instance.SecureArgs(i)
		if err != nil {
			t.Fatal(err)
		}
		pos := slices.Index(args, "-minTLSversion")
		if pos < 0 || pos+1 >= len(args) {
			return ""
		}
		return args[pos+1]
	}

	// instance, then global, then 1.2
	if got := minVersion(); got != "1.2" {
		t.Errorf("default minimum TLS version = %q, want 1.2", got)
	}

	config.Set(config.Global(), globalKey, "1.3")
	if got := minVersion(); got != "1.3" {
		t.Errorf("global minimum TLS version = %q, want 1.3", got)
	}

	config.Set(cf, cf.Join(instance.TLSBASE, instance.MINVERSION), "1.1")
	if got := minVersion(); got != "1.1" {
		t.Errorf("instance minimum TLS version = %q, want 1.1", got)
	}
}

// TestSnapshotVersionLimit checks that 5.14 is accepted, as the message
// says "5.14 or above"
func TestSnapshotVersionLimit(t *testing.T) {
	root := newTestRoot(t)

	packages := filepath.Join(root, "packages", "licd")
	if err := os.RemoveAll(filepath.Join(packages, "active_prod")); err != nil {
		t.Fatal(err)
	}

	i, err := instance.Add(geneos.LOCAL, &licd.Component, "snapshot-version-test", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i.Unload() })

	for version, tooOld := range map[string]bool{"5.13.2": true, "5.14.0": false, "5.14.1": false, "6.0.0": false} {
		if err := os.MkdirAll(filepath.Join(packages, version), 0o755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(packages, "active_prod")
		_ = os.Remove(link)
		if err := os.Symlink(version, link); err != nil {
			t.Fatal(err)
		}

		// there are no paths to look up, so only the version is checked
		_, err := instance.SnapshotDataviews(i, nil)
		if got := err != nil && strings.Contains(err.Error(), "too old"); got != tooOld {
			t.Errorf("version %s: too old = %v (%v), want %v", version, got, err, tooOld)
		}
	}
}

func TestReadAESKeyFile(t *testing.T) {
	newTestRoot(t)

	i, err := instance.Add(geneos.LOCAL, &licd.Component, "keyfile-test", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i.Unload() })

	keyfile, wantCRC, err := instance.CreateAESKeyFile(i)
	if err != nil {
		t.Fatal(err)
	}

	got, kv, crc, err := instance.ReadAESKeyFile(i)
	if err != nil || got != keyfile || kv == nil {
		t.Fatalf("ReadAESKeyFile() = %q, %v, %d, %v", got, kv, crc, err)
	}
	if crc == 0 || crc != wantCRC {
		t.Errorf("ReadAESKeyFile() crc = %d, want %d", crc, wantCRC)
	}

	// the setting to read the file name from can be given
	config.Set(i.Config(), "prevkeyfile", string(keyfile))
	config.Set(i.Config(), "keyfile", filepath.Join(t.TempDir(), "missing.aes"))
	if _, _, _, err := instance.ReadAESKeyFile(i); err == nil {
		t.Error("ReadAESKeyFile() of a missing key file succeeded")
	}
	if got, _, crc, err := instance.ReadAESKeyFile(i, "prevkeyfile"); err != nil || got != keyfile || crc != wantCRC {
		t.Errorf("ReadAESKeyFile(prevkeyfile) = %q, %d, %v, want %q, %d", got, crc, err, keyfile, wantCRC)
	}
}
