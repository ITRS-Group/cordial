/*
Copyright © 2022 ITRS Group

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

package aescmd

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var aesListTabWriter *tabwriter.Writer
var listCmdCSV, listCmdJSON, listCmdIndent bool
var listCmdShared bool

var aesListCSVWriter *csv.Writer

type listCmdType struct {
	Name        string          `json:"name,omitempty"`
	Type        string          `json:"type,omitempty"`
	Host        string          `json:"host,omitempty"`
	Keyfile     *config.KeyFile `json:"keyfile,omitempty"`
	PrevKeyfile bool            `json:"prevkeyfile,omitempty"`
	CRC32       string          `json:"crc32,omitempty"`
	Modtime     string          `json:"modtime,omitempty"`
}

func init() {
	aesCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listCmdShared, "shared", "S", false, "List shared key files")

	listCmd.Flags().BoolVarP(&listCmdJSON, "json", "j", false, "Output JSON")
	listCmd.Flags().BoolVarP(&listCmdIndent, "pretty", "i", false, "Output indented JSON")
	listCmd.Flags().BoolVarP(&listCmdCSV, "csv", "c", false, "Output CSV")
	listCmd.Flags().SortFlags = false
}

//go:embed _docs/list.md
var listCmdDescription string

var listCmd = &cobra.Command{
	Use:   "list [flags] [TYPE] [NAME...]",
	Short: "List key files",
	Long:  listCmdDescription,
	Example: `
geneos aes list gateway
geneos aes ls -S gateway -H localhost -c
`,
	Aliases:      []string{"ls"},
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "true",
		cmd.AnnotationNeedsHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, names := cmd.TypeNames(command)

		h := geneos.GetHost(cmd.Hostname)

		switch {
		case listCmdJSON, listCmdIndent:
			var results []interface{}
			if listCmdShared {
				results, _ = aesListSharedJSON(ct, h)
			} else {
				results, err = instance.Do(h, ct, names, aesListInstanceJSON)
				if err != nil {
					return
				}
			}
			instance.ResultsAsJSON(os.Stdout, results, listCmdIndent)
		case listCmdCSV:
			aesListCSVWriter = csv.NewWriter(os.Stdout)
			aesListCSVWriter.Write([]string{"Type", "Name", "Host", "Keyfile", "CRC32", "Modtime"})
			if listCmdShared {
				var results [][]string
				results, err = aesListSharedCSV(ct, h)
				aesListCSVWriter.WriteAll(results)
			} else {
				var results []any
				results, err = instance.Do(h, ct, names, aesListInstanceCSV)
				if err != nil {
					return
				}
				if err = instance.ResultsToCSVWriter(aesListCSVWriter, results); err != nil {
					return
				}
			}
			aesListCSVWriter.Flush()
		default:
			var results []interface{}
			if listCmdShared {
				results, err = aesListShared(ct, h)
			} else {
				results, err = instance.Do(h, ct, names, aesListInstance)
			}
			if err != nil {
				return
			}
			aesListTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(aesListTabWriter, "Type\tName\tHost\tKeyfile\tCRC32\tModtime\n")
			for _, r := range results {
				fmt.Fprint(aesListTabWriter, r)
			}
			aesListTabWriter.Flush()
		}
		if err == os.ErrNotExist {
			err = nil
		}
		return
	},
}

func aesListPath(ct *geneos.Component, h *geneos.Host, name string, path config.KeyFile) (output string, err error) {
	if path == "" {
		return
	}

	s, err := h.Stat(path.String())
	if err != nil {
		return fmt.Sprintf("%s\t%s\t%s\t%s\t-\t-\n", ct, name, h, path), nil
	}

	crc, _, err := path.Check(false)
	if err != nil {
		return fmt.Sprintf("%s\t%s\t%s\t%s\t-\t%s\n", ct, name, h, path, s.ModTime().Format(time.RFC3339)), nil
	}
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%08X\t%s\n", ct, name, h, path, crc, s.ModTime().Format(time.RFC3339)), nil
}

func aesListInstance(c geneos.Instance) (result any, err error) {
	var output, outputprev string
	path := config.KeyFile(instance.PathOf(c, "keyfile"))
	prev := config.KeyFile(instance.PathOf(c, "prevkeyfile"))
	if path == "" {
		return
	}
	output, err = aesListPath(c.Type(), c.Host(), c.Name(), path)
	if err != nil {
		return
	}
	outputprev, err = aesListPath(c.Type(), c.Host(), c.Name()+" (prev)", prev)
	if err != nil {
		return
	}
	output += outputprev
	return output, nil
}

func aesListShared(ct *geneos.Component, h *geneos.Host) (results []interface{}, err error) {
	for _, h := range h.OrList(geneos.AllHosts()...) {
		for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
			var dirs []fs.DirEntry
			dirs, err = h.ReadDir(ct.Shared(h, "keyfiles"))
			if err != nil {
				return
			}
			for _, dir := range dirs {
				if dir.IsDir() || !strings.HasSuffix(dir.Name(), ".aes") {
					continue
				}
				output, err := aesListPath(ct, h, "shared", config.KeyFile(ct.Shared(h, "keyfiles", dir.Name())))
				if err != nil {
					continue
				}
				results = append(results, output)
			}
		}
	}
	return
}

func aesListPathCSV(ct *geneos.Component, h *geneos.Host, name string, path config.KeyFile) (result []string, err error) {
	if path == "" {
		return
	}
	s, err := h.Stat(path.String())
	if err != nil {
		return []string{ct.String(), name, h.String(), path.String(), "-", "-"}, nil
	}

	crc, _, err := path.Check(false)
	if err != nil {
		return []string{ct.String(), name, h.String(), path.String(), "-", s.ModTime().Format(time.RFC3339)}, nil
	}
	crcstr := fmt.Sprintf("%08X", crc)
	result = []string{ct.String(), name, h.String(), path.String(), crcstr, s.ModTime().Format(time.RFC3339)}
	return
}

func aesListInstanceCSV(c geneos.Instance) (results any, err error) {
	rows := [][]string{}
	var row []string

	path := config.KeyFile(instance.PathOf(c, "keyfile"))
	row, err = aesListPathCSV(c.Type(), c.Host(), c.Name(), path)
	if err != nil {
		return
	}
	rows = append(rows, row)

	prev := config.KeyFile(instance.PathOf(c, "prevkeyfile"))
	if prev != "" {
		row, err = aesListPathCSV(c.Type(), c.Host(), c.Name()+" (prev)", prev)
		if err != nil {
			return
		}
		rows = append(rows, row)
	}
	results = rows
	return
}

func aesListSharedCSV(ct *geneos.Component, h *geneos.Host) (results [][]string, err error) {
	for _, h := range h.OrList(geneos.AllHosts()...) {
		for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
			var dirs []fs.DirEntry
			dirs, err = h.ReadDir(ct.Shared(h, "keyfiles"))
			if err != nil {
				return
			}
			for _, dir := range dirs {
				if dir.IsDir() || !strings.HasSuffix(dir.Name(), ".aes") {
					continue
				}
				var result []string
				result, err = aesListPathCSV(ct, h, "shared", config.KeyFile(ct.Shared(h, "keyfiles", dir.Name())))
				if err != nil {
					return
				}
				results = append(results, result)
			}
		}
	}
	return
}

// aesListPathJSON fills in a listCmdType struct. paths is either one or
// two paths, the second being a previous keyfile.
func aesListPathJSON(ct *geneos.Component, h *geneos.Host, name string, paths ...config.KeyFile) (result []any, err error) {
	var keyfile, prevkeyfile *config.KeyFile

	if len(paths) == 0 {
		return
	}

	keyfile = &paths[0]
	if len(paths) > 1 {
		prevkeyfile = &paths[1]
	}

	if keyfile != nil && len(*keyfile) != 0 {
		r := listCmdType{
			Name:    name,
			Type:    ct.String(),
			Host:    h.String(),
			Keyfile: keyfile,
			CRC32:   "-",
			Modtime: "-",
		}
		s, err := h.Stat(keyfile.String())
		if err == nil {
			r.Modtime = s.ModTime().Format(time.RFC3339)

			crc, _, err := keyfile.Check(false)
			if err == nil {
				crcstr := fmt.Sprintf("%08X", crc)
				r.CRC32 = crcstr
			}
		}
		err = nil
		result = append(result, r)
	}

	if prevkeyfile != nil && len(*prevkeyfile) != 0 {
		r := listCmdType{
			Name:        name,
			Type:        ct.String(),
			Host:        h.String(),
			Keyfile:     prevkeyfile,
			PrevKeyfile: true,
			CRC32:       "-",
			Modtime:     "-",
		}

		s, err := h.Stat(prevkeyfile.String())
		if err == nil {
			r.Modtime = s.ModTime().Format(time.RFC3339)

			crc, _, err := prevkeyfile.Check(false)
			if err == nil {
				crcstr := fmt.Sprintf("%08X", crc)
				r.CRC32 = crcstr
			}
		}
		err = nil
		result = append(result, r)
	}

	return
}

func aesListSharedJSON(ct *geneos.Component, h *geneos.Host) (results []any, err error) {
	results = []interface{}{}
	for _, h := range h.OrList(geneos.AllHosts()...) {
		for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
			var dirs []fs.DirEntry
			dirs, err = h.ReadDir(ct.Shared(h, "keyfiles"))
			if err != nil {
				return
			}
			for _, dir := range dirs {
				if dir.IsDir() || !strings.HasSuffix(dir.Name(), ".aes") {
					continue
				}
				result, err := aesListPathJSON(ct, h, "shared", config.KeyFile(ct.Shared(h, "keyfiles", dir.Name())))
				if err != nil {
					continue
				}
				results = append(results, result...)
			}
		}
	}
	return
}

func aesListInstanceJSON(c geneos.Instance) (result any, err error) {
	keys := []any{}

	k, err := aesListPathJSON(c.Type(), c.Host(), c.Name(), config.KeyFile(instance.PathOf(c, "keyfile")), config.KeyFile(instance.PathOf(c, "prevkeyfile")))
	if err == nil {
		keys = append(keys, k...)
	}
	result = keys
	return
}
