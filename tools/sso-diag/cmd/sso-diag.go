package cmd

import (
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/jcmturner/goidentity/v6"
	"github.com/jcmturner/gokrb5/v8/client"
	krb5config "github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/iana/etypeID"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/jcmturner/gokrb5/v8/messages"
	"github.com/jcmturner/gokrb5/v8/service"
	"github.com/jcmturner/gokrb5/v8/spnego"

	"github.com/itrs-group/cordial/integrations/servicenow/snow"
	"github.com/itrs-group/cordial/pkg/config"
)

// var username = "geneossso"
// var realm = "GWH.COM"
// var spn = "HTTP/ec2-34-245-48-133.eu-west-1.compute.amazonaws.com"
var ldapurl string

var tlsConfig = &tls.Config{
	InsecureSkipVerify: true,
}

// curl -v --negotiate -u : "http://endpoint:8080/authorize?response_type=token&client_id=active_console&state=XXXXXX"

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

	err := vc.MergeHOCONFile(ssoCfgFile)
	if err != nil {
		panic(err)
	}

	username, realm := SplitUsername(vc.GetString("kerberos.user"))
	if username == "" {
		username, realm = SplitUsername(vc.GetString("ldap.user"))
	}

	cf, err := krb5config.Load(vc.GetString("kerberos.krb5_conf"))
	if err != nil {
		panic(err)
	}

	password := vc.GetString("kerberos.password")
	if password == "" {
		password = vc.GetString("ldap.password")
	}

	n, kdcs, err := cf.GetKDCs(realm, false)
	if n == 0 || err != nil {
		panic(err)
	}
	s := strings.SplitN(kdcs[len(kdcs)], ":", 2)
	if len(s) != 2 {
		panic("kdc in wrong format")
	}
	ldapurl = fmt.Sprintf("ldaps://%s", s[0])
	fmt.Println("ldap url:", ldapurl)

	c := client.NewWithPassword(username, realm, password, cf, client.DisablePAFXFAST(true))
	err = c.Login()
	if err != nil {
		panic(err)
	}

	// get the kvno from a tgt request
	asreq, err := messages.NewASReqForTGT(c.Credentials.Domain(), c.Config, c.Credentials.CName())
	if err != nil {
		panic(err)
	}
	asrep, err := c.ASExchange(c.Credentials.Domain(), asreq, 0)
	if err != nil {
		panic(err)
	}
	serviceKVNO := asrep.EncPart.KVNO

	// c.Diagnostics(os.Stdout)

	kt := keytab.New()

	for _, e := range cf.LibDefaults.DefaultTktEnctypes {
		if eid := etypeID.EtypeSupported(e); eid != 0 {
			if err := kt.AddEntry(username, realm, password, time.Now(), uint8(serviceKVNO), eid); err != nil {
				panic(err)
			}
		}
	}

	// w, err := os.Create("key.tab")
	// if err != nil {
	// 	panic(err)
	// }
	// kt.Write(w)
	// w.Close()

	// fmt.Println(kt.JSON())

	// h := http.HandlerFunc(apphandler)
	// http.Handle("/testuser", spnego.SPNEGOKRB5Authenticate(h, kt, service.KeytabPrincipal(username)))
	// log.Fatal().Err(http.ListenAndServe(":8080", nil)).Msg("")

	initServer(vc, kt, username)

}

// timestamp the start of the request
func Timestamp() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("starttime", time.Now())
			return next(c)
		}
	}
}

