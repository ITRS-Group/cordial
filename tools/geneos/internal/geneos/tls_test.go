package geneos

import (
	"bytes"
	"crypto/x509"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

func TestImportRootsOnly(t *testing.T) {
	dir := t.TempDir()
	h := &Host{Host: host.NewLocal(), Config: config.New()}
	config.Set(h.Config, cordial.ExecutableName(), dir)

	root1, err := certs.WriteNewRootCert(filepath.Join(dir, "root1"), "root one", certs.ECDSA)
	if err != nil {
		t.Fatal(err)
	}
	root2, err := certs.WriteNewRootCert(filepath.Join(dir, "root2"), "root two", certs.ECDSA)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := importRootsOnly(&out, h, "file1.pem", []*x509.Certificate{root1}, 2, 1, true); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`Importing root certificate(s) from "file1.pem"`, "not changed", "root one", "Ignoring 2 non-root", "Ignoring 1 private key", "updated"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q:\n%s", want, out.String())
		}
	}

	// a second root is added to the existing bundle, keeping the first
	out.Reset()
	if err := importRootsOnly(&out, h, "file2.pem", []*x509.Certificate{root2}, 0, 0, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "contains only root certificate(s)") {
		t.Errorf("unexpected output:\n%s", out.String())
	}
	bundle, err := certs.ReadCertificates(h, PathToCABundlePEM(h))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle) != 2 {
		t.Fatalf("ca-bundle has %d certificates, want 2", len(bundle))
	}

	// importing again changes nothing
	out.Reset()
	if err := importRootsOnly(&out, h, "file2.pem", []*x509.Certificate{root2}, 0, 0, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "no changes made") {
		t.Errorf("unexpected output:\n%s", out.String())
	}

	if err := importRootsOnly(&out, h, "none.pem", nil, 3, 0, true); err == nil {
		t.Error("importRootsOnly() succeeded with no root certificates")
	}
}
