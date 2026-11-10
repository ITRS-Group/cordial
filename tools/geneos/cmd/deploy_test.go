package cmd

import "testing"

func TestDeployLogImpliesStart(t *testing.T) {
	previousStart, previousLogs := deployCmdStart, deployCmdLogs
	previousArchive, previousLocal := deployCmdArchive, deployCmdLocal
	t.Cleanup(func() {
		deployCmdStart, deployCmdLogs = previousStart, previousLogs
		deployCmdArchive, deployCmdLocal = previousArchive, previousLocal
	})

	for _, test := range []struct {
		name    string
		start   bool
		logs    bool
		archive string
		want    bool
	}{
		{name: "default"},
		{name: "start only", start: true, want: true},
		{name: "log only", logs: true, want: true},
		{name: "start and log", start: true, logs: true, want: true},
		{name: "archive only", archive: "/path/to/archive.tgz"},
		{name: "archive and log", archive: "/path/to/archive.tgz", logs: true, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			deployCmdStart, deployCmdLogs = test.start, test.logs
			deployCmdArchive, deployCmdLocal = test.archive, false
			deployCmd.PreRun(deployCmd, nil)
			if deployCmdStart != test.want {
				t.Errorf("start after deploy = %v, want %v", deployCmdStart, test.want)
			}
		})
	}
}

func TestDeployArchiveImpliesLocal(t *testing.T) {
	previousArchive, previousLocal := deployCmdArchive, deployCmdLocal
	t.Cleanup(func() {
		deployCmdArchive, deployCmdLocal = previousArchive, previousLocal
	})

	for _, test := range []struct {
		name    string
		archive string
		local   bool
		want    bool
	}{
		{name: "default"},
		{name: "explicit local", local: true, want: true},
		{name: "archive file", archive: "release.tar.gz", want: true},
		{name: "archive directory", archive: "/tmp/releases", want: true},
		{name: "missing archive", archive: "/nonexistent/release.tar.gz", want: true},
		{name: "archive URL", archive: "https://example.com/release.tar.gz"},
		{name: "archive URL with explicit local", archive: "https://example.com/release.tar.gz", local: true, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			deployCmdArchive, deployCmdLocal = test.archive, test.local
			deployCmd.PreRun(deployCmd, nil)
			if deployCmdLocal != test.want {
				t.Errorf("local mode = %v, want %v", deployCmdLocal, test.want)
			}
		})
	}
}
