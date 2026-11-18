package profile

import (
	"slices"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
)

func TestProfileHelpArgs(t *testing.T) {
	for _, test := range []struct {
		line []string
		want []string
	}{
		{nil, nil},
		{[]string{"geneos"}, nil},
		{[]string{"geneos", "profile"}, nil},
		{[]string{"geneos", "help", "profile", "demo"}, []string{"demo"}},
		{[]string{"geneos", "profile", "demo", "--help"}, []string{"demo"}},
		{[]string{"geneos", "--debug", "profiles", "-h", "demo", "other"}, []string{"demo", "other"}},
		{[]string{"geneos", "help", "gateway", "demo"}, nil},
	} {
		if got := profileHelpArgs(test.line); !slices.Equal(got, test.want) {
			t.Errorf("profileHelpArgs(%q) = %q, want %q", test.line, got, test.want)
		}
	}
}

// `geneos add profile NAME` has no flag set to save as options
func TestAddProfileWithoutFlags(t *testing.T) {
	geneostest.NewRoot(t, "", "licd")
	previous := cmd.Profiles
	t.Cleanup(func() { cmd.Profiles = previous })
	cmd.Profiles = config.New(config.WithDefaults([]byte("profiles:\n  nf:\n    licd:\n      - name: nf-member\n"), "yaml"))
	profile.SetExpandOptions(geneos.LOCAL, cmd.Profiles, nil, nil)

	if _, err := instance.Add(geneos.LOCAL, &Component, "nf", instance.Insecure(true)); err != nil {
		t.Fatal(err)
	}
	if members := instance.Instances(geneos.LOCAL, geneos.ParseComponent("licd"), instance.MatchNames("nf-member")); len(members) != 1 {
		t.Errorf("profile members = %v, want one", members)
	}
}
