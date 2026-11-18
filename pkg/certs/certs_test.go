package certs

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/host"
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

// testCA creates a self-signed CA certificate and its key for cn,
// valid between notBefore and notAfter. The key usage is ku.
func testCA(t *testing.T, cn string, notBefore, notAfter time.Time, ku x509.KeyUsage) (*x509.Certificate, PrivateKey) {
	t.Helper()
	keyDER, _, err := GenerateKey(ECDSA)
	if err != nil {
		t.Fatal(err)
	}
	k, _, err := ParsePrivateKey(keyDER)
	if err != nil {
		t.Fatal(err)
	}
	signer := k.(crypto.Signer)
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              ku,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, signer.Public(), signer)
	if err != nil {
		t.Fatal(err)
	}
	c, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return c, keyDER
}

func validRoot(t *testing.T, cn string) (*x509.Certificate, PrivateKey) {
	t.Helper()
	now := time.Now()
	return testCA(t, cn, now.Add(-time.Hour), now.Add(time.Hour), x509.KeyUsageCertSign|x509.KeyUsageCRLSign)
}

func TestCreateCertificateSelfSigned(t *testing.T) {
	tmpl := Template("self signed root", IsCA(), BasicConstraintsValid(), KeyUsage(x509.KeyUsageCertSign), ExtKeyUsage())

	// both without and with a signing key, which is only used to
	// select the key type and must not sign the certificate
	other, otherKey := validRoot(t, "other")
	for name, signingKey := range map[string]PrivateKey{"nil key": nil, "other key": otherKey} {
		t.Run(name, func(t *testing.T) {
			root, key, err := CreateCertificate(tmpl, tmpl, signingKey)
			if err != nil {
				t.Fatalf("CreateCertificate() error = %v", err)
			}
			if err := root.CheckSignatureFrom(root); err != nil {
				t.Errorf("self-signed root does not verify against itself: %v", err)
			}
			if err := root.CheckSignatureFrom(other); err == nil {
				t.Error("self-signed root verified against an unrelated certificate")
			}
			if !CheckKeyMatch(key, root) {
				t.Error("returned key does not match the certificate")
			}
			if !IsValidRootCA(root) {
				t.Error("self-signed root is not a valid root CA")
			}
		})
	}
}

func TestCreateCertificateSubjectKeyID(t *testing.T) {
	root, rootKey := createTestRoot(t)
	tmpl := Template("leaf.example", DNSNames("leaf.example"))

	var skids [][]byte
	for range 2 {
		leaf, _, err := CreateCertificate(tmpl, root, rootKey)
		if err != nil {
			t.Fatal(err)
		}
		if len(leaf.SubjectKeyId) != sha1.Size {
			t.Fatalf("SubjectKeyId length = %d, want %d", len(leaf.SubjectKeyId), sha1.Size)
		}
		if want := sha1.Sum(nil); bytes.Equal(leaf.SubjectKeyId, want[:]) {
			t.Fatal("SubjectKeyId is the hash of no data")
		}
		want, err := subjectKeyID(leaf.PublicKey)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(leaf.SubjectKeyId, want) {
			t.Errorf("SubjectKeyId = %x, want %x derived from the public key", leaf.SubjectKeyId, want)
		}
		if !bytes.Equal(leaf.AuthorityKeyId, root.SubjectKeyId) {
			t.Errorf("AuthorityKeyId = %x, want %x", leaf.AuthorityKeyId, root.SubjectKeyId)
		}
		skids = append(skids, leaf.SubjectKeyId)

		if tmpl.SubjectKeyId != nil {
			t.Fatal("CreateCertificate() modified the caller's template")
		}
	}
	if bytes.Equal(skids[0], skids[1]) {
		t.Error("two certificates created from one template have the same SubjectKeyId")
	}

	// an explicit key id is kept
	tmpl.SubjectKeyId = []byte{9, 9}
	leaf, _, err := CreateCertificate(tmpl, root, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leaf.SubjectKeyId, []byte{9, 9}) {
		t.Errorf("explicit SubjectKeyId = %x, want 0909", leaf.SubjectKeyId)
	}
}

