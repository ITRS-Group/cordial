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
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
)

var once bool
var netprobeHost, entity, sampler string
var netprobePort int16
var secure, skipVerify, onStart, onStartEMail bool

func init() {
	GDNACmd.AddCommand(startCmd)

	startCmd.Flags().BoolVarP(&daemon, "daemon", "D", false, "Daemonise the process")
	startCmd.Flags().BoolVarP(&once, "once", "1", false, "Run once and exit")
	startCmd.Flags().BoolVarP(&onStart, "on-start", "O", false, "Run immediately on start-up, then follow schedule")
	startCmd.Flags().BoolVarP(&onStartEMail, "on-start-email", "E", false, "Run immediately on start-up, send email report, then follow schedule")

	startCmd.Flags().StringVarP(&reportNames, "reports", "r", "", reportNamesDescription)

	startCmd.Flags().StringVarP(&netprobeHost, "hostname", "H", "localhost", "Connect to netprobe at `hostname`")
	startCmd.Flags().Int16VarP(&netprobePort, "port", "P", 7036, "Connect to netprobe on `port`")
	startCmd.Flags().BoolVarP(&secure, "secure", "S", false, "Use TLS connection to Netprobe")
	startCmd.Flags().BoolVarP(&skipVerify, "skip-verify", "k", false, "Skip certificate verification for Netprobe connections")
	startCmd.Flags().StringVarP(&entity, "entity", "e", "GDNA", "Send reports to Managed `Entity`")
	startCmd.Flags().StringVarP(&sampler, "sampler", "s", "GDNA", "Send reports to `Sampler`")
	startCmd.Flags().BoolVarP(&resetViews, "reset", "R", false, "Reset/Delete configured Dataviews on first run")

	startCmd.Flags().SortFlags = false
}

//go:embed _docs/start.md
var startCmdDescription string

var mainjob, emailjob gocron.Job
var sched gocron.Scheduler
var maintask, emailtask gocron.Task

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
		cf.Viper.BindPFlag("geneos.netprobe.hostname", cmd.Flags().Lookup("host"))
		cf.Viper.BindPFlag("geneos.netprobe.port", cmd.Flags().Lookup("port"))
		cf.Viper.BindPFlag("geneos.netprobe.secure", cmd.Flags().Lookup("secure"))
		cf.Viper.BindPFlag("geneos.netprobe.skip-verify", cmd.Flags().Lookup("skip-verify"))
		cf.Viper.BindPFlag("geneos.entity", cmd.Flags().Lookup("entity"))
		cf.Viper.BindPFlag("geneos.sampler", cmd.Flags().Lookup("sampler"))
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)
		return start()
	},
}

func start() (err error) {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	db, err := openDB(ctx, cf, "db.dsn", false)
	if err != nil {
		return
	}
	defer db.Close()

	if once {
		return do(ctx, cf, db)
	}

	sched, err = gocron.NewScheduler(gocron.WithLimitConcurrentJobs(1, gocron.LimitModeWait))
	if err != nil {
		return
	}

	sched.Start()

	// save these as a global for config updates
	maintask = gocron.NewTask(do, ctx, cf, db)
	emailtask = gocron.NewTask(doEmail, ctx, cf, db, reportNames)

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

	mainjob, err = sched.NewJob(
		gocron.CronJob(cf.GetString("gdna.schedule"), false),
		maintask,
		gocron.WithName("main"),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		listeners,
	)
	if err != nil {
		return
	}

	rjt, _ := mainjob.NextRun()
	log.Info().Msgf("next scheduled report job %v", rjt)

	if es := cf.GetString("gdna.email-schedule"); es != "" {
		emailjob, err = sched.NewJob(
			gocron.CronJob(es, false),
			emailtask,
			gocron.WithName("email"),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
			listeners,
		)
		if err != nil {
			return err
		} else {
			ejt, _ := emailjob.NextRun()
			log.Info().Msgf("next scheduled email job %v", ejt)
		}
	}

	defer sched.Shutdown()

	<-ctx.Done()

	return
}

func updateJobs() {
	var err error
	if once {
		return
	}
	if mainjob != nil {
		mainjob, err = sched.Update(mainjob.ID(),
			gocron.CronJob(cf.GetString("gdna.schedule"), false),
			maintask,
			gocron.WithName("main"),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
			listeners,
		)
		if err != nil {
			log.Error().Err(err).Msg("updating main job")
			return
		}

		rjt, _ := mainjob.NextRun()
		log.Info().Msgf("next report job %v", rjt)
	}

	if es := cf.GetString("gdna.email-schedule"); es != "" {
		log.Debug().Msgf("email schedule: %s", es)
		if emailjob != nil {
			emailjob, err = sched.Update(emailjob.ID(),
				gocron.CronJob(es, false),
				emailtask,
				gocron.WithName("email"),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
				listeners,
			)
		} else {
			emailjob, err = sched.NewJob(
				gocron.CronJob(es, false),
				emailtask,
				gocron.WithName("email"),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
				listeners,
			)
		}
		if err != nil {
			log.Error().Err(err).Msg("updating email job")
			return
		}
		ejt, _ := emailjob.NextRun()
		log.Info().Msgf("next email job %v", ejt)
	}

}

var listeners = gocron.WithEventListeners(
	gocron.BeforeJobRuns(beforejob),
	gocron.AfterJobRuns(afterjob),
	gocron.AfterJobRunsWithError(afterjoberr),
)

func beforejob(_ uuid.UUID, jobName string) {
	log.Info().Msgf("running %s", jobName)
}

func afterjob(_ uuid.UUID, jobName string) {
	log.Info().Msgf("finished %s", jobName)
}

func afterjoberr(_ uuid.UUID, jobName string, err error) {
	log.Error().Err(err).Msgf("finished %s with error", jobName)
}

func do(ctx context.Context, cf *config.Config, db *sql.DB) (err error) {
	sources, err := fetch(ctx, cf, db)
	if err != nil {
		return
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Error().Err(err).Msg("cannot BEGIN transaction")
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
		log.Error().Err(err).Msg("cannot BEGIN transaction")
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
