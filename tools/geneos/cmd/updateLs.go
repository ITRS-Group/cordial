/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/spf13/cobra"
)

var updateLsCmdHost string
var updateLsCmdLocal, updateLsCmdJSON, updateLsCmdIndent, updateLsCmdCSV bool

var updateLsTabWriter *tabwriter.Writer
var updateLsCSVWriter *csv.Writer

type packageDirDetails struct {
	Component string    `json:"Component"`
	Host      string    `json:"Host"`
	Version   string    `json:"Version"`
	Latest    bool      `json:"Latest,string"`
	Link      string    `json:"Link,omitempty"`
	ModTime   time.Time `json:"LastModified"`
	Path      string    `json:"Path"`
}

func init() {
	updateCmd.AddCommand(updateLsCmd)

	updateLsCmd.Flags().StringVarP(&updateLsCmdHost, "host", "H", string(host.ALLHOSTS),
		`Apply only on remote host. "all" (the default) means all remote hosts and locally`)
	updateLsCmd.Flags().BoolVarP(&updateLsCmdLocal, "local", "L", false, "Display local times")
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

The |-L| flags shows directory times in the local time zone. This is
not the default as the display may jump between summer and winter
times between releases and give a confusing view.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ct, _ := cmdArgs(cmd)

		h := host.Get(updateLsCmdHost)
		versions := []packageDirDetails{}

		for _, h := range h.Range(host.AllHosts()...) {
			for _, ct := range ct.Range(componentsWithKeyfiles...) {
				basedir := h.Filepath("packages", ct)
				ents, err := h.ReadDir(basedir)
				if err != nil && !errors.Is(err, fs.ErrNotExist) {
					return err
				}

				var links = make(map[string]string)
				for _, ent := range ents {
					einfo, err := ent.Info()
					if err != nil {
						return err
					}
					if einfo.Mode()&fs.ModeSymlink != 0 {
						link, err := h.Readlink(filepath.Join(basedir, ent.Name()))
						if err != nil {
							return err
						}
						links[link] = ent.Name()
					}
				}

				latest, _ := geneos.LatestRelease(h, basedir, "", func(d os.DirEntry) bool { // ignore error, empty is valid
					return !d.IsDir()
				})

				for _, ent := range ents {
					if ent.IsDir() {
						einfo, err := ent.Info()
						if err != nil {
							return err
						}
						link := links[ent.Name()]
						if link == "" {
							link = "-"
						}
						mtime := einfo.ModTime().UTC()
						if updateLsCmdLocal {
							mtime = mtime.Local()
						}
						versions = append(versions, packageDirDetails{
							Component: ct.String(),
							Host:      h.String(),
							Version:   ent.Name(),
							Latest:    ent.Name() == latest,
							Link:      link,
							ModTime:   mtime,
							Path:      filepath.Join(basedir, ent.Name()),
						})

					}
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
