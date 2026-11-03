package tlscmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func setLocalHostForTest(t *testing.T) {
	t.Helper()
	previous := geneos.LOCAL
	geneos.LOCAL = &geneos.Host{Host: host.NewLocal(), Config: config.New()}
	t.Cleanup(func() { geneos.LOCAL = previous })
}

func TestCreateKeyPropagatesWriteError(t *testing.T) {
	setLocalHostForTest(t)
	destination := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(destination, []byte("file"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, _, err := CreateKey(destination, false, certs.ECDSA); err == nil {
		t.Fatal("CreateKey() succeeded when destination is not a directory")
	}
}

func TestCreateKeyWritesPrivateKeyWithRestrictivePermissions(t *testing.T) {
	setLocalHostForTest(t)
	destination := t.TempDir()
	if _, _, err := CreateKey(destination, false, certs.ECDSA); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	keyPath := filepath.Join(destination, "private-key.pem")
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("private key permissions = %04o, want 0600", got)
	}
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if _, err := certs.ReadPrivateKeyFromPEM(data); err != nil {
		t.Fatalf("ReadPrivateKeyFromPEM() error = %v", err)
	}
}

func TestCreateKeyRejectsInvalidTypeWithoutCreatingFile(t *testing.T) {
	setLocalHostForTest(t)
	destination := t.TempDir()
	if _, _, err := CreateKey(destination, false, certs.KeyType("invalid")); err == nil {
		t.Fatal("CreateKey() succeeded for an invalid key type")
	}
	if _, err := os.Stat(filepath.Join(destination, "private-key.pem")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("private key file exists after invalid key type, stat error = %v", err)
	}
}

func TestCreateKeyDoesNotOverwriteWithoutPermission(t *testing.T) {
	setLocalHostForTest(t)
	destination := t.TempDir()
	keyPath := filepath.Join(destination, "private-key.pem")
	original := []byte("existing key material")
	if err := os.WriteFile(keyPath, original, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, _, err := CreateKey(destination, false, certs.ECDSA); !errors.Is(err, os.ErrExist) {
		t.Fatalf("CreateKey() error = %v, want os.ErrExist", err)
	}
	got, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("existing key was changed to %q", got)
	}
}
