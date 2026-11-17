package certs

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

func TestP12ToRootCerts(t *testing.T) {
	root, rootKey := createTestRoot(t)
	now := time.Now()
	leaf, leafKey, err := CreateCertificate(&x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "service.example"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, root, rootKey)
	if err != nil {
		t.Fatalf("CreateCertificate(leaf) error = %v", err)
	}
	parsedLeafKey, _, err := ParsePrivateKey(leafKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey() error = %v", err)
	}

	const password = "secret"
	write := func(name string, data []byte, err error) string {
		t.Helper()
		if err != nil {
			t.Fatalf("encoding %s: %v", name, err)
		}
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	rootStoreData, err := pkcs12.Modern.EncodeTrustStore([]*x509.Certificate{root}, password)
	rootStore := write("roots.p12", rootStoreData, err)
	mixedStoreData, err := pkcs12.Modern.EncodeTrustStore([]*x509.Certificate{leaf, root}, password)
	mixedStore := write("mixed.p12", mixedStoreData, err)
	bundleData, err := pkcs12.Modern.Encode(parsedLeafKey, leaf, []*x509.Certificate{root}, password)
	bundle := write("bundle.p12", bundleData, err)

	tests := []struct {
		name                string
		path, password      string
		roots, others, keys int
		wantErr             bool
	}{
		{"truststore with only root", rootStore, password, 1, 0, 0, false},
		{"truststore with root and leaf", mixedStore, password, 1, 1, 0, false},
		{"key bundle", bundle, password, 1, 1, 1, false},
		{"wrong password", rootStore, "wrong", 0, 0, 0, true},
		{"missing file", filepath.Join(t.TempDir(), "none.p12"), password, 0, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roots, others, keys, err := P12ToRootCerts(tt.path, []byte(tt.password))
			if (err != nil) != tt.wantErr {
				t.Fatalf("P12ToRootCerts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(roots) != tt.roots || others != tt.others || keys != tt.keys {
				t.Errorf("P12ToRootCerts() = %d roots, %d others, %d keys; want %d, %d, %d",
					len(roots), others, keys, tt.roots, tt.others, tt.keys)
			}
		})
	}
}
