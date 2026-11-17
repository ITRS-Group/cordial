package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/config"
)

// writeScheduleConfig writes a configuration file for the tests of the
// schedules and returns its path
func writeScheduleConfig(t *testing.T, dir, name, schedule, emailSchedule, dbFile string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	body := fmt.Sprintf("gdna:\n  schedule: %q\n  email-schedule: %q\ndb:\n  file: %s\n", schedule, emailSchedule, dbFile)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckSchedules(t *testing.T) {
	dir := t.TempDir()

	for _, test := range []struct {
		name, schedule, email string
		wantErr               string
	}{
		{"valid", "*/15 * * * *", "", ""},
		{"valid with email", "*/15 * * * *", "0 6 * * *", ""},
		{"no schedule", "", "", "gdna.schedule"},
		{"bad schedule", "not a schedule", "", "gdna.schedule"},
		{"bad email schedule", "*/15 * * * *", "0 6 * *", "gdna.email-schedule"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cf, err := loadConfig(writeScheduleConfig(t, dir, "gdna.yaml", test.schedule, test.email, "x.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			err = checkSchedules(cf)
			switch {
			case test.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)):
				t.Errorf("error = %v, want one containing %q", err, test.wantErr)
			}
		})
	}
}

// nextRun returns when the job next runs
func nextRun(t *testing.T, job gocron.Job) time.Time {
	t.Helper()
	next, err := job.NextRun()
	if err != nil {
		t.Fatal(err)
	}
	return next
}

