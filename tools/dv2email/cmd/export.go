/*
Copyright © 2023 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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
	"github.com/wneessen/go-mail"
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
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if exportCmdDir == "" {
			exportCmdDir = "."
		}

		gw, err := dialGateway(cf)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		data, err := fetchDataviews(cmd, gw, exportCmdFirstColumn, exportCmdHeadlines, exportCmdRows, exportCmdColumns, exportCmdRowOrder)
		if err != nil {
			return
		}
		if len(data.Dataviews) == 0 {
			fmt.Println("no matching dataviews")
			return
		}

		return writeFiles(exportCmdDir, data)
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
		m := mail.NewMsg()

		if err = buildHTMLAttachments(cf, m, data, run); err != nil {
			return err
		}

		files := m.GetAttachments()

		for _, file := range files {
			var f *os.File
			f, err = os.Create(filepath.Join(dir, file.Name))
			if err != nil {
				return
			}

			if _, err = file.Writer(f); err != nil {
				return
			}
			fmt.Printf("written %s\n", filepath.Join(dir, file.Name))
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
