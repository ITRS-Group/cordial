package geneos

import (
	"strings"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

func TestSiteNameAndValidation(t *testing.T) {
	previous := config.Get[string](config.Global(), "site")
	t.Cleanup(func() { config.Set(config.Global(), "site", previous) })
	for _, test := range []struct {
		site, want string
		valid      bool
	}{
		{"", "local", true},
		{"Acme Site", "Acme_Site", true},
		{"../outside", "", false},
		{".", "", false},
		{"config", "", false},
		{"Config", "", false},
		{"include", "", false},
		{"includes", "", false},
		{"gateways", "", false},
		{"shared", "", false},
		{"templates", "", false},
		{"gateway_shared", "", false},
		{"GATEWAY SHARED", "", false},
		{"gateway_config", "", false},
		{"config2", "config2", true},
		{"bad\nname", "", false},
	} {
		t.Run(test.site, func(t *testing.T) {
			err := ValidateSite(test.site)
			if (err == nil) != test.valid {
				t.Fatalf("ValidateSite(%q) = %v", test.site, err)
			}
			if test.valid {
				config.Set(config.Global(), "site", test.site)
				if got := SiteName(); got != test.want {
					t.Errorf("SiteName() = %q, want %q", got, test.want)
				}
			}
		})
	}
}

func TestSetInitialSitePersistsWithoutReplacing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	previous := config.Get[string](config.Global(), "site")
	config.Set(config.Global(), "site", "")
	t.Cleanup(func() { config.Set(config.Global(), "site", previous) })

	if err := SetInitialSite("config"); err == nil {
		t.Fatal("reserved initial site was accepted")
	}
	if got := config.Get[string](config.Global(), "site"); got != "" {
		t.Fatalf("rejected site changed configuration to %q", got)
	}
	if err := SetInitialSite("Acme Site"); err != nil {
		t.Fatal(err)
	}
	saved, err := config.Read(cordial.ExecutableName(), config.SkipSystemDir(), config.SkipWorkingDir(), config.MustExist())
	if err != nil {
		t.Fatal(err)
	}
	if got := config.Get[string](saved, "site"); got != "Acme Site" {
		t.Errorf("saved site = %q", got)
	}
	if err := SetInitialSite("Acme Site"); err != nil {
		t.Fatal(err)
	}
	if err := SetInitialSite("other"); err == nil || !strings.Contains(err.Error(), "already set") {
		t.Errorf("changing initial site error = %v", err)
	}
}