func TestCreateCertificateInvalidArguments(t *testing.T) {
	root, rootKey := createTestRoot(t)
	tmpl := Template("x")

	for name, fn := range map[string]func() error{
		"nil template": func() error { _, _, err := CreateCertificate(nil, root, rootKey); return err },
		"nil parent":   func() error { _, _, err := CreateCertificate(tmpl, nil, rootKey); return err },
		"no key":       func() error { _, _, err := CreateCertificate(tmpl, root, nil); return err },
		"bad key type": func() error { _, _, err := CreateCertificate(tmpl, root, rootKey, WithKeyType("bogus")); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := fn(); err == nil {
				t.Fatal("CreateCertificate() succeeded")
			}
		})
	}
}

func TestVerifyIsNotLimitedToServerAuth(t *testing.T) {
	root, rootKey := createTestRoot(t)
	for name, eku := range map[string][]x509.ExtKeyUsage{
		"client only":  {x509.ExtKeyUsageClientAuth},
		"code signing": {x509.ExtKeyUsageCodeSigning},
		"none":         nil,
	} {
		t.Run(name, func(t *testing.T) {
			leaf, leafKey, err := CreateCertificate(Template("client", ExtKeyUsage(eku...)), root, rootKey)
			if err != nil {
				t.Fatal(err)
			}
			if !Verify(leaf, root) {
				t.Error("Verify() rejected the chain")
			}
			if _, _, _, err := ParseCertChain(leaf, root); err != nil {
				t.Errorf("ParseCertChain() error = %v", err)
			}
			keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: leafKey})
			bundle, err := ParsePEM(keyPEM, certificatePEM(leaf), certificatePEM(root))
			if err != nil {
				t.Fatal(err)
			}
			if !bundle.Valid {
				t.Error("ParsePEM() bundle is not valid")
			}
		})
	}
}

func TestVerifyNilAndEmpty(t *testing.T) {
	if Verify() || Verify(nil) {
		t.Error("Verify() succeeded with no certificates")
	}
	if _, _, _, err := ParseCertChain(nil); err == nil {
		t.Error("ParseCertChain(nil) succeeded")
	}
}

func TestIsValidRootCA(t *testing.T) {
	now := time.Now()
	good, _ := validRoot(t, "root")
	noUsage, _ := testCA(t, "root", now.Add(-time.Hour), now.Add(time.Hour), 0)
	sigOnly, _ := testCA(t, "root", now.Add(-time.Hour), now.Add(time.Hour), x509.KeyUsageDigitalSignature)
	expired, _ := testCA(t, "root", now.Add(-2*time.Hour), now.Add(-time.Hour), x509.KeyUsageCertSign)

	// same CN in the subject and the issuer, but a different issuer
	_, parentKey := validRoot(t, "unused")
	parent := mustParseIssuer(t, &x509.Certificate{
		SerialNumber: big.NewInt(5), Subject: pkix.Name{CommonName: "same", Organization: []string{"Other Org"}},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}, parentKey)
	sameCN, _, err := CreateCertificate(Template("same", IsCA(), KeyUsage(x509.KeyUsageCertSign), ExtKeyUsage()), parent, parentKey)
	if err != nil {
		t.Fatal(err)
	}
	if sameCN.Subject.CommonName != sameCN.Issuer.CommonName {
		t.Fatal("test setup: expected matching common names")
	}

	for name, tt := range map[string]struct {
		cert *x509.Certificate
		want bool
	}{
		"nil":                {nil, false},
		"valid":              {good, true},
		"no key usage":       {noUsage, true},
		"no certsign":        {sigOnly, false},
		"expired":            {expired, false},
		"same CN, other org": {sameCN, false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := IsValidRootCA(tt.cert); got != tt.want {
				t.Errorf("IsValidRootCA() = %t, want %t", got, tt.want)
			}
		})
	}
}

