package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func TestAddReturnsErrorBeforeFollowingLogs(t *testing.T) {
	previousLogs, previousParams := addCmdLogs, addCmdExtras.Params
	t.Cleanup(func() {
		addCmdLogs, addCmdExtras.Params = previousLogs, previousParams
	})

	for _, logs := range []bool{false, true} {
		name := "without log"
		if logs {
			name = "with log"
		}
		t.Run(name, func(t *testing.T) {
			addCmdLogs = logs
			command := &cobra.Command{Use: "add-test"}
			command.SetContext(context.WithValue(context.Background(), CmdKey, &CmdValType{
				names: []string{"test-instance"},
			}))
			err := addCmd.RunE(command, nil)
			if !errors.Is(err, geneos.ErrInvalidArgs) || !strings.Contains(err.Error(), "unknown or no component type given") {
				t.Fatalf("add error = %v, want the original instance.Add error", err)
			}
		})
	}
}