func TestReloadSchedules(t *testing.T) {
	dir := t.TempDir()

	// the jobs and the configuration file are globals, which the test
	// must leave as it found them
	oldSched, oldMain, oldEmail, oldCfgFile := sched, mainjob, emailjob, cfgFile
	t.Cleanup(func() {
		if sched != nil && sched != oldSched {
			sched.Shutdown()
		}
		sched, mainjob, emailjob, cfgFile = oldSched, oldMain, oldEmail, oldCfgFile
	})
	mainjob, emailjob = nil, nil

	var err error
	if sched, err = gocron.NewScheduler(); err != nil {
		t.Fatal(err)
	}
	sched.Start()

	noop := func() gocron.Task { return gocron.NewTask(func() {}) }

	// the schedules are far enough ahead that nothing is ever run
	initial, err := loadConfig(writeScheduleConfig(t, dir, "initial.yaml", "0 0 1 1 *", "", "a.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if err = scheduleJobs(initial, noop(), noop()); err != nil {
		t.Fatal(err)
	}
	if mainjob == nil || emailjob != nil {
		t.Fatalf("jobs after start-up: main %v, email %v", mainjob, emailjob)
	}
	mainID := mainjob.ID()

	ctx := context.Background()

	// a reload changes the schedule of the existing job and adds an email job
	cfgFile = writeScheduleConfig(t, dir, "gdna.yaml", "0 0 2 1 *", "0 6 * * *", "a.sqlite")
	current, err := reload(ctx, initial, nil)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if mainjob.ID() != mainID {
		t.Error("reload replaced the main job, not updated it")
	}
	if next := nextRun(t, mainjob); next.Month() != time.January || next.Day() != 2 {
		t.Errorf("main job next run = %s, want 2 January", next)
	}
	if emailjob == nil {
		t.Fatal("email job not added")
	}

	// a configuration that is not valid, or that changes the database,
	// is refused and nothing is changed
	for _, test := range []struct {
		name, schedule, db, wantErr string
	}{
		{"invalid schedule", "not a schedule", "a.sqlite", "gdna.schedule"},
		{"changed database", "0 0 3 1 *", "b.sqlite", "need a restart"},
	} {
		cfgFile = writeScheduleConfig(t, dir, "gdna.yaml", test.schedule, "0 6 * * *", test.db)
		if _, err = reload(ctx, current, nil); err == nil || !strings.Contains(err.Error(), test.wantErr) {
			t.Errorf("%s: error = %v, want one containing %q", test.name, err, test.wantErr)
		}
		if next := nextRun(t, mainjob); next.Day() != 2 {
			t.Errorf("%s: main job changed, next run = %s", test.name, next)
		}
		if emailjob == nil {
			t.Errorf("%s: email job removed", test.name)
		}
	}

	// removing the email schedule removes the job
	cfgFile = writeScheduleConfig(t, dir, "gdna.yaml", "0 0 2 1 *", "", "a.sqlite")
	if _, err = reload(ctx, current, nil); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if emailjob != nil {
		t.Error("email job not removed")
	}
	if jobs := sched.Jobs(); len(jobs) != 1 {
		t.Errorf("%d jobs scheduled, want 1", len(jobs))
	}
}

func TestRunningConfigFile(t *testing.T) {
	oldCfgFile := cfgFile
	t.Cleanup(func() { cfgFile = oldCfgFile })

	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("gdna.yaml", []byte("gdna:\n  schedule: \"0 0 1 1 *\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// relative, as given, which must be absolute for the commands to find
	// it from other directories
	cfgFile = "gdna.yaml"
	got := runningConfigFile()
	want, _ := filepath.EvalSymlinks(filepath.Join(dir, "gdna.yaml"))
	if g, _ := filepath.EvalSymlinks(got); !filepath.IsAbs(got) || g != want {
		t.Errorf("runningConfigFile() = %q, want %q", got, want)
	}

	// a file given on the command line is returned even if it has gone, so
	// that a reload fails and does not use the defaults
	cfgFile = filepath.Join(dir, "missing.yaml")
	if got = runningConfigFile(); got != cfgFile {
		t.Errorf("runningConfigFile() for a missing file = %q, want %q", got, cfgFile)
	}

	// with no file found, which is not a path such as "internal defaults"
	empty := t.TempDir()
	t.Chdir(empty)
	t.Setenv("HOME", empty)
	t.Setenv("XDG_CONFIG_HOME", empty)
	cfgFile = ""
	if got = runningConfigFile(); got != "" {
		t.Errorf("runningConfigFile() with no file = %q, want none", got)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	dir := t.TempDir()

	if _, err := loadConfig(filepath.Join(dir, "missing.yaml")); err == nil || !strings.Contains(err.Error(), "missing.yaml") {
		t.Errorf("loadConfig of a missing file: error = %v, want one naming the file", err)
	}

	if _, err := loadConfig(writeScheduleConfig(t, dir, "gdna.yaml", "0 0 1 1 *", "", "a.sqlite")); err != nil {
		t.Errorf("loadConfig of an existing file: %v", err)
	}
}

// the options for the connection to the Netprobe given on the command
// line must override the configuration, which depends on binding every
// flag by its real name
func TestBindNetprobeFlags(t *testing.T) {
	for _, test := range []struct {
		name  string
		cmd   *cobra.Command
		tls   string
		extra string
	}{
		{"start", startCmd, "secure", "--secure=false"},
		{"report", reportCmd, "tls", "--tls=false"},
	} {
		t.Run(test.name, func(t *testing.T) {
			flags := test.cmd.Flags()
			t.Cleanup(func() {
				flags.VisitAll(func(f *pflag.Flag) {
					f.Value.Set(f.DefValue)
					f.Changed = false
				})
			})

			if err := flags.Parse([]string{"--hostname", "np.example.com", "--port", "9999", test.extra, "--skip-verify", "--entity", "E", "--sampler", "S"}); err != nil {
				t.Fatal(err)
			}

			// a fresh configuration for each command, as binding changes it
			cf, err := loadConfig(writeScheduleConfig(t, t.TempDir(), "gdna.yaml", "0 0 1 1 *", "", "a.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			if err = bindNetprobeFlags(cf, flags, test.tls); err != nil {
				t.Fatal(err)
			}

			if got := config.Get[string](cf, "geneos.netprobe.hostname"); got != "np.example.com" {
				t.Errorf("hostname = %q, want np.example.com", got)
			}
			if got := config.Get[int](cf, "geneos.netprobe.port"); got != 9999 {
				t.Errorf("port = %d, want 9999", got)
			}
			if got := config.Get[bool](cf, "geneos.netprobe.secure"); got {
				t.Error("TLS = true, want the command line value false")
			}
			if got := config.Get[bool](cf, "geneos.netprobe.skip-verify"); !got {
				t.Error("skip-verify = false, want true")
			}
			if got := config.Get[string](cf, "geneos.entity"); got != "E" {
				t.Errorf("entity = %q, want E", got)
			}
			if got := config.Get[string](cf, "geneos.sampler"); got != "S" {
				t.Errorf("sampler = %q, want S", got)
			}
		})
	}
}