// mustParseIssuer returns the self-signed certificate for tmpl and
// key, for use as a parent certificate.
func mustParseIssuer(t *testing.T, tmpl *x509.Certificate, keyDER PrivateKey) *x509.Certificate {
	t.Helper()
	k, _, err := ParsePrivateKey(keyDER)
	if err != nil {
		t.Fatal(err)
	}
	signer := k.(crypto.Signer)
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, signer.Public(), signer)
	if err != nil {
		t.Fatal(err)
	}
	c, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestIsValidLeafCert(t *testing.T) {
	root, rootKey := createTestRoot(t)
	now := time.Now()
	leaf := func(mod func(*x509.Certificate)) *x509.Certificate {
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(7),
			Subject:      pkix.Name{CommonName: "leaf"},
			NotBefore:    now.Add(-time.Hour),
			NotAfter:     now.Add(time.Hour),
		}
		mod(tmpl)
		c, _, err := CreateCertificate(tmpl, root, rootKey)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}

	for name, tt := range map[string]struct {
		cert *x509.Certificate
		want bool
	}{
		"nil":                 {nil, false},
		"no key usage or EKU": {leaf(func(*x509.Certificate) {}), true},
		"unknown EKU only": {leaf(func(c *x509.Certificate) {
			c.KeyUsage = x509.KeyUsageDigitalSignature
			c.UnknownExtKeyUsage = []asn1.ObjectIdentifier{{1, 2, 3, 4}}
		}), true},
		"key encipherment": {leaf(func(c *x509.Certificate) { c.KeyUsage = x509.KeyUsageKeyEncipherment }), true},
		"cert sign only":   {leaf(func(c *x509.Certificate) { c.KeyUsage = x509.KeyUsageCertSign }), false},
		"CA":               {root, false},
		"expired": {leaf(func(c *x509.Certificate) {
			c.NotBefore, c.NotAfter = now.Add(-2*time.Hour), now.Add(-time.Hour)
		}), false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := IsValidLeafCert(tt.cert); got != tt.want {
				t.Errorf("IsValidLeafCert() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestDecodePEM(t *testing.T) {
	now := time.Now()
	root, rootKey := createTestRoot(t)
	expired, _ := testCA(t, "old root", now.Add(-2*time.Hour), now.Add(-time.Hour), x509.KeyUsageCertSign)

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ecDER, err := x509.MarshalECPrivateKey(ecKey)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("keys are normalised to PKCS#8", func(t *testing.T) {
		for name, block := range map[string]*pem.Block{
			"pkcs1": {Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaKey)},
			"sec1":  {Type: "EC PRIVATE KEY", Bytes: ecDER},
			"pkcs8": {Type: "PRIVATE KEY", Bytes: rootKey},
		} {
			_, _, _, keys, err := DecodePEM(pem.EncodeToMemory(block))
			if err != nil || len(keys) != 1 {
				t.Fatalf("%s: DecodePEM() = %d keys, %v", name, len(keys), err)
			}
			if _, err := x509.ParsePKCS8PrivateKey(keys[0]); err != nil {
				t.Errorf("%s: returned key is not PKCS#8: %v", name, err)
			}
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		_, _, _, _, err := DecodePEM(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("junk")}))
		if err == nil {
			t.Fatal("DecodePEM() accepted a malformed private key")
		}
	})

	t.Run("encrypted keys are skipped", func(t *testing.T) {
		_, _, _, keys, err := DecodePEM(
			pem.EncodeToMemory(&pem.Block{Type: "ENCRYPTED PRIVATE KEY", Bytes: []byte("junk")}),
			pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Headers: map[string]string{"Proc-Type": "4,ENCRYPTED"}, Bytes: []byte("junk")}),
		)
		if err != nil || len(keys) != 0 {
			t.Fatalf("DecodePEM() = %d keys, %v", len(keys), err)
		}
	})

	t.Run("expired certificates are reported by ParsePEM", func(t *testing.T) {
		_, _, roots, _, err := DecodePEM(certificatePEM(expired))
		if err != nil || len(roots) != 0 {
			t.Fatalf("DecodePEM() = %d roots, %v", len(roots), err)
		}
		_, err = ParsePEM(certificatePEM(expired))
		if err == nil || !strings.Contains(err.Error(), "expired") || !strings.Contains(err.Error(), "old root") {
			t.Fatalf("ParsePEM() error = %v, want one that mentions the expired certificate", err)
		}
	})

	t.Run("valid root", func(t *testing.T) {
		_, _, roots, _, err := DecodePEM(certificatePEM(root))
		if err != nil || len(roots) != 1 {
			t.Fatalf("DecodePEM() = %d roots, %v", len(roots), err)
		}
	})
}

