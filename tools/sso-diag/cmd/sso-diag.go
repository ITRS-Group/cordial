package cmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/gurkankaymak/hocon"
	"github.com/rs/zerolog/log"

	"github.com/jcmturner/goidentity/v6"
	"github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/iana/etypeID"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/jcmturner/gokrb5/v8/messages"
	"github.com/jcmturner/gokrb5/v8/service"
	"github.com/jcmturner/gokrb5/v8/spnego"
)

var username = "geneossso"
var realm = "GWH.COM"
var spn = "HTTP/ec2-34-245-48-133.eu-west-1.compute.amazonaws.com"
var ldapurl string
var tlsConfig = &tls.Config{
	InsecureSkipVerify: true,
}

func start() {
	hc, err := hocon.ParseResource("test.conf")
	if err != nil {
		panic(err)
	}
	log.Debug().Msg(hc.String())

	cf, err := config.Load("krb5.conf")
	if err != nil {
		panic(err)
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

	c := client.NewWithPassword(username, realm, os.Getenv("PW"), cf, client.DisablePAFXFAST(true))
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
			if err := kt.AddEntry(username, realm, os.Getenv("PW"), time.Now(), uint8(serviceKVNO), eid); err != nil {
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

	h := http.HandlerFunc(apphandler)
	http.Handle("/testuser", spnego.SPNEGOKRB5Authenticate(h, kt, service.KeytabPrincipal(username)))
	log.Fatal().Err(http.ListenAndServe(":8080", nil)).Msg("")

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
	if err = l.Bind(fmt.Sprintf("%s@%s", username, "gwh.com"), os.Getenv("PW")); err != nil {
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
