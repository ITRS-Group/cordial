package certs

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

func createTestRoot(t *testing.T) (*x509.Certificate, PrivateKey) {
	t.Helper()
	privateKey, _, err := GenerateKey(ECDSA)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	parsedKey, _, err := ParsePrivateKey(privateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey() error = %v", err)
	}
	signer, ok := parsedKey.(crypto.Signer)
	if !ok {
		t.Fatal("root private key does not implement crypto.Signer")
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test root"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		SubjectKeyId:          []byte{1, 2, 3, 4},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, signer.Public(), signer)
	if err != nil {
		t.Fatalf("CreateCertificate(root) error = %v", err)
	}
	root, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate(root) error = %v", err)
	}
	return root, privateKey
}

func certificatePEM(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
}

func TestCreateCertificateAndParsePEM(t *testing.T) {
	root, rootKey := createTestRoot(t)
	now := time.Now()
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "service.example"},
		DNSNames:     []string{"service.example"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leaf, leafKey, err := CreateCertificate(leafTemplate, root, rootKey)
	if err != nil {
		t.Fatalf("CreateCertificate(leaf) error = %v", err)
	}
	if !CheckKeyMatch(leafKey, leaf) {
		t.Fatal("created private key does not match the leaf certificate")
	}
	roots := x509.NewCertPool()
	roots.AddCert(root)
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: "service.example"}); err != nil {
		t.Fatalf("leaf Verify() error = %v", err)
	}
	if !Verify(leaf, root) {
		t.Fatal("Verify() rejected a valid leaf/root chain")
	}
	parsedLeaf, intermediates, parsedRoot, err := ParseCertChain(leaf, root)
	if err != nil {
		t.Fatalf("ParseCertChain() error = %v", err)
	}
	if !parsedLeaf.Equal(leaf) || len(intermediates) != 0 || !parsedRoot.Equal(root) {
		t.Fatalf("ParseCertChain() returned an unexpected chain: leaf=%v intermediates=%d root=%v", parsedLeaf, len(intermediates), parsedRoot)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: leafKey})
	bundle, err := ParsePEM(keyPEM, certificatePEM(leaf), certificatePEM(root))
	if err != nil {
		t.Fatalf("ParsePEM() error = %v", err)
	}
	if !bundle.Valid || bundle.Key == nil || bundle.Leaf == nil || bundle.Root == nil {
		t.Fatalf("ParsePEM() returned an incomplete or invalid bundle: %#v", bundle)
	}
}

func TestParsePEMDoesNotTrustMismatchedPrivateKey(t *testing.T) {
	root, rootKey := createTestRoot(t)
	now := time.Now()
	leaf, _, err := CreateCertificate(&x509.Certificate{
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
	otherKey, _, err := GenerateKey(ECDSA)
	if err != nil {
		t.Fatalf("GenerateKey(other) error = %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: otherKey})
	bundle, err := ParsePEM(keyPEM, certificatePEM(leaf), certificatePEM(root))
	if err != nil {
		t.Fatalf("ParsePEM() error = %v", err)
	}
	if bundle.Valid || bundle.Key != nil {
		t.Fatal("ParsePEM() accepted a private key that does not match the leaf certificate")
	}
}
