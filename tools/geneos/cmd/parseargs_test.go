package cmd

import (
	"context"
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
