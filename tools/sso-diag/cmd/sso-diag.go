package cmd

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"

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
)

// SplitUsername takes a name of the form 'domain\user' or 'user@domain'
// and returns the two. realm will be empty of none found
func SplitUsername(in string) (username, realm string) {
	p := strings.SplitN(in, "\\", 2)
	if len(p) == 2 {
		return p[1], p[0]
	}
	p = strings.SplitN(in, "@", 2)
	username = p[0]
	if len(p) == 2 {
		realm = p[1]
	}
	return
}

var vc *config.Config

func start() {
	// workaround issues in hocon package until fixed
	vc = config.New()
	vc.SetDefault("kerberos.krb5_conf", "krb5.conf")
	// ... set default from a defaults map

	ssoCfgFile := filepath.Join(confDir, "conf/sso-agent.conf")
	err := vc.MergeHOCONFile(ssoCfgFile)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	username, realm := SplitUsername(vc.GetString("kerberos.user"))
	if username == "" {
		username, realm = SplitUsername(vc.GetString("ldap.user"))
	}

	krb5conf := vc.GetString("kerberos.krb5_conf")
	if !filepath.IsAbs(krb5conf) {
		krb5conf = filepath.Join(confDir, krb5conf)
	}
	cf, err := krb5config.Load(krb5conf)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	password := vc.GetString("kerberos.password")
	if password == "" {
		password = vc.GetString("ldap.password")
	}

	if realm == "" {
		realm = cf.LibDefaults.DefaultRealm
	}
	c := client.NewWithPassword(username, realm, password, cf, client.DisablePAFXFAST(true))
	err = c.Login()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// get the kvno from a tgt request
	asreq, err := messages.NewASReqForTGT(c.Credentials.Domain(), c.Config, c.Credentials.CName())
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	asrep, err := c.ASExchange(c.Credentials.Domain(), asreq, 0)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	serviceKVNO := asrep.EncPart.KVNO

	// c.Diagnostics(os.Stdout)
	kt := keytab.New()

	for _, e := range cf.LibDefaults.DefaultTktEnctypes {
		if eid := etypeID.EtypeSupported(e); eid != 0 {
			if err := kt.AddEntry(username, realm, password, time.Now(), uint8(serviceKVNO), eid); err != nil {
				log.Fatal().Err(err).Msg("")
			}
		}
	}

	initServer(vc, kt, username)

}

func initServer(vc *config.Config, kt *keytab.Keytab, username string) {
	e := echo.New()
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Info().
				Str("URI", v.URI).
				Int("status", v.Status).
				Msg("request")

			return nil
		},
	}))

	e.GET("/status", statusPage)

	if vc.GetBool("server.key_store.enable_public_key_endpoint") {
		e.GET("/public_key", publicKeyPage)
	}

	e.GET("/testuser", echo.WrapHandler(spnego.SPNEGOKRB5Authenticate(http.HandlerFunc(testuserPage), kt, service.KeytabPrincipal(username))))
	e.GET("/authorize", echo.WrapHandler(spnego.SPNEGOKRB5Authenticate(http.HandlerFunc(authorizePage), kt, service.KeytabPrincipal(username))))

	i := fmt.Sprintf("%s:%d", vc.GetString("server.bind_address"), vc.GetInt("server.port"))

	if !vc.GetBool("server.enable_ssl") {
		e.Logger.Fatal(e.Start(i))
	}

	// this doesn't work, get cert from keystore...
	var cert interface{}
	certstr := config.GetString("api.tls.certificate")
	certpem, _ := pem.Decode([]byte(certstr))
	if certpem == nil {
		cert = certstr
	} else {
		cert = []byte(certstr)
	}

	var key interface{}
	keystr := config.GetString("api.tls.key")
	keypem, _ := pem.Decode([]byte(keystr))
	if keypem == nil {
		key = keystr
	} else {
		key = []byte(keystr)
	}
	e.Logger.Fatal(e.StartTLS(i, cert, key))

}

func loadSSOkey(cf *config.Config) *rsa.PrivateKey {
	ks := cf.GetString("server.key_store.location")
	if !filepath.IsAbs(ks) {
		ks = filepath.Join(confDir, ks)
	}
	pw := []byte(cf.GetString("server.key_store.password"))

	f, err := os.Open(ks)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	defer f.Close()
	k := keystore.New()
	if err := k.Load(f, pw); err != nil {
		log.Fatal().Err(err).Msg("")
	}
	pk, err := k.GetPrivateKeyEntry("ssokey", pw)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	key, err := x509.ParsePKCS8PrivateKey(pk.PrivateKey)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	if r, ok := key.(*rsa.PrivateKey); ok {
		return r
	}
	return nil
}

func loadTLSCert(cf *config.Config) {

}

func statusPage(c echo.Context) error {
	return c.String(http.StatusOK, "status page")
}

func publicKeyPage(c echo.Context) error {
	key := loadSSOkey(vc)

	var pubkey *rsa.PublicKey
	pubkey = &key.PublicKey

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(pubkey),
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

func testuserPage(w http.ResponseWriter, r *http.Request) {
	var user string

	w.Write([]byte("Test User Diagnostics\n\n"))

	// Get a goidentity credentials object from the request's context
	creds := goidentity.FromHTTPRequestContext(r)

	// Check if it indicates it is authenticated
	if creds != nil && creds.Authenticated() {
		log.Printf("cred: %+v\n", creds)
		// Check for Active Directory attributes
		if ADCreds, ok := creds.Attributes()[credentials.AttributeKeyADCredentials]; ok {
			b, _ := json.MarshalIndent(ADCreds, "", "    ")
			fmt.Fprintf(w, "creds: %s\n\n", b)
			Creds := new(credentials.ADCredentials)
			json.Unmarshal(b, Creds)
			user = Creds.EffectiveName
		}
	} else {
		// Not authenticated user
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Authentication failed")
	}

	var tlsConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	l, err := ldap.DialURL(vc.GetString("ldap.location"), ldap.DialWithTLSConfig(tlsConfig))
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	if err = l.Bind(vc.GetString("ldap.user"), vc.GetString("ldap.password")); err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	qf := vc.GetString("ldap.users.query_filter")
	if qf == "" {
		qf = fmt.Sprintf("(objectCategory=person)(objectClass=%s)", vc.GetString("ldap.users.class"))
	}

	query := fmt.Sprintf("(&%s(%s=%s))", qf, vc.GetString("ldap.fields.user"), user)
	fmt.Fprintf(w, "LDAP Query: %s\n", query)
	log.Printf("LDAP query: %s", query)
	search := ldap.NewSearchRequest(vc.GetString("ldap.base"), ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		query, vc.GetStringSlice("ldap.fields"), []ldap.Control{})

	result, err := l.Search(search)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	if len(result.Entries) > 0 {
		for _, e := range result.Entries {
			fmt.Fprintf(w, "\nDN: %s\n", e.DN)
			for _, a := range e.Attributes {
				b, _ := json.MarshalIndent(a, "  ", "    ")
				fmt.Fprintf(w, "%s\n", string(b))
			}
		}
	}
	fmt.Fprintln(w, "end of data")
}
