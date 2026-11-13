package pkgcmd

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func TestInstallArchiveSources(t *testing.T) {
	root := t.TempDir()
	key := cordial.ExecutableName()
	previousRoot := config.Get[string](config.Global(), key)
	previousArchives, previousOverride := installCmdArchives, installCmdOverride
	t.Cleanup(func() {
		config.Set(config.Global(), key, previousRoot)
		installCmdArchives, installCmdOverride = previousArchives, previousOverride
	})
	config.Set(config.Global(), key, root)
	installCmdOverride = "netprobe:1.0.0"
	first := filepath.Join(root, "first.tar.gz")
	second := filepath.Join(root, "archives,with-comma")

	for _, test := range []struct {
		name       string
		args       []string
		wantPaths  []string
		wantSingle bool
	}{
		{name: "long flag", args: []string{"--archive", first}, wantPaths: []string{first}, wantSingle: true},
		{name: "short flag", args: []string{"-A", first}, wantPaths: []string{first}, wantSingle: true},
		{name: "repeated flags", args: []string{"-A", first, "--archive", second}, wantPaths: []string{first, second}},
		{name: "mixed sources", args: []string{first, "--archive", second}, wantPaths: []string{second}},
	} {
		t.Run(test.name, func(t *testing.T) {
			installCmdArchives = nil
			flags := pflag.NewFlagSet("install-test", pflag.ContinueOnError)
			archiveFlag := *installCmd.Flags().Lookup("archive")
			flags.AddFlag(&archiveFlag)
			if err := flags.Parse(test.args); err != nil {
				t.Fatal(err)
			}

			if !slices.Equal(installCmdArchives, test.wantPaths) {
				t.Fatalf("archive paths = %q, want %q", installCmdArchives, test.wantPaths)
			}
			command := &cobra.Command{
				Use: "install-test",
				Annotations: map[string]string{
					cmd.CmdGlobal:       "false",
					cmd.CmdAuditCommand: "never",
					cmd.CmdAuditActions: "never",
				},
			}
			command.SetContext(context.WithValue(context.Background(), cmd.CmdKey, &cmd.CmdValType{}))
			if err := cmd.ParseArgs(command, flags.Args()); err != nil {
				t.Fatal(err)
			}
			err := installCmd.RunE(command, nil)
			if test.wantSingle {
				if !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("single missing archive error = %v, want file-not-found validation", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "only valid with a single file parameter") {
				t.Fatalf("combined sources error = %v, want multiple-source override validation", err)
			}
		})
	}
	if got := installCmd.Flags().ShorthandLookup("A"); got == nil || got.Name != "archive" {
		t.Fatal("-A must select --archive")
	}
	if got := installCmd.Flags().Lookup("all"); got == nil || got.Shorthand != "" {
		t.Fatal("--all must remain available without a conflicting shorthand")
	}
}

func TestInstallExplicitArchiveReportsMissingVersion(t *testing.T) {
	root := t.TempDir()
	h := &geneos.Host{Host: host.NewLocal(), Config: config.New()}
	config.Set(h.Config, cordial.ExecutableName(), root)
	previousLocal, previousVersion := geneos.LOCAL, installCmdVersion
	geneos.LOCAL, installCmdVersion = h, "1.2"
	t.Cleanup(func() {
		geneos.LOCAL, installCmdVersion = previousLocal, previousVersion
	})
	ct := &geneos.Component{Name: "archive-test-component"}
	ct.Register(nil)

	err := install(h, ct, true, geneos.Source(t.TempDir()), geneos.Version("1.2"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("install error = %v, want missing archive", err)
	}
}

func TestInstallDownloadRejectsLocalArchive(t *testing.T) {
	previousDownload, previousArchives := installCmdDownloadOnly, installCmdArchives
	previousLocal, previousBase := installCmdLocal, installCmdBase
	previousUpdate, previousNoSave, previousOverride := installCmdUpdate, installCmdNoSave, installCmdOverride
	t.Cleanup(func() {
		installCmdDownloadOnly, installCmdArchives = previousDownload, previousArchives
		installCmdLocal, installCmdBase = previousLocal, previousBase
		installCmdUpdate, installCmdNoSave, installCmdOverride = previousUpdate, previousNoSave, previousOverride
	})
	installCmdDownloadOnly, installCmdArchives = true, []string{t.TempDir()}
	installCmdLocal, installCmdBase = false, "active_prod"
	installCmdUpdate, installCmdNoSave, installCmdOverride = false, false, ""
	if err := installCmd.RunE(installCmd, nil); err == nil || !strings.Contains(err.Error(), "local --archive") {
		t.Fatalf("install error = %v, want incompatible local archive error", err)
	}
}

func TestInstallArchiveURLWithOverride(t *testing.T) {
	root := t.TempDir()
	h := &geneos.Host{Host: host.NewLocal(), Config: config.New()}
	config.Set(h.Config, cordial.ExecutableName(), root)
	previousRoot := config.Get[string](config.Global(), cordial.ExecutableName())
	previousLocal, previousHostname := geneos.LOCAL, cmd.Hostname
	previousArchives, previousOverride := installCmdArchives, installCmdOverride
	t.Cleanup(func() {
		config.Set(config.Global(), cordial.ExecutableName(), previousRoot)
		geneos.LOCAL, cmd.Hostname = previousLocal, previousHostname
		installCmdArchives, installCmdOverride = previousArchives, previousOverride
	})
	config.Set(config.Global(), cordial.ExecutableName(), root)
	geneos.LOCAL, cmd.Hostname = h, "localhost"
	(&geneos.Component{Name: "archive-url-test"}).Register(nil)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	installCmdArchives = []string{server.URL + "/release.tar.gz"}
	installCmdOverride = "archive-url-test:1.0.0"
	command := &cobra.Command{Use: "install-test"}
	command.SetContext(context.WithValue(context.Background(), cmd.CmdKey, &cmd.CmdValType{}))
	if err := cmd.ParseArgs(command, nil); err != nil {
		t.Fatal(err)
	}
	if err := installCmd.RunE(command, nil); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("install error = %v, want HTTP error from explicit URL", err)
	}

	installCmdOverride = ""
	installCmdArchives = []string{t.TempDir(), server.URL + "/archive-url-test-1.0.0-linux-x64.tar.gz"}
	command.SetContext(context.WithValue(context.Background(), cmd.CmdKey, &cmd.CmdValType{}))
	if err := cmd.ParseArgs(command, []string{"archive-url-test"}); err != nil {
		t.Fatal(err)
	}
	if err := installCmd.RunE(command, nil); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("install error = %v, want explicit URL attempted after empty directory", err)
	}
}
