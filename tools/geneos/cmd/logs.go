/*
Copyright © 2022 ITRS Group

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
	"io"
	"io/fs"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var logCmdLines int
var logCmdStderr, logCmdNoNormal, logCmdCALog, logCmdFollow, logCmdCat bool
var logCmdMatch, logCmdIgnore string
var logCmdNoHeaders bool

type files struct {
	instance geneos.Instance
	reader   io.ReadSeekCloser
	offset   int64
}

// global watchers for logs
var tails *sync.Map

func init() {
	Cmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolVarP(&logCmdFollow, "follow", "f", false, "Follow file")
	logsCmd.Flags().IntVarP(&logCmdLines, "lines", "n", 10, "Lines to tail")
	logsCmd.Flags().BoolVarP(&logCmdCat, "cat", "c", false, "Output whole file")

	logsCmd.Flags().BoolVarP(&logCmdStderr, "stderr", "E", false, "Show STDERR output files")
	logsCmd.Flags().BoolVarP(&logCmdNoNormal, "no-stdout", "N", false, "Do not show STDOUT log files")
	logsCmd.Flags().BoolVarP(&logCmdNoHeaders, "no-headers", "X", false, "Do not show log file headers, useful for --match/-g output")
	logsCmd.Flags().BoolVarP(&logCmdCALog, "ca", "C", false, "Include Collection Agent log for Netprobe instances")

	logsCmd.Flags().StringVarP(&logCmdMatch, "match", "g", "", "Match lines with STRING")
	logsCmd.Flags().StringVarP(&logCmdIgnore, "ignore", "v", "", "Match lines without STRING")

	logsCmd.MarkFlagsMutuallyExclusive("match", "ignore")
	logsCmd.MarkFlagsMutuallyExclusive("cat", "follow")

	logsCmd.Flags().SortFlags = false
}

//go:embed _docs/logs.md
var logsCmdDescription string

var logsCmd = &cobra.Command{
	Use:          "logs [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupView,
	Short:        "View Instance Logs",
	Long:         logsCmdDescription,
	Aliases:      []string{"log"},
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:               "true",
		CmdRequireHome:          "true",
		CmdWildcardNames:        "true",
		CmdAllowRoot:            "true",
		CmdNonInstanceArgsError: "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names, _, err := FetchArgs(cmd)
		if err != nil {
			return
		}

		// if we have match or exclude with other defaults, then turn on logcat
		if (logCmdMatch != "" || logCmdIgnore != "") && !logCmdFollow {
			logCmdCat = true
		}

		if !term.IsTerminal(int(os.Stdout.Fd())) {
			color.NoColor = true
		}

		switch {
		case logCmdCat:
			instance.Do(geneos.GetHost(Hostname), ct, names, instance.CatInstance,
				instance.WithStderr(logCmdStderr),
				instance.WithoutNormal(logCmdNoNormal),
				instance.WithCALog(logCmdCALog),
				instance.WithLines(logCmdLines),
				instance.WithMatch(logCmdMatch),
				instance.WithIgnore(logCmdIgnore),
			).Report(os.Stdout, responses.SkipOnErr(false), responses.IgnoreErrs(fs.ErrNotExist))
		case logCmdFollow:
			instance.FollowLogs(geneos.GetHost(Hostname), ct, names,
				instance.WithStderr(logCmdStderr),
				instance.WithoutNormal(logCmdNoNormal),
				instance.WithCALog(logCmdCALog),
				instance.WithLines(logCmdLines),
				instance.WithMatch(logCmdMatch),
				instance.WithIgnore(logCmdIgnore),
			) // never returns
		default:
			instance.Do(geneos.GetHost(Hostname), ct, names, instance.TailInstance,
				instance.WithStderr(logCmdStderr),
				instance.WithoutNormal(logCmdNoNormal),
				instance.WithCALog(logCmdCALog),
				instance.WithLines(logCmdLines),
				instance.WithMatch(logCmdMatch),
				instance.WithIgnore(logCmdIgnore),
			).Report(os.Stdout, responses.SkipOnErr(false), responses.IgnoreErrs(fs.ErrNotExist))
		}

		return
	},
}
