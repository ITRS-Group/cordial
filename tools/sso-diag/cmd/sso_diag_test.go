package cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jcmturner/goidentity/v6"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestSplitUsername(t *testing.T) {
	tests := []struct {
		in, user, realm string
	}{
		{"user", "user", ""},
		{"user@EXAMPLE.COM", "user", "EXAMPLE.COM"},
		{`EXAMPLE\user`, "user", "EXAMPLE"},
		{"", "", ""},
	}
	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			if user, realm := SplitUsername(test.in); user != test.user || realm != test.realm {
				t.Errorf("SplitUsername(%q) = %q, %q, want %q, %q", test.in, user, realm, test.user, test.realm)
			}
		})
	}
}

func TestLDAPFilterEscapesUser(t *testing.T) {
	got := userFilter("(objectClass=person)", "sAMAccountName", "a)(uid=*))(|(x=*")
	want := `(&(objectClass=person)(sAMAccountName=a\29\28uid=\2a\29\29\28|\28x=\2a))`
	if got != want {
		t.Errorf("filter = %s, want %s", got, want)
	}
}

func TestLDAPTLSSkipsVerificationByDefault(t *testing.T) {
	old := ldapVerify
	t.Cleanup(func() { ldapVerify = old })

	ldapVerify = false
	if !ldapTLSConfig().InsecureSkipVerify {
		t.Error("LDAP certificates are verified by default")
	}
	ldapVerify = true
	if ldapTLSConfig().InsecureSkipVerify {
		t.Error("--ldap-verify has no effect")
	}
}

func TestBuildKeytabKVNO(t *testing.T) {
	tests := []struct {
		kvno    int
		wantErr bool
	}{
		{0, false}, {1, false}, {255, false}, {256, true}, {1000, true}, {-1, true},
	}
	for _, test := range tests {
		kt, err := buildKeytab("svc", "EXAMPLE.COM", "secret", test.kvno, []string{"aes256-cts-hmac-sha1-96"})
		if (err != nil) != test.wantErr {
			t.Errorf("kvno %d: err = %v, want error %v", test.kvno, err, test.wantErr)
			continue
		}
		if err == nil && len(kt.Entries) != 1 {
			t.Errorf("kvno %d: entries = %d, want 1", test.kvno, len(kt.Entries))
		}
	}
}

func TestPrivateKeyFromEntry(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsaDER, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	if err != nil {
		t.Fatal(err)
	}
	_, edKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	edDER, err := x509.MarshalPKCS8PrivateKey(edKey)
	if err != nil {
		t.Fatal(err)
	}

	if k, err := privateKeyFromEntry(rsaDER); err != nil || k == nil {
		t.Errorf("RSA key: %v, %v", k, err)
	}
	if k, err := privateKeyFromEntry(edDER); err == nil || k != nil {
		t.Errorf("non-RSA key was accepted: %v, %v", k, err)
	}
	if _, err := privateKeyFromEntry([]byte("garbage")); err == nil {
		t.Error("garbage was accepted")
	}
}

func TestPublicKeyPageErrorsAreNotFatal(t *testing.T) {
	oldVC, oldDir := vc, confDir
	t.Cleanup(func() { vc, confDir = oldVC, oldDir })
	vc = config.New()
	vc.Default("server.key_store.location", "missing.jks")
	confDir = t.TempDir()

	e := echo.New()
	rec := httptest.NewRecorder()
	ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/public_key", nil), rec)

	// the handler must return an error response and not exit the process
	if err := publicKeyPage(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestTestUserPageUnauthenticated(t *testing.T) {
	// nothing in the request's context
	rec := httptest.NewRecorder()
	testuserPage(rec, httptest.NewRequest(http.MethodGet, "/testuser", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if strings.Contains(rec.Body.String(), "Test User Diagnostics") || strings.Contains(rec.Body.String(), "LDAP") {
		t.Errorf("an unauthenticated request got diagnostics: %q", rec.Body.String())
	}

	// an identity that is not authenticated
	user := goidentity.NewUser("someone")
	req := goidentity.AddToHTTPRequestContext(&user, httptest.NewRequest(http.MethodGet, "/testuser", nil))
	rec = httptest.NewRecorder()
	testuserPage(rec, req)
	if rec.Code != http.StatusUnauthorized || strings.Contains(rec.Body.String(), "Test User Diagnostics") {
		t.Errorf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestTestUserPageWithoutADUser(t *testing.T) {
	user := goidentity.NewUser("someone")
	user.SetAuthenticated(true)
	req := goidentity.AddToHTTPRequestContext(&user, httptest.NewRequest(http.MethodGet, "/testuser", nil))
	rec := httptest.NewRecorder()
	testuserPage(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "cannot search LDAP") {
		t.Errorf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestConfigFileIsConsumed(t *testing.T) {
	oldFile, oldDir, oldVerify := cfgFile, confDir, ldapVerify
	t.Cleanup(func() {
		cfgFile, confDir, ldapVerify = oldFile, oldDir, oldVerify
		viper.Reset()
		rootCmd.PersistentFlags().Lookup("dir").Changed = false
		rootCmd.PersistentFlags().Lookup("ldap-verify").Changed = false
	})

	conf := filepath.Join(t.TempDir(), "sso-diag.yaml")
	if err := os.WriteFile(conf, []byte("dir: /some/agent\nldap-verify: true\n"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Run("used when flags are not given", func(t *testing.T) {
		viper.Reset()
		cfgFile, confDir, ldapVerify = conf, ".", false
		initConfig()
		if confDir != "/some/agent" || !ldapVerify {
			t.Errorf("confDir = %q, ldapVerify = %v", confDir, ldapVerify)
		}
	})

	t.Run("flags take precedence", func(t *testing.T) {
		viper.Reset()
		cfgFile, confDir, ldapVerify = conf, "/from/flag", false
		rootCmd.PersistentFlags().Lookup("dir").Changed = true
		rootCmd.PersistentFlags().Lookup("ldap-verify").Changed = true
		initConfig()
		if confDir != "/from/flag" || ldapVerify {
			t.Errorf("confDir = %q, ldapVerify = %v", confDir, ldapVerify)
		}
	})
}

func TestDefaultConfigPathMatchesHelp(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	dirs := defaultConfigDirs()
	if len(dirs) != 1 || dirs[0] != filepath.Join(home, ".config", "geneos") {
		t.Fatalf("config dirs = %v", dirs)
	}
	usage := rootCmd.PersistentFlags().Lookup("conf").Usage
	if !strings.Contains(usage, "$HOME/.config/geneos/sso-diag.yaml") {
		t.Errorf("conf usage = %q", usage)
	}
}

func TestCommandDescriptions(t *testing.T) {
	for _, text := range []string{rootCmd.Short, rootCmd.Long, runCmd.Short, runCmd.Long} {
		if strings.Contains(text, "Cobra") || strings.Contains(text, "brief description") {
			t.Errorf("boilerplate description: %q", text)
		}
	}
	if rootCmd.Version == "" {
		t.Error("no version set")
	}
}
