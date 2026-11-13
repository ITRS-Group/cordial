package initcmd

import (
	"strings"
	"testing"
)

func TestInitRejectsEmptyArchive(t *testing.T) {
	flag := initCmd.PersistentFlags().Lookup("archive")
	previousArchive, previousChanged := initOptions.Archive, flag.Changed
	initOptions.Archive, flag.Changed = "", true
	t.Cleanup(func() { initOptions.Archive, flag.Changed = previousArchive, previousChanged })
	if err := initCmd.RunE(initCmd, nil); err == nil || !strings.Contains(err.Error(), "--archive must not be empty") {
		t.Fatalf("init error = %v, want empty archive error", err)
	}
}
