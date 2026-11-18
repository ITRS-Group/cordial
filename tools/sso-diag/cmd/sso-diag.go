package cmd

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/pavlo-v-chernykh/keystore-go/v4"

	"github.com/jcmturner/goidentity/v6"
	"github.com/jcmturner/gokrb5/v8/client"
	krb5config "github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/iana/etypeID"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/jcmturner/gokrb5/v8/messages"
	"github.com/jcmturner/gokrb5/v8/service"
	"github.com/jcmturner/gokrb5/v8/spnego"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
)

var log = logger.Logger

// SplitUsername takes a name of the form 'domain\user' or 'user@domain'
// and returns the two. realm will be empty of none found
func SplitUsername(in string) (username, realm string) {
	var found bool
	username, realm, found = strings.Cut(in, "\\")
	if found {
		return realm, username
	}
	username, realm, _ = strings.Cut(in, "@")
	return username, realm
}

var vc *config.Config

// start reads the SSO Agent configuration, logs in to Kerberos and runs
// the diagnostic server until it fails
func start() (err error) {
	// workaround issues in hocon package until fixed
	vc = config.New()
	vc.Default("kerberos.krb5_conf", "krb5.conf")
	// ... set default from a defaults map

	ssoCfgFile := filepath.Join(confDir, "conf", "sso-agent.conf")
	err = vc.MergeHOCONFile(ssoCfgFile)
	if err != nil {
		return fmt.Errorf("failed to merge HOCON file: %w", err)
	}

	username, realm := SplitUsername(config.Get[string](vc, "kerberos.user"))
	if username == "" {
		username, realm = SplitUsername(config.Get[string](vc, "ldap.user"))
	}

	krb5conf := config.Get[string](vc, "kerberos.krb5_conf")
	if !filepath.IsAbs(krb5conf) {
		krb5conf = filepath.Join(confDir, krb5conf)
	}
	cf, err := krb5config.Load(krb5conf)
	if err != nil {
		return fmt.Errorf("failed to load krb5 configuration: %w", err)
	}

	password := config.Get[string](vc, "kerberos.password")
	if password == "" {
		password = config.Get[string](vc, "ldap.password")
	}

	if realm == "" {
		realm = cf.LibDefaults.DefaultRealm
	}
	c := client.NewWithPassword(username, realm, password, cf, client.DisablePAFXFAST(true))
	err = c.Login()
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	// get the kvno from a tgt request
	asreq, err := messages.NewASReqForTGT(c.Credentials.Domain(), c.Config, c.Credentials.CName())
	if err != nil {
		return fmt.Errorf("failed to create AS request for TGT: %w", err)
	}
	asrep, err := c.ASExchange(c.Credentials.Domain(), asreq, 0)
	if err != nil {
		return fmt.Errorf("failed to exchange AS request for TGT: %w", err)
	}

	// c.Diagnostics(os.Stdout)
	kt, err := buildKeytab(username, realm, password, asrep.EncPart.KVNO, cf.LibDefaults.DefaultTktEnctypes)
	if err != nil {
		return err
	}

	return initServer(vc, kt, username)
}

// buildKeytab returns a keytab with an entry for each supported
// encryption type in enctypes. The keytab format only supports a
// single byte key version number, so larger values are an error.
func buildKeytab(username, realm, password string, kvno int, enctypes []string) (kt *keytab.Keytab, err error) {
	if kvno < 0 || kvno > math.MaxUint8 {
		return nil, fmt.Errorf("key version number %d cannot be stored in a keytab entry (maximum %d)", kvno, math.MaxUint8)
	}

	kt = keytab.New()
	for _, e := range enctypes {
		if eid := etypeID.EtypeSupported(e); eid != 0 {
			if err = kt.AddEntry(username, realm, password, time.Now(), uint8(kvno), eid); err != nil {
				return nil, fmt.Errorf("failed to add entry to keytab: %w", err)
			}
		}
	}
	return
}

func initServer(vc *config.Config, kt *keytab.Keytab, username string) error {
	e := echo.New()
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Info(
				"request",
				slog.String("URI", v.URI),
				slog.Int("status", v.Status),
			)
			return nil
		},
	}))

	e.GET("/status", statusPage)

	if config.Get[bool](vc, "server.key_store.enable_public_key_endpoint") {
		e.GET("/public_key", publicKeyPage)
	}

	e.GET("/testuser", echo.WrapHandler(spnego.SPNEGOKRB5Authenticate(http.HandlerFunc(testuserPage), kt, service.KeytabPrincipal(username))))
	e.GET("/authorize", echo.WrapHandler(spnego.SPNEGOKRB5Authenticate(http.HandlerFunc(authorizePage), kt, service.KeytabPrincipal(username))))

	i := fmt.Sprintf("%s:%d", config.Get[string](vc, "server.bind_address"), config.Get[int](vc, "server.port"))

	if !config.Get[bool](vc, "server.enable_ssl") {
		return e.Start(i)
	}

	// the certificate and key are not yet taken from the key store, so
	// they have to be given using these settings, as PEM or file paths
	certstr := config.Get[string](vc, "api.tls.certificate")
	keystr := config.Get[string](vc, "api.tls.key")
	if certstr == "" || keystr == "" {
		return errors.New("server.enable_ssl is set but api.tls.certificate and api.tls.key are not: reading them from the key store is not supported")
	}

	var cert, key any = certstr, keystr
	if certpem, _ := pem.Decode([]byte(certstr)); certpem != nil {
		cert = []byte(certstr)
	}
	if keypem, _ := pem.Decode([]byte(keystr)); keypem != nil {
		key = []byte(keystr)
	}
	return e.StartTLS(i, cert, key)
}

