package snow

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestEscapeQuery(t *testing.T) {
	for in, want := range map[string]string{
		"":                       "",
		"plain":                  "plain",
		"a^b":                    "a^^b",
		"^":                      "^^",
		"^^":                     "^^^^",
		"x^ORactive=false":       "x^^ORactive=false",
		"x^NQnumber=1":           "x^^NQnumber=1",
		`back\slash and "quote"`: `back\slash and "quote"`,
	} {
		if got := EscapeQuery(in); got != want {
			t.Errorf("EscapeQuery(%q) = %q, want %q", in, got, want)
		}
	}

	in := map[string]string{"a": "1^2", "b": "plain"}
	got := EscapeQueryValues(in)
	if got["a"] != "1^^2" || got["b"] != "plain" || in["a"] != "1^2" {
		t.Errorf("EscapeQueryValues() = %v, input is now %v", got, in)
	}
	if EscapeQueryValues(nil) != nil {
		t.Error("EscapeQueryValues(nil) is not nil")
	}
}

// TestLiveEncodedQueryEscaping checks, against a real instance, that a
// `^` in a value has to be doubled. It only reads, using the aggregate
// API, and is only run if CORDIAL_SNOW_LIVE names a configuration file
// with servicenow.url, servicenow.username and servicenow.password, such
// as the one for the proxy, and the credentials may be encoded.
//
//	CORDIAL_SNOW_LIVE=~/.config/geneos/servicenow2.proxy.yaml go test -run Live ./pkg/ims/snow
func TestLiveEncodedQueryEscaping(t *testing.T) {
	path := os.Getenv("CORDIAL_SNOW_LIVE")
	if path == "" {
		t.Skip("CORDIAL_SNOW_LIVE is not set")
	}
	cf, err := config.Read("live", config.FilePath(path), config.UseDefaults(false), config.MustExist())
	if err != nil {
		t.Fatal(err)
	}
	base := strings.TrimRight(config.Get[string](cf, "servicenow.url"), "/")
	user, pass := config.Get[string](cf, "servicenow.username"), config.Get[string](cf, "servicenow.password")
	if base == "" || user == "" {
		t.Fatal("servicenow.url and servicenow.username are required")
	}
	client := &http.Client{Timeout: 60 * time.Second}

	count := func(query string) string {
		t.Helper()
		u := base + "/api/now/stats/sys_script?" + url.Values{"sysparm_count": {"true"}, "sysparm_query": {query}}.Encode()
		req, _ := http.NewRequest(http.MethodGet, u, nil)
		req.SetBasicAuth(user, pass)
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("query %q: HTTP %d", query, resp.StatusCode)
		}
		var r struct {
			Result struct {
				Stats struct {
					Count string `json:"count"`
				} `json:"stats"`
			} `json:"result"`
		}
		if err := json.Unmarshal(b, &r); err != nil {
			t.Fatal(err)
		}
		return r.Result.Stats.Count
	}

	// business rule scripts commonly contain encoded queries such as
	// "active=true^...", so some of them contain a literal `^`
	literal := count("scriptLIKE" + EscapeQuery("true^"))
	if literal == "0" {
		t.Skip("no script on this instance contains a literal ^ to find")
	}
	if truncated := count("scriptLIKEtrue^"); truncated == literal {
		t.Errorf("an unescaped ^ found the same %s records as the escaped one, so it was not treated as a separator", literal)
	}
	if mixed := count("scriptLIKE" + EscapeQuery("true^") + "^active=true"); mixed != literal {
		t.Errorf("a condition after an escaped ^ found %s records, want %s", mixed, literal)
	}
}
