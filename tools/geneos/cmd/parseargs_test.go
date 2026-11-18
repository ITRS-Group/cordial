package cmd

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestParseArgsRequiresCommandContext(t *testing.T) {
	command := &cobra.Command{
		Use: "test",
		Annotations: map[string]string{
			CmdAuditCommand: "never",
			CmdAuditActions: "never",
		},
	}
	if err := ParseArgs(command, nil); err == nil || !strings.Contains(err.Error(), "command context not found") {
		t.Fatalf("ParseArgs() error = %v, want missing-context error", err)
	}
}

func TestFetchArgsRejectsUnexpectedParameters(t *testing.T) {
	command := &cobra.Command{
		Use: "test",
		Annotations: map[string]string{
			CmdNonInstanceArgsError: "true",
		},
	}
	command.SetContext(context.WithValue(context.Background(), CmdKey, &CmdValType{
		params: []string{"unexpected"},
	}))

	_, _, params, err := FetchArgs(command)
	if err == nil || !strings.Contains(err.Error(), "unexpected parameters") {
		t.Fatalf("FetchArgs() error = %v, want unexpected-parameter error", err)
	}
	if len(params) != 1 || params[0] != "unexpected" {
		t.Fatalf("FetchArgs() params = %v, want [unexpected]", params)
	}
}

func TestRedactArgs(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{"none", []string{"geneos", "set", "gateway", "gw1", "-e", "A=B"}, []string{"geneos", "set", "gateway", "gw1", "-e", "A=B"}},
		{"secure env", []string{"geneos", "set", "-E", "NAME=VALUE", "gw1"}, []string{"geneos", "set", "-E", "NAME=[REDACTED]", "gw1"}},
		{"secure env long", []string{"set", "--secureenv", "NAME=VALUE", "--secureenv=N2=V2"}, []string{"set", "--secureenv", "NAME=[REDACTED]", "--secureenv=N2=[REDACTED]"}},
		{"secure env attached", []string{"set", "-ENAME=VALUE", "-E=N2=V2"}, []string{"set", "-ENAME=[REDACTED]", "-E=N2=[REDACTED]"}},
		{"secure param", []string{"set", "-s", "NAME=VALUE", "-s", "PROMPTED"}, []string{"set", "-s", "NAME=[REDACTED]", "-s", "PROMPTED"}},
		{"secret variable", []string{"set", "-v", "secret:foo=VALUE", "-v", "string:bar=VALUE"}, []string{"set", "-v", "secret:foo=[REDACTED]", "-v", "string:bar=VALUE"}},
		{"after separator", []string{"set", "--", "-E", "NAME=VALUE"}, []string{"set", "--", "-E", "NAME=VALUE"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := slices.Clone(test.args)
			if got := redactArgs(setCmd, test.args); !slices.Equal(got, test.want) {
				t.Errorf("redactArgs() = %q, want %q", got, test.want)
			}
			if !slices.Equal(original, test.args) {
				t.Errorf("redactArgs() changed its argument: %q", test.args)
			}
		})
	}
}
