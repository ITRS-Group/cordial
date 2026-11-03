package pscmd

import (
	"testing"
)

func TestFilesAndNetworkFlagsAreMutuallyExclusive(t *testing.T) {
	t.Cleanup(func() {
		for _, name := range []string{"files", "network"} {
			flag := psCmd.Flags().Lookup(name)
			_ = psCmd.Flags().Set(name, "false")
			flag.Changed = false
		}
	})

	if err := psCmd.Flags().Parse([]string{"--files", "--network"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	err := psCmd.ValidateFlagGroups()
	if err == nil {
		t.Fatal("ValidateFlagGroups() accepted --files and --network together")
	}
}