// privateKeyFromEntry parses a PKCS8 private key, which must be an RSA key
func privateKeyFromEntry(der []byte) (*rsa.PrivateKey, error) {
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
	}
	r, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is a %T, not an RSA key", key)
	}
	return r, nil
}

func loadSSOkey(cf *config.Config) (*rsa.PrivateKey, error) {
	ks := config.Get[string](cf, "server.key_store.location")
	if !filepath.IsAbs(ks) {
		ks = filepath.Join(confDir, ks)
	}
	pw := []byte(config.Get[string](cf, "server.key_store.password"))

	f, err := os.Open(ks)
	if err != nil {
		return nil, fmt.Errorf("failed to open key store: %w", err)
	}
	defer f.Close()
	k := keystore.New()
	if err := k.Load(f, pw); err != nil {
		return nil, fmt.Errorf("failed to load key store: %w", err)
	}
	pk, err := k.GetPrivateKeyEntry("ssokey", pw)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key entry: %w", err)
	}

	return privateKeyFromEntry(pk.PrivateKey)
}

// TODO: Revisit unused declaration loadTLSCert; restore it only when needed.
// func loadTLSCert(cf *config.Config) {
//
// }

func statusPage(c echo.Context) error {
	return c.String(http.StatusOK, "status page")
}

func publicKeyPage(c echo.Context) error {
	key, err := loadSSOkey(vc)
	if err != nil {
		log.Error("failed to load SSO key", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "unable to load the SSO key")
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&key.PublicKey),
	})
	return c.String(http.StatusOK, string(keyPEM))
}

func authorizePage(w http.ResponseWriter, r *http.Request) {
	// curl -v --negotiate -u : "http://endpoint:8080/authorize?response_type=token&client_id=active_console&state=XXXXXX"
	q := r.URL.Query()
	resp := q.Get("response_type")
	if resp != "token" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	client_id := q.Get("client_id")
	if client_id != "active_console" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// state := q.Get("state")

	// pk := loadSSOkey(vc)
	// creds := goidentity.FromHTTPRequestContext(r)

	// token := jwt.NewWithClaims(jwt.SigningMethodPS256.SigningMethodRSA, jwt.MapClaims{
	// 	"": "xx",
	// })

}

// ldapTLSConfig returns the TLS settings for connecting to the LDAP
// server, which do not verify the server certificate unless the user
// has asked for it with --ldap-verify
func ldapTLSConfig() *tls.Config {
	return &tls.Config{
		// TODO(maintainer): verify the LDAP server certificate by default
		InsecureSkipVerify: !ldapVerify,
	}
}

// userFilter builds the LDAP search filter for user, which has special
// characters escaped, using the configured base query filter and the
// name of the user field
func userFilter(queryFilter, userField, user string) string {
	return fmt.Sprintf("(&%s(%s=%s))", queryFilter, userField, ldap.EscapeFilter(user))
}

func testuserPage(w http.ResponseWriter, r *http.Request) {
	var user string

	// Get a goidentity credentials object from the request's context
	creds := goidentity.FromHTTPRequestContext(r)

	// Check if it indicates it is authenticated
	if creds == nil || !creds.Authenticated() {
		// Not authenticated user
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Authentication failed")
		return
	}

	w.Write([]byte("Test User Diagnostics\n\n"))

	log.Debug("cred", slog.Any("cred", creds))
	// Check for Active Directory attributes
	if ADCreds, ok := creds.Attributes()[credentials.AttributeKeyADCredentials]; ok {
		b, _ := json.MarshalIndent(ADCreds, "", "    ")
		fmt.Fprintf(w, "creds: %s\n\n", b)
		Creds := new(credentials.ADCredentials)
		json.Unmarshal(b, Creds)
		user = Creds.EffectiveName
	}
	if user == "" {
		fmt.Fprintln(w, "no Active Directory user name in the credentials, cannot search LDAP")
		return
	}

	l, err := ldap.DialURL(config.Get[string](vc, "ldap.location"), ldap.DialWithTLSConfig(ldapTLSConfig()))
	if err != nil {
		log.Error("Failed to dial LDAP URL", slog.Any("error", err))
		fmt.Fprintf(w, "failed to connect to LDAP: %s\n", err)
		return
	}
	defer l.Close()

	if err = l.Bind(config.Get[string](vc, "ldap.user"), config.Get[string](vc, "ldap.password")); err != nil {
		log.Error("Failed to bind LDAP user", slog.Any("error", err))
		fmt.Fprintf(w, "failed to bind to LDAP: %s\n", err)
		return
	}

	qf := config.Get[string](vc, "ldap.users.query_filter")
	if qf == "" {
		qf = fmt.Sprintf("(objectCategory=person)(objectClass=%s)", config.Get[string](vc, "ldap.users.class"))
	}

	query := userFilter(qf, config.Get[string](vc, "ldap.fields.user"), user)
	fmt.Fprintf(w, "LDAP Query: %s\n", query)
	log.Debug("LDAP query", slog.String("query", query))
	search := ldap.NewSearchRequest(config.Get[string](vc, "ldap.base"), ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		query, config.Get[[]string](vc, "ldap.fields"), []ldap.Control{})

	result, err := l.Search(search)
	if err != nil {
		log.Error("Failed to search LDAP", slog.Any("error", err))
		fmt.Fprintf(w, "failed to search LDAP: %s\n", err)
		return
	}

	for _, e := range result.Entries {
		fmt.Fprintf(w, "\nDN: %s\n", e.DN)
		for _, a := range e.Attributes {
			b, _ := json.MarshalIndent(a, "  ", "    ")
			fmt.Fprintf(w, "%s\n", string(b))
		}
	}
	fmt.Fprintln(w, "end of data")
}
