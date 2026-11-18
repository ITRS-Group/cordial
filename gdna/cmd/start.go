/*
Copyright © 2024 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/pkg/process/control"
)

// netprobeDefaults are the defaults of the options for the connection
// to the Netprobe, which are those in the default configuration so that
// the help of the start and report commands is correct
var netprobeDefaults = func() (d struct {
	hostname, entity, sampler string
	port                      uint16
	secure, skipVerify        bool
}) {
	cf := config.New(config.WithDefaults(defaults, "yaml"))
	d.hostname = config.Get[string](cf, "geneos.netprobe.hostname")
	d.port = uint16(config.Get[int](cf, "geneos.netprobe.port"))
	d.secure = config.Get[bool](cf, "geneos.netprobe.secure")
	d.skipVerify = config.Get[bool](cf, "geneos.netprobe.skip-verify")
	d.entity = config.Get[string](cf, "geneos.entity")
	d.sampler = config.Get[string](cf, "geneos.sampler")
	return
}()

var once bool
var netprobeHost, entity, sampler string
var netprobePort uint16
var secure, skipVerify, onStart, onStartEMail bool

func init() {
	Cmd.AddCommand(startCmd)

	startCmd.Flags().BoolVarP(&daemon, "daemon", "D", false, "Daemonise the process")
	startCmd.Flags().StringVar(&pidFile, "pidfile", "", control.FlagUsageFor(pidFileKey))
	startCmd.Flags().BoolVarP(&once, "once", "1", false, "Run once and exit")
	startCmd.Flags().BoolVarP(&onStart, "on-start", "O", false, "Run immediately on start-up, then follow schedule")
	startCmd.Flags().BoolVarP(&onStartEMail, "on-start-email", "E", false, "Run immediately on start-up, send email report, then follow schedule")

	startCmd.Flags().StringVarP(&reportNames, "reports", "r", "", reportNamesDescription)

	startCmd.Flags().StringVarP(&netprobeHost, "hostname", "H", netprobeDefaults.hostname, "Connect to netprobe at `hostname`")
	startCmd.Flags().Uint16VarP(&netprobePort, "port", "P", netprobeDefaults.port, "Connect to netprobe on `port`")
	startCmd.Flags().BoolVarP(&secure, "secure", "S", netprobeDefaults.secure, "Use TLS connection to Netprobe")
	startCmd.Flags().BoolVarP(&skipVerify, "skip-verify", "k", netprobeDefaults.skipVerify, "Skip certificate verification for Netprobe connections")
	startCmd.Flags().StringVarP(&entity, "entity", "e", netprobeDefaults.entity, "Send reports to Managed `Entity`")
	startCmd.Flags().StringVarP(&sampler, "sampler", "s", netprobeDefaults.sampler, "Send reports to `Sampler`")
	startCmd.Flags().BoolVarP(&resetViews, "reset", "R", false, "Reset/Delete configured Dataviews on first run")

	startCmd.Flags().SortFlags = false
	startFlags = startCmd.Flags()
}

//go:embed _docs/start.md
var startCmdDescription string

// the jobs run by the schedules, which are variables so that the tests
// can replace them
var (
	runMain  = do
	runEmail = doEmail
)

var mainjob, emailjob gocron.Job
var sched gocron.Scheduler

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start cycling though fetch, report etc.",
	Long:  startCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := bindNetprobeFlags(cf, cmd.Flags(), "secure"); err != nil {
			log.Error("binding command line options", slog.Any("error", err))
		}
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		return start()
	},
}

// startFlags are the flags of the start command, which a reload binds
// to the new configuration
var startFlags *pflag.FlagSet

// bindNetprobeFlags makes the options given on the command line for the
// connection to the Netprobe override the configuration. The options are
// the same for the start and report commands, except for the name of the
// flag for TLS, which is tlsFlag. It returns an error for any flag that
// does not exist.
func bindNetprobeFlags(cf *config.Config, flags *pflag.FlagSet, tlsFlag string) error {
	var errs []error
	for _, b := range []struct{ key, flag string }{
		{"geneos.netprobe.hostname", "hostname"},
		{"geneos.netprobe.port", "port"},
		{"geneos.netprobe.secure", tlsFlag},
		{"geneos.netprobe.skip-verify", "skip-verify"},
		{"geneos.entity", "entity"},
		{"geneos.sampler", "sampler"},
	} {
		if err := cf.BindPFlag(b.key, flags.Lookup(b.flag)); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", b.flag, err))
		}
	}
	return errors.Join(errs...)
}

// daemonize starts `gdna start` again in the background, waits until it
// is ready, which is when its schedules are running and it holds its PID
// file, and prints its PID. It does not return if it is successful.
func daemonize(cmd *cobra.Command) error {
	// load the configuration first so that a missing or invalid one is
	// reported here and not lost in the log of the background process
	cf, err := loadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("cannot start in the background: %w", err)
	}
	if err = validateLayerReportSelection(reportNames); err != nil {
		return err
	}
	if err = checkSchedules(cf); err != nil {
		return fmt.Errorf("cannot start in the background: %w", err)
	}

	logName := logFile
	if f := cmd.Flag("logfile"); f != nil && !f.Changed {
		logName = config.Get[string](cf, cf.Join("gdna", "log", "filename"))
	}

	pid, err := control.Daemonize(pidFilePath(cf))
	if err != nil {
		return fmt.Errorf("failed to start in the background, see the log file %q: %w", logName, err)
	}
	fmt.Println(pid)
	os.Exit(0)
	return nil // not reached
}

// reloadSignals returns the signals that make a background process read
// its configuration again, which are SIGUSR1 and, as it is the usual
// signal to ask for that, SIGHUP, which is also what a hung up terminal
// sends. There are none on Windows.
func reloadSignals() []os.Signal {
	if process.SigReload == nil {
		return nil
	}
	return []os.Signal{process.SigReload, syscall.SIGHUP}
}

func start() (err error) {
	// Handle SIGINT (CTRL+C) and SIGTERM, which is what `gdna stop`
	// sends, gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// catch the reload signals before there is anything that can send
	// them, as the default action is to terminate the process. They are
	// not available on Windows.
	var reloads chan os.Signal
	if signals := reloadSignals(); len(signals) > 0 && !once {
		reloads = make(chan os.Signal, 1)
		signal.Notify(reloads, signals...)
		defer signal.Stop(reloads)
	}

	// hold the PID file, if there is one, before anything else so that
	// a second process fails with a clear error. It is only written
	// once the schedules are running, which is how the process says it
	// is ready. A single run does not use it, so that it can be run
	// while a background process is.
	var pf *process.PIDFile
	if path := control.ConfiguredPIDFileKey(cf, pidFile, pidFileKey); path != "" && !once {
		if pf, err = process.LockPIDFile(path); err != nil {
			log.Error("cannot use pid file", slog.String("path", path), slog.Any("error", err))
			return
		}
		defer pf.Release()
	}

	db, err := openDB(ctx, cf, cf.Join("db", "dsn"), false)
	if err != nil {
		return
	}
	defer db.Close()
	defer closeLicdClient()

	if once {
		return do(ctx, cf, db)
	}

	sched, err = gocron.NewScheduler(gocron.WithLimitConcurrentJobs(1, gocron.LimitModeWait))
	if err != nil {
		return
	}

	log.Debug("starting scheduler")
	sched.Start()

	// the tasks use the configuration they are given, so that a reload
	// can replace them
	maintask := gocron.NewTask(runMain, ctx, cf, db)
	emailtask := gocron.NewTask(runEmail, ctx, cf, db, reportNames)

	if onStartEMail || onStart {
		sched.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
			maintask,
			gocron.WithName("on start-up"),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
			listeners,
		)
		if onStartEMail {
			sched.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
				emailtask,
				gocron.WithName("on start-up email"),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
				listeners,
			)
		}
	} else {
		sched.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
			gocron.NewTask(fetch, ctx, cf, db),
			gocron.WithName("initial fetch"),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
			listeners,
		)
	}

	if err = scheduleJobs(cf, maintask, emailtask); err != nil {
		return
	}

	defer sched.Shutdown()

	// the schedules are running, so say that the process is ready
	if pf != nil {
		if err = writePIDFile(pf); err != nil {
			log.Error("cannot write pid file", slog.String("path", pf.Path()), slog.Any("error", err))
			return
		}
	}

	// the configuration in use, which is replaced by a reload; the
	// package level one is only for the commands
	current := cf

	for {
		select {
		case <-ctx.Done():
			log.Info("shutting down")
			return

		case <-reloads:
			log.Info("reloading configuration")
			ncf, err := reload(ctx, current, db)
			if err != nil {
				log.Error("configuration not reloaded, continuing with the current configuration", slog.Any("error", err))
				continue
			}
			current = ncf
			log.Info("configuration reloaded")
		}
	}
}

// reload reads the configuration again and, if it can be used, makes
// the schedules and all later runs of the jobs use it, returning the new
// configuration. A job that is running completes using the one it
// started with. Otherwise the error is returned and everything continues
// as it was.
//
// Only the schedules and the settings that are read for each run can be
// changed. The database, the logging and the PID file are not.
func reload(ctx context.Context, current *config.Config, db *sql.DB) (*config.Config, error) {
	ncf, err := loadConfig(runningConfigFile())
	if err != nil {
		return nil, fmt.Errorf("cannot load configuration: %w", err)
	}

	// keep the options given on the command line
	bindNetprobeFlags(ncf, startFlags, "secure")

	if err = checkSchedules(ncf); err != nil {
		return nil, err
	}

	if dsn := config.Get[string](ncf, ncf.Join("db", "dsn")); dsn != config.Get[string](current, current.Join("db", "dsn")) {
		return nil, fmt.Errorf("changes to the database (%s) need a restart", ncf.Join("db", "dsn"))
	}

	if err = scheduleJobs(ncf,
		gocron.NewTask(runMain, ctx, ncf, db),
		gocron.NewTask(runEmail, ctx, ncf, db, reportNames),
	); err != nil {
		return nil, err
	}

	// the connections to the licds were made using the old settings
	closeLicdClient()
	return ncf, nil
}

// checkSchedules returns an error if the schedules in the configuration
// are not valid, without changing anything
func checkSchedules(cf *config.Config) error {
	// a scheduler that is not started accepts jobs, and checks their
	// schedules, but never runs them
	s, err := gocron.NewScheduler()
	if err != nil {
		return err
	}
	defer s.Shutdown()

	for _, schedule := range []struct {
		key      string
		optional bool
	}{
		{"schedule", false},
		{"email-schedule", true},
	} {
		expr := config.Get[string](cf, cf.Join("gdna", schedule.key))
		if expr == "" && schedule.optional {
			continue
		}
		if _, err = s.NewJob(gocron.CronJob(expr, false), gocron.NewTask(func() {})); err != nil {
			return fmt.Errorf("%s %q: %w", cf.Join("gdna", schedule.key), expr, err)
		}
	}
	return nil
}

// scheduleJobs adds the main and, if configured, the email jobs to the
// scheduler, using the schedules in the configuration, or updates them
// if they already exist, which leaves them unchanged if the update
// fails. The email job is removed if it is no longer scheduled.
func scheduleJobs(cf *config.Config, maintask, emailtask gocron.Task) (err error) {
	job, err := setJob(mainjob, "main", config.Get[string](cf, cf.Join("gdna", "schedule")), maintask)
	if err != nil {
		log.Error("scheduling main job", slog.Any("error", err))
		return
	}
	mainjob = job

	rjt, _ := mainjob.NextRun()
	log.Info("next scheduled report job", slog.Time("at", rjt))

	es := config.Get[string](cf, cf.Join("gdna", "email-schedule"))
	if es == "" {
		if emailjob != nil {
			if err = sched.RemoveJob(emailjob.ID()); err != nil {
				log.Error("removing email job", slog.Any("error", err))
				return
			}
			emailjob = nil
		}
		return
	}

	job, err = setJob(emailjob, "email", es, emailtask)
	if err != nil {
		log.Error("scheduling email job", slog.Any("error", err))
		return
	}
	emailjob = job

	ejt, _ := emailjob.NextRun()
	log.Info("next scheduled email job", slog.Time("at", ejt))
	return
}

// setJob adds a job to the scheduler with the cron schedule or, if job
// is not nil, updates it. The job passed is returned if there is an error.
func setJob(job gocron.Job, name, schedule string, task gocron.Task) (gocron.Job, error) {
	var updated gocron.Job
	var err error
	if job == nil {
		updated, err = sched.NewJob(
			gocron.CronJob(schedule, false),
			task,
			gocron.WithName(name),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
			listeners,
		)
	} else {
		updated, err = sched.Update(job.ID(),
			gocron.CronJob(schedule, false),
			task,
			gocron.WithName(name),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
			listeners,
		)
	}
	if err != nil {
		return job, err
	}
	return updated, nil
}

var listeners = gocron.WithEventListeners(
	gocron.BeforeJobRuns(beforejob),
	gocron.AfterJobRuns(afterjob),
	gocron.AfterJobRunsWithError(afterjoberr),
)

func beforejob(_ uuid.UUID, jobName string) {
	log.Info("running job", slog.String("job", jobName))
}

func afterjob(_ uuid.UUID, jobName string) {
	log.Info("finished job", slog.String("job", jobName))
}

func afterjoberr(_ uuid.UUID, jobName string, err error) {
	log.Error("finished job with error", slog.Any("error", err), slog.String("job", jobName))
}

func do(ctx context.Context, cf *config.Config, db *sql.DB) (err error) {
	sources, err := fetch(ctx, cf, db)
	if err != nil {
		return
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("cannot BEGIN transaction", slog.Any("error", err))
		return
	}

	if err = updateReportingDatabase(ctx, cf, tx, sources); err != nil {
		tx.Rollback()
		return
	}

	// commit the reporting database changes in case we are not using
	// temp tables and want to catch the data for debug purposes
	if err = tx.Commit(); err != nil {
		return
	}

	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("cannot BEGIN transaction", slog.Any("error", err))
		return
	}
	defer tx.Rollback()
	if err = report(ctx, cf, tx, io.Discard, "dataview", reportNames); err != nil {
		return
	}

	// reset after first report if looping
	resetViews = false
	return tx.Commit()
}
