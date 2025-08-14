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
	_ "embed"
	"errors"
	"os"
	"syscall"
	"time"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

//go:embed _docs/stop.md
var stopCmdDescription string

//go:embed _docs/restart.md
var restartCmdDescription string

func init() {
	GDNACmd.AddCommand(stopCmd)
	GDNACmd.AddCommand(restartCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop background GDNA process",
	Long:  stopCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		return stop()
	},
}

func stop() (err error) {
	pid, err := process.GetPID(host.Localhost, "gdna", true, nil, nil, "start")
	if err != nil {
		if err == os.ErrProcessDone {
			log.Info().Msg("process not found")
			return nil
		}
		log.Error().Err(err).Msg("failed to find process")
		return
	}

	log.Info().Msgf("would kill pid %d", pid)
	h := host.Localhost
	if err = h.Signal(pid, syscall.SIGTERM); err == os.ErrProcessDone {
		log.Info().Msgf("terminated PID %d", pid)
		return nil
	}

	if errors.Is(err, syscall.EPERM) {
		return os.ErrPermission
	}

	for j := 0; j < 10; j++ {
		time.Sleep(250 * time.Millisecond)
		if err = h.Signal(pid, syscall.SIGTERM); err == os.ErrProcessDone {
			log.Info().Msgf("terminated PID %d", pid)
			return nil
		}
	}

	if err = h.Signal(pid, syscall.SIGKILL); err == os.ErrProcessDone {
		log.Info().Msgf("killing PID %d", pid)
		return nil
	}

	return err
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart background GDNA process",
	Long:  restartCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		return nil
	},
}