func bodyDumpLog(c echo.Context, reqBody, resBody []byte) {
	var reqMethod string
	var resStatus int

	// request and response object
	req := c.Request()
	res := c.Response()
	// rendering variables for response status and request method
	resStatus = res.Status
	reqMethod = req.Method

	// print formatting the custom logger tailored for DEVELOPMENT environment
	var result map[string]string
	var message string

	if err := json.Unmarshal(resBody, &result); err == nil {
		if result["message"] != "" {
			message = result["message"]
		} else if result["action"] == "Failed" {
			message = fmt.Sprintf("Failed to create event for %s", result["host"])
		} else {
			message = fmt.Sprintf("%s %s %s", result["event_type"], result["number"], result["action"])
		}
	}

	bytes_in := req.Header.Get(echo.HeaderContentLength)
	if bytes_in == "" {
		bytes_in = "0"
	}
	starttime := c.Get("starttime").(time.Time)
	latency := time.Since(starttime)
	latency = latency.Round(time.Millisecond)

	fmt.Printf("%v %s %s %3d %s/%d %v %s %s %s %q\n",
		time.Now().Format(time.RFC3339),     // TIMESTAMP for route access
		vc.GetString("servicenow.instance"), // name of server (APP) with the environment
		req.Proto,                           // protocol
		resStatus,                           // response status
		// stats here
		bytes_in,
		res.Size,
		latency,
		c.RealIP(), // client IP
		reqMethod,  // request method
		req.URL,    // request URI (path)
		message,
	)
}

func initServer(vc *config.Config, kt *keytab.Keytab, username string) {
	// Initialization of go-echo server
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true

	// pass configuration into handlers
	// as per https://echo.labstack.com/guide/context/
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &snow.RouterContext{Context: c, Conf: vc}
			return next(cc)
		}
	})
	e.Use(Timestamp())
	e.Use(middleware.BodyDump(bodyDumpLog))
	// e.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
	// 	return key == vc.GetString("api.apikey"), nil
	// }))

	// list of endpoint routes
	// APIRoute := e.Group("/api")
	// grouping routes for version 1.0 API
	// v1route := APIRoute.Group("/v1")

	// Get Endpoints
	h := echo.WrapHandler(spnego.SPNEGOKRB5Authenticate(http.HandlerFunc(apphandler), kt, service.KeytabPrincipal(username)))
	e.GET("/testuser", h)

	// Put Endpoints
	// v1route.POST("/incident", snow.AcceptEvent)

	i := fmt.Sprintf("%s:%d", vc.GetString("server.bind_address"), vc.GetInt("server.port"))

	// firing up the server
	if !vc.GetBool("api.tls.enabled") {
		e.Logger.Fatal(e.Start(i))
	} else if vc.GetBool("api.tls.enabled") {
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
}

func apphandler(w http.ResponseWriter, r *http.Request) {
	var user string

	w.Write([]byte("welcome!\n\n"))

	// Get a goidentity credentials object from the request's context
	creds := goidentity.FromHTTPRequestContext(r)

	// Check if it indicates it is authenticated
	if creds != nil && creds.Authenticated() {
		fmt.Fprintf(w, "cred: %+v\n", creds)
		// Check for Active Directory attributes
		if ADCreds, ok := creds.Attributes()[credentials.AttributeKeyADCredentials]; ok {
			b, _ := json.MarshalIndent(ADCreds, "", "    ")
			fmt.Fprintf(w, "creds: %s\n", b)
			Creds := new(credentials.ADCredentials)
			json.Unmarshal(b, Creds)
			user = Creds.EffectiveName
		}
	} else {
		// Not authenticated user
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Authentication failed")
	}

	fmt.Fprintf(w, "dialing ldap on %s\n", ldapurl)

	l, err := ldap.DialURL(ldapurl, ldap.DialWithTLSConfig(tlsConfig))
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	if err = l.Bind(vc.GetString("ldap.user"), vc.GetString("ldap.password")); err != nil {
		fmt.Fprint(w, err)
		return
	}
	fmt.Fprintln(w, "here")

	search := ldap.NewSearchRequest("DC=GWH,DC=COM", ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(sAMAccountName=%s)", user), []string{"memberOf"}, []ldap.Control{})

	result, err := l.Search(search)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	if len(result.Entries) > 0 {
		for _, e := range result.Entries {
			fmt.Fprintln(w, e.DN)
			for _, a := range e.Attributes {
				b, _ := json.MarshalIndent(a, "  ", "    ")
				fmt.Fprintf(w, "%s\n", string(b))
			}
		}
	}
	fmt.Fprintln(w, "end of data")
}
