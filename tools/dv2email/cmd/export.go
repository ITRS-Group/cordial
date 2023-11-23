/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/commands"
)

//go:embed _docs/root.md
var exportCmdDescription string

var exportCmdDir string
var exportCmdFirstColumn, exportCmdHeadlines, exportCmdRows, exportCmdColumns string

func init() {
	DV2EMAILCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVar(&exportCmdDir, "dir", "", "destination `directory`, defaults to current")
	exportCmd.Flags().StringVar(&exportCmdFirstColumn, "rowname", "", "set row `name`")
	exportCmd.Flags().StringVar(&exportCmdHeadlines, "headlines", "", "filter headlines, comma-separated string")
	exportCmd.Flags().StringVar(&exportCmdRows, "rows", "", "filter rows, comma-separated string")
	exportCmd.Flags().StringVar(&exportCmdColumns, "columns", "", "filter columns, comma-separated string")
}

// exportCmd represents the write command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export dataview(s) to local files",
	Long:  exportCmdDescription,
	Run: func(cmd *cobra.Command, args []string) {
		if exportCmdDir == "" {
			exportCmdDir = "."
		}

		gw, err := dialGateway(cf)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		data, err := fetchDataviews(gw, exportCmdFirstColumn, exportCmdHeadlines, exportCmdRows, exportCmdColumns)

		switch cf.GetString("email.split") {
		case "entity":
			entities := map[string][]*commands.Dataview{}
			for _, d := range data.Dataviews {
				if len(entities[d.XPath.Entity.Name]) == 0 {
					entities[d.XPath.Entity.Name] = []*commands.Dataview{}
				}
				entities[d.XPath.Entity.Name] = append(entities[d.XPath.Entity.Name], d)
			}
			for _, e := range entities {
				many := DV2EMailData{
					Dataviews: e,
					Env:       data.Env,
				}
				if err = writeFiles(exportCmdDir, many); err != nil {
					log.Fatal().Err(err).Msg("")
				}
			}
		case "dataview":
			for _, d := range data.Dataviews {
				one := DV2EMailData{
					Dataviews: []*commands.Dataview{d},
					Env:       data.Env,
				}
				if err = writeFiles(exportCmdDir, one); err != nil {
					log.Fatal().Err(err).Msg("")
				}
			}
		default:
			if err = writeFiles(exportCmdDir, data); err != nil {
				log.Fatal().Err(err).Msg("")
			}
		}
	},
}

func writeFiles(dir string, data DV2EMailData) (err error) {
	run := time.Now()

	if err = os.MkdirAll(dir, 0775); err != nil {
		return
	}

	if slices.Contains(cf.GetStringSlice("files"), "texttable") {
		var files []dataFile
		files, err = buildTextTableFiles(cf, data, run)

		if err != nil {
			return err
		}
		for _, file := range files {
			var f *os.File
			f, err = os.Create(filepath.Join(dir, file.name))
			if err != nil {
				return
			}
			if _, err = io.Copy(f, file.content); err != nil {
				return
			}
			fmt.Printf("written %s\n", filepath.Join(dir, file.name))
			f.Close()
		}
	}

	if slices.Contains(cf.GetStringSlice("files"), "html") {
		var files []dataFile
		files, err = buildHTMLFiles(cf, data, run, inlineCSS)
		if err != nil {
			return err
		}
		for _, file := range files {
			var f *os.File
			f, err = os.Create(filepath.Join(dir, file.name))
			if err != nil {
				return
			}
			if _, err = io.Copy(f, file.content); err != nil {
				return
			}
			fmt.Printf("written %s\n", filepath.Join(dir, file.name))
			f.Close()
		}
	}

	if slices.Contains(cf.GetStringSlice("files"), "xlsx") {
		var files []dataFile
		files, err = buildXLSXFiles(cf, data, run)
		if err != nil {
			return err
		}

		for _, file := range files {
			var f *os.File
			f, err = os.Create(filepath.Join(dir, file.name))
			if err != nil {
				return
			}
			if _, err = io.Copy(f, file.content); err != nil {
				return
			}
			f.Close()
			fmt.Printf("written %s\n", filepath.Join(dir, file.name))
		}
	}
	return
}
