/*
Copyright © 2023 ITRS Group

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
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/spf13/cobra"
	"github.com/wneessen/go-mail"

	"github.com/itrs-group/cordial/pkg/config"
)

//go:embed _docs/export.md
var exportCmdDescription string

var exportCmdDir string
var exportCmdFirstColumn, exportCmdHeadlines, exportCmdRows, exportCmdColumns, exportCmdRowOrder string

func init() {
	Cmd.AddCommand(exportCmd)

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

		gw, err := dialGateway(globalCf)
		if err != nil {
			return fmt.Errorf("failed to dial gateway: %w", err)
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

// writeFile creates the file name in dir and copies r to it, returning
// any error from the copy or from closing the file
func writeFile(dir, name string, r io.Reader) (err error) {
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return
	}
	_, err = io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return
	}
	fmt.Printf("written %s\n", filepath.Join(dir, name))
	return
}

func writeFiles(dir string, data DV2EMailData) (err error) {
	run := time.Now()

	if err = os.MkdirAll(dir, 0775); err != nil {
		return
	}

	formats := config.Get[[]string](globalCf, "files")

	if slices.Contains(formats, "texttable") {
		var files []dataFile
		files, err = buildTextTableFiles(globalCf, data, run)
		if err != nil {
			return err
		}
		for _, file := range files {
			if err = writeFile(dir, file.name, file.content); err != nil {
				return
			}
		}
	}

	if slices.Contains(formats, "html") {
		m := mail.NewMsg()

		if err = buildHTMLAttachments(globalCf, m, data, run); err != nil {
			return err
		}

		for _, file := range m.GetAttachments() {
			var buf bytes.Buffer
			if _, err = file.Writer(&buf); err != nil {
				return
			}
			if err = writeFile(dir, file.Name, &buf); err != nil {
				return
			}
		}
	}

	if slices.Contains(formats, "xlsx") {
		var files []dataFile
		files, err = buildXLSXFiles(globalCf, data, run)
		if err != nil {
			return err
		}

		for _, file := range files {
			if err = writeFile(dir, file.name, file.content); err != nil {
				return
			}
		}
	}
	return
}
