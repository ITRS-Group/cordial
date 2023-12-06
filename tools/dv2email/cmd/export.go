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
)

//go:embed _docs/root.md
var exportCmdDescription string

var exportCmdDir string
var exportCmdFirstColumn, exportCmdHeadlines, exportCmdRows, exportCmdColumns, exportCmdRowOrder string

func init() {
	DV2EMAILCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVar(&exportCmdDir, "dir", "", "destination `directory`, defaults to current")

	exportCmd.Flags().StringVarP(&exportCmdFirstColumn, "rowname", "N", "", "set row `name`")
	exportCmd.Flags().StringVarP(&exportCmdHeadlines, "headlines", "H", "", "order and filter headlines, comma-separated")
	exportCmd.Flags().StringVarP(&exportCmdRows, "rows", "R", "", "filter rows, comma-separated")
	exportCmd.Flags().StringVarP(&exportCmdRowOrder, "order", "O", "", "order rows, comma-separated column names with optional '+'/'-' suffixes")
	exportCmd.Flags().StringVarP(&exportCmdColumns, "columns", "C", "", "order and filter columns, comma-separated")

	exportCmd.Flags().SortFlags = false
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
		data, err := fetchDataviews(gw, exportCmdFirstColumn, exportCmdHeadlines, exportCmdRows, exportCmdColumns, exportCmdRowOrder)

		if len(data.Dataviews) == 0 {
			fmt.Println("no matching dataviews")
			return
		}

		if err = writeFiles(exportCmdDir, data); err != nil {
			log.Fatal().Err(err).Msg("")
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
