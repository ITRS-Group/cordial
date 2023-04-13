/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/spf13/cobra"
)

var updateLsCmdHost string
var updateLsCmdJSON, updateLsCmdIndent, updateLsCmdCSV bool

var updateLsTabWriter *tabwriter.Writer
var updateLsCSVWriter *csv.Writer

func init() {
	updateCmd.AddCommand(updateLsCmd)

	updateLsCmd.Flags().StringVarP(&updateLsCmdHost, "host", "H", string(host.ALLHOSTS),
		`Apply only on remote host. "all" (the default) means all remote hosts and locally`)
	updateLsCmd.Flags().BoolVarP(&updateLsCmdJSON, "json", "j", false, "Output JSON")
	updateLsCmd.Flags().BoolVarP(&updateLsCmdIndent, "pretty", "i", false, "Output indented JSON")
	updateLsCmd.Flags().BoolVarP(&updateLsCmdCSV, "csv", "c", false, "Output CSV")

	updateLsCmd.Flags().SortFlags = false
}

var updateLsCmd = &cobra.Command{
	Use:   "ls [flags] [TYPE]",
	Short: "List packages available for update command",
	Long: strings.ReplaceAll(`
List the packages for the matching TYPE or all component types if no
TYPE is given. The |-H| flags restricts the check to a specific
remote host.

All timestamps are displayed in UTC to avoid filesystem confusion
between local summer/winter times in some locales.

Versions are listed in descending order for each component type, i.e.
|latest| is always the first entry for each component.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ct, _ := cmdArgs(cmd)

		h := host.Get(updateLsCmdHost)
		versions := []geneos.PackageDetails{}

		for _, h := range h.Range(host.AllHosts()...) {
			for _, ct := range ct.Range(geneos.RealComponents()...) {
				v, err := geneos.GetPackages(h, ct)
				if err != nil {
					return err
				}
				// append in reverse order
				for i := len(v) - 1; i >= 0; i-- {
					if v[i].Link == "" {
						v[i].Link = "-"
					}
					versions = append(versions, v[i])
				}
			}
		}

		switch {
		case updateLsCmdJSON, updateLsCmdIndent:
			var b []byte
			if updateLsCmdIndent {
				b, err = json.MarshalIndent(versions, "", "    ")
			} else {
				b, err = json.Marshal(versions)
			}
			if err != nil {
				return err
			}
			fmt.Println(string(b))
		case updateLsCmdCSV:
			updateLsCSVWriter = csv.NewWriter(os.Stdout)
			updateLsCSVWriter.Write([]string{"Component", "Host", "Version", "Latest", "Link", "LastModified", "Path"})
			for _, d := range versions {
				updateLsCSVWriter.Write([]string{d.Component, d.Host, d.Version, fmt.Sprintf("%v", d.Latest), d.Link, d.ModTime.Format(time.RFC3339), d.Path})
			}
			updateLsCSVWriter.Flush()
		default:
			updateLsTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(updateLsTabWriter, "Component\tHost\tVersion\tLink\tLastModified\tPath\n")
			for _, d := range versions {
				name := d.Version
				if d.Latest {
					name = fmt.Sprintf("%s (latest)", d.Version)
				}
				fmt.Fprintf(updateLsTabWriter, "%s\t%s\t%s\t%s\t%s\t%s\n", d.Component, d.Host, name, d.Link, d.ModTime.Format(time.RFC3339), d.Path)
			}
			updateLsTabWriter.Flush()
		}

		if err == os.ErrNotExist {
			err = nil
		}

		return
	},
}
