package cmd

import (
	"strings"
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func TestProfileExpandHost(t *testing.T) {
	previous := geneos.LOCAL
	t.Cleanup(func() { geneos.LOCAL = previous })
	geneos.LOCAL = geneos.NewHost("localhost")

	for _, name := range []string{"", geneos.ALLHOSTS, "localhost"} {
		if got := profileExpandHost(name); got != geneos.LOCAL {
			t.Errorf("profileExpandHost(%q) = %v, want local host", name, got)
		}
	}
}

func TestCommandFlagNormalization(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "license", want: "licence"},
		{input: "attr", want: "attribute"},
		{input: "var", want: "variable"},
		{input: "output", want: "output"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			if got := string(cmdNormalizeFunc(nil, test.input)); got != test.want {
				t.Errorf("cmdNormalizeFunc(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
	if Cmd.PersistentFlags().Lookup("profile") == nil {
		t.Error("global --profile alias is not registered")
	}
	if Cmd.PersistentFlags().Lookup("profiles") == nil {
		t.Error("global --profiles flag is not registered")
	}
}

func TestUnknownHostFlagIsRejected(t *testing.T) {
	cmdtest.Env(t)

	for _, args := range [][]string{
		{"start", "-H", "typo"},
		{"start", "gateway", "-H", "typo"},
		{"add", "gateway", "x", "-H", "typo"},
		{"backup", "gateway", "-H", "typo"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			res := cmdtest.Run(t, Cmd, args...)
			if res.Err == nil || !strings.Contains(res.Err.Error(), `unknown host "typo"`) {
				t.Fatalf("error = %v, want unknown host error (stdout %q, stderr %q)", res.Err, res.Stdout, res.Stderr)
			}
		})
	}
}

func TestValidateHostFlag(t *testing.T) {
	cmdtest.Env(t)
	geneos.InitHosts("geneos")

	for _, test := range []struct {
		name    string
		wantErr bool
	}{
		{name: ""},
		{name: geneos.ALLHOSTS},
		{name: geneos.LOCALHOST},
		{name: "typo", wantErr: true},
	} {
		if err := validateHostFlag(test.name); (err != nil) != test.wantErr {
			t.Errorf("validateHostFlag(%q) = %v, want error %v", test.name, err, test.wantErr)
		}
	}
}
