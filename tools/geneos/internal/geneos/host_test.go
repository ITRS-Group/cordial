package geneos

import (
	"errors"
	"slices"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// setTestHosts replaces the global LOCAL, ALL and LOCALHOST for the test
// and restores them when it is finished. The local host uses root as
// the Geneos directory.
func setTestHosts(t *testing.T, root string) {
	t.Helper()
	previousLocal, previousAll, previousName := LOCAL, ALL, LOCALHOST
	t.Cleanup(func() {
		LOCAL, ALL, LOCALHOST = previousLocal, previousAll, previousName
	})
	LOCALHOST = "localhost"
	LOCAL = &Host{Host: host.NewLocal(), Config: config.New(config.KeyDelimiter("::")), loaded: true}
	config.Set(LOCAL.Config, "name", LOCALHOST)
	config.Set(LOCAL.Config, cordial.ExecutableName(), root)
	ALL = &Host{Host: host.NewLocal(), Config: config.New(config.KeyDelimiter("::")), loaded: true}
	config.Set(ALL.Config, "name", ALLHOSTS)
}

func TestUnknownHost(t *testing.T) {
	setTestHosts(t, t.TempDir())

	if got := GetHost(""); got != nil {
		t.Errorf("GetHost(\"\") = %v, want nil (no host given)", got)
	}
	if got := GetHost("localhost"); got != LOCAL {
		t.Errorf("GetHost(\"localhost\") = %v, want LOCAL", got)
	}

	unknown := GetHost("no-such-host.example.invalid")
	if unknown == nil || unknown != UNKNOWN {
		t.Fatalf("GetHost(unknown name) = %v, want the non-nil UNKNOWN sentinel", unknown)
	}
	if !unknown.IsUnknown() || !(*Host)(nil).IsUnknown() || LOCAL.IsUnknown() {
		t.Error("IsUnknown() is wrong for UNKNOWN, nil or LOCAL")
	}
	if unknown.Exists() {
		t.Error("UNKNOWN host exists")
	}
	if unknown.String() != "unknown" {
		t.Errorf("UNKNOWN.String() = %q", unknown.String())
	}

	t.Run("OrList", func(t *testing.T) {
		if got := slices.Collect(UNKNOWN.OrList()); len(got) != 0 {
			t.Errorf("UNKNOWN.OrList() = %v, want nothing (not all hosts)", got)
		}
		if got := slices.Collect(UNKNOWN.OrList(LOCAL)); len(got) != 0 {
			t.Errorf("UNKNOWN.OrList(LOCAL) = %v, want nothing", got)
		}
		if got := slices.Collect((*Host)(nil).OrList(UNKNOWN)); len(got) != 0 {
			t.Errorf("nil.OrList(UNKNOWN) = %v, want nothing", got)
		}
		// nil still means all hosts
		if got := slices.Collect((*Host)(nil).OrList()); !slices.Contains(got, LOCAL) {
			t.Errorf("nil.OrList() = %v, want to include LOCAL", got)
		}
	})

	t.Run("Match", func(t *testing.T) {
		if got := slices.Collect(Match("no-such-host.example.invalid")); len(got) != 0 {
			t.Errorf("Match(unknown) = %v, want nothing", got)
		}
		if got := slices.Collect(Match("")); len(got) != 0 {
			t.Errorf("Match(\"\") = %v, want nothing", got)
		}
		if got := slices.Collect(Match("localhost")); !slices.Equal(got, []*Host{LOCAL}) {
			t.Errorf("Match(localhost) = %v", got)
		}
	})

	t.Run("MakeDirs", func(t *testing.T) {
		for name, h := range map[string]*Host{"unknown": UNKNOWN, "nil": nil, "all": ALL} {
			if err := RootComponent.MakeDirs(h); !errors.Is(err, ErrInvalidArgs) {
				t.Errorf("MakeDirs(%s) error = %v, want ErrInvalidArgs", name, err)
			}
		}
	})
}

// TestNewHostLocalAlias checks that "localhost" is always the local
// host, even when the local host name is different, rather than a new
// SSH remote called "localhost".
func TestNewHostLocalAlias(t *testing.T) {
	setTestHosts(t, t.TempDir())
	LOCALHOST = "testhost"
	config.Set(LOCAL.Config, "name", LOCALHOST)

	if got := NewHost("localhost"); got != LOCAL {
		t.Errorf("NewHost(\"localhost\") = %v, want LOCAL", got)
	}
	if got := NewHost(LOCALHOST); got != LOCAL {
		t.Errorf("NewHost(%q) = %v, want LOCAL", LOCALHOST, got)
	}
	if _, ok := hosts.Load("localhost"); ok {
		t.Error("a remote host called localhost was registered")
	}
}