func TestWritePrivateKeyToNormalisesToPKCS8(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ecDER, err := x509.MarshalECPrivateKey(ecKey)
	if err != nil {
		t.Fatal(err)
	}

	for name, der := range map[string]PrivateKey{
		"pkcs1": x509.MarshalPKCS1PrivateKey(rsaKey),
		"sec1":  ecDER,
	} {
		t.Run(name, func(t *testing.T) {
			var b bytes.Buffer
			n, err := WritePrivateKeyTo(&b, der)
			if err != nil {
				t.Fatal(err)
			}
			if n != b.Len() {
				t.Errorf("WritePrivateKeyTo() n = %d, wrote %d bytes", n, b.Len())
			}
			block, _ := pem.Decode(b.Bytes())
			if block == nil || block.Type != "PRIVATE KEY" {
				t.Fatalf("block = %v, want type PRIVATE KEY", block)
			}
			if _, err := x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
				t.Errorf("written key is not PKCS#8: %v", err)
			}
		})
	}

	var b bytes.Buffer
	if n, err := WritePrivateKeyTo(&b, []byte("junk")); err == nil || n != 0 || b.Len() != 0 {
		t.Errorf("WritePrivateKeyTo(junk) = %d, %v with %d bytes written, want an error and no output", n, err, b.Len())
	}
}

func TestWriteCertificatesToCountsAllBytes(t *testing.T) {
	a, _ := validRoot(t, "a")
	b, _ := validRoot(t, "b")
	var buf bytes.Buffer
	n, err := WriteCertificatesTo(&buf, a, nil, b)
	if err != nil {
		t.Fatal(err)
	}
	if n != buf.Len() {
		t.Errorf("WriteCertificatesTo() = %d, but wrote %d bytes", n, buf.Len())
	}
}

func TestUpdateCACertsFilesDoesNotModifyArguments(t *testing.T) {
	now := time.Now()
	expired, _ := testCA(t, "old", now.Add(-2*time.Hour), now.Add(-time.Hour), x509.KeyUsageCertSign)
	good, _ := validRoot(t, "good")
	roots := []*x509.Certificate{expired, nil, good}

	base := filepath.Join(t.TempDir(), "ca-bundle")
	updated, err := UpdateCACertsFiles(host.Localhost, base, roots...)
	if err != nil || !updated {
		t.Fatalf("UpdateCACertsFiles() = %t, %v", updated, err)
	}
	if roots[0] != expired || roots[1] != nil || roots[2] != good {
		t.Errorf("UpdateCACertsFiles() modified the caller's slice: %v", roots)
	}

	got, err := ReadCertificates(host.Localhost, base+PEMExtension)
	if err != nil || len(got) != 1 || !got[0].Equal(good) {
		t.Errorf("ca bundle contains %d certificates (%v), want only the valid root", len(got), err)
	}

	// a second call with the same roots is a no-op
	if updated, err = UpdateCACertsFiles(host.Localhost, base, roots...); err != nil || updated {
		t.Errorf("second UpdateCACertsFiles() = %t, %v, want no update", updated, err)
	}
}

func TestReadCertPool(t *testing.T) {
	dir := t.TempDir()
	good, _ := validRoot(t, "good")
	goodPath := filepath.Join(dir, "good.pem")
	emptyPath := filepath.Join(dir, "empty.pem")
	if err := os.WriteFile(goodPath, certificatePEM(good), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(emptyPath, []byte("nothing here"), 0o600); err != nil {
		t.Fatal(err)
	}

	if pool := ReadCertPool(host.Localhost, goodPath); pool == nil {
		t.Error("ReadCertPool() = nil for a valid file")
	}
	for name, path := range map[string]string{"missing": filepath.Join(dir, "missing.pem"), "no certificates": emptyPath} {
		if pool := ReadCertPool(host.Localhost, path); pool != nil {
			t.Errorf("ReadCertPool(%s) = %v, want nil", name, pool)
		}
	}
}

func TestReadPrivateKey(t *testing.T) {
	dir := t.TempDir()
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ecDER, err := x509.MarshalECPrivateKey(ecKey)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(ecKey)
	if err != nil {
		t.Fatal(err)
	}

	write := func(name string, blocks ...*pem.Block) string {
		var data []byte
		for _, b := range blocks {
			data = append(data, pem.EncodeToMemory(b)...)
		}
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	got, err := ReadPrivateKey(host.Localhost, write("sec1.key", &pem.Block{Type: "EC PRIVATE KEY", Bytes: ecDER}))
	if err != nil || !bytes.Equal(got, pkcs8) {
		t.Errorf("ReadPrivateKey(SEC1) = %x, %v, want the PKCS#8 form of the key", got, err)
	}

	encrypted := write("enc.key", &pem.Block{Type: "ENCRYPTED PRIVATE KEY", Bytes: []byte("junk")})
	if _, err := ReadPrivateKey(host.Localhost, encrypted); err == nil || !strings.Contains(err.Error(), "encrypted") {
		t.Errorf("ReadPrivateKey(encrypted) error = %v, want an encrypted key error", err)
	}

	// an encrypted key is skipped if there is a usable one as well
	both := write("both.key", &pem.Block{Type: "ENCRYPTED PRIVATE KEY", Bytes: []byte("junk")}, &pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})
	if got, err := ReadPrivateKey(host.Localhost, both); err != nil || !bytes.Equal(got, pkcs8) {
		t.Errorf("ReadPrivateKey(both) = %x, %v", got, err)
	}

	if _, err := ReadPrivateKeyFromPEM([]byte("nothing")); err == nil {
		t.Error("ReadPrivateKeyFromPEM(no key) succeeded")
	}
}

func TestNewTemplate(t *testing.T) {
	for name, opts := range map[string][]TemplateOption{
		"bad IP":  {IPAddresses("192.0.2.1", "not-an-ip")},
		"bad URI": {URIs("https://ok.example/", "http://bad host/%zz")},
	} {
		t.Run(name, func(t *testing.T) {
			if tmpl, err := NewTemplate("x", opts...); err == nil || tmpl != nil {
				t.Errorf("NewTemplate() = %v, %v, want an error", tmpl, err)
			}
			if tmpl := Template("x", opts...); tmpl != nil {
				t.Error("Template() returned a template for invalid options")
			}
		})
	}

	tmpl, err := NewTemplate("x", IPAddresses("192.0.2.1", "2001:db8::1"), URIs("spiffe://example/x"))
	if err != nil || len(tmpl.IPAddresses) != 2 || len(tmpl.URIs) != 1 {
		t.Errorf("NewTemplate() = %+v, %v", tmpl, err)
	}

	// a nil template is rejected cleanly
	root, rootKey := createTestRoot(t)
	if _, _, err := CreateCertificate(Template("x", IPAddresses("bogus")), root, rootKey); !errors.Is(err, os.ErrInvalid) {
		t.Errorf("CreateCertificate() with an invalid template error = %v, want os.ErrInvalid", err)
	}
}

func TestTemplateIsCA(t *testing.T) {
	// IsCA alone must produce a working CA
	tmpl := Template("ca", IsCA(), ExtKeyUsage())
	if !tmpl.IsCA || !tmpl.BasicConstraintsValid || tmpl.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Fatalf("Template(IsCA()) = IsCA %t, BasicConstraintsValid %t, KeyUsage %v", tmpl.IsCA, tmpl.BasicConstraintsValid, tmpl.KeyUsage)
	}
	ca, _, err := CreateCertificate(tmpl, tmpl, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ca.IsCA || !IsValidRootCA(ca) {
		t.Error("certificate created from an IsCA() template is not a valid root CA")
	}

	// an explicit key usage is respected
	if tmpl := Template("ca", IsCA(), KeyUsage(x509.KeyUsageCRLSign)); tmpl.KeyUsage != x509.KeyUsageCRLSign {
		t.Errorf("KeyUsage = %v, want CRLSign only", tmpl.KeyUsage)
	}
}
