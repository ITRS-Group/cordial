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
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var listCmdCSV, listCmdJSON, listCmdIndent bool
var listCmdShared bool

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
		cmd.CmdGlobal:        "true",
		cmd.CmdRequireHome:   "true",
		cmd.CmdWildcardNames: "true",
	},
	Run: func(command *cobra.Command, _ []string) {
		ct, names := cmd.ParseTypeNames(command)

		h := geneos.GetHost(cmd.Hostname)

		switch {
		case listCmdJSON, listCmdIndent:
			if listCmdShared {
				results, _ := aesListSharedJSON(ct, h)
				results.Write(os.Stdout, instance.WriterIndent(listCmdIndent))
			} else {
				instance.Do(h, ct, names, aesListInstanceJSON).Write(os.Stdout, instance.WriterIndent(listCmdIndent))
			}
		case listCmdCSV:
			aesListCSVWriter := csv.NewWriter(os.Stdout)
			aesListCSVWriter.Write([]string{"Type", "Name", "Host", "Keyfile", "CRC32", "Modtime"})

			if listCmdShared {
				instance.Responses{aesListSharedCSV(ct, h)}.Write(aesListCSVWriter)
			} else {
				instance.Do(h, ct, names, aesListInstanceCSV).Write(aesListCSVWriter)
			}
		default:
			aesListTabWriter := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(aesListTabWriter, "Type\tName\tHost\tKeyfile\tCRC32\tModtime\n")

			if listCmdShared {
				responses, _ := aesListShared(ct, h)
				responses.Write(aesListTabWriter)
			} else {
				instance.Do(h, ct, names, aesListInstance).Write(aesListTabWriter)
			}
		}
	},
}

func aesListPath(ct *geneos.Component, h *geneos.Host, name string, path config.KeyFile) (output string) {
	if path == "" {
		return
	}

	s, err := h.Stat(path.String())
	if err != nil {
		return fmt.Sprintf("%s\t%s\t%s\t%s\t-\t-", ct, name, h, path)
	}

	crc, _, err := path.ReadOrCreate(h, false)
	if err != nil {
		return fmt.Sprintf("%s\t%s\t%s\t%s\t-\t%s", ct, name, h, path, s.ModTime().Format(time.RFC3339))
	}
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%08X\t%s", ct, name, h, path, crc, s.ModTime().Format(time.RFC3339))
}

func aesListInstance(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	path := config.KeyFile(instance.PathOf(i, "keyfile"))
	if path != "" {
		resp.Lines = append(resp.Lines, aesListPath(i.Type(), i.Host(), i.Name(), path))
	}

	prev := config.KeyFile(instance.PathOf(i, "prevkeyfile"))
	if path != "" {
		resp.Lines = append(resp.Lines, aesListPath(i.Type(), i.Host(), i.Name()+" (prev)", prev))
	}
	return
}

func aesListShared(ct *geneos.Component, h *geneos.Host) (results instance.Responses, err error) {
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
				results = append(results, &instance.Response{
					Line: aesListPath(ct, h, "shared", config.KeyFile(ct.Shared(h, "keyfiles", dir.Name()))),
				})
			}
		}
	}
	return
}

func aesListPathCSV(ct *geneos.Component, h *geneos.Host, name string, path config.KeyFile) (row []string) {
	if path == "" {
		return
	}
	s, err := h.Stat(path.String())
	if err != nil {
		return []string{ct.String(), name, h.String(), path.String(), "-", "-"}
	}

	crc, _, err := path.ReadOrCreate(host.Localhost, false)
	if err != nil {
		return []string{ct.String(), name, h.String(), path.String(), "-", s.ModTime().Format(time.RFC3339)}
	}
	crcstr := fmt.Sprintf("%08X", crc)
	return []string{ct.String(), name, h.String(), path.String(), crcstr, s.ModTime().Format(time.RFC3339)}
}

func aesListInstanceCSV(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	path := config.KeyFile(instance.PathOf(i, "keyfile"))
	if path != "" {
		row := aesListPathCSV(i.Type(), i.Host(), i.Name(), path)
		resp.Rows = append(resp.Rows, row)
	}

	prev := config.KeyFile(instance.PathOf(i, "prevkeyfile"))
	if prev != "" {
		row := aesListPathCSV(i.Type(), i.Host(), i.Name()+" (prev)", prev)
		resp.Rows = append(resp.Rows, row)
	}

	return
}

func aesListSharedCSV(ct *geneos.Component, h *geneos.Host) (resp *instance.Response) {
	resp = instance.NewResponse(nil)

	for _, h := range h.OrList(geneos.AllHosts()...) {
		for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
			dirs, err := h.ReadDir(ct.Shared(h, "keyfiles"))
			if err != nil {
				continue
			}
			for _, dir := range dirs {
				if dir.IsDir() || !strings.HasSuffix(dir.Name(), ".aes") {
					continue
				}
				resp.Rows = append(resp.Rows, aesListPathCSV(ct, h, "shared", config.KeyFile(ct.Shared(h, "keyfiles", dir.Name()))))
			}
		}
	}
	return
}

// aesListPathJSON fills in a listCmdType struct. paths is either one or
// two paths, the second being a previous keyfile.
func aesListPathJSON(ct *geneos.Component, h *geneos.Host, name string, paths ...config.KeyFile) (resp *instance.Response) {
	resp = instance.NewResponse(nil)

	var keyfile, prevkeyfile *config.KeyFile
	results := []listCmdType{}

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

			crc, _, err := keyfile.ReadOrCreate(h, false)
			if err == nil {
				crcstr := fmt.Sprintf("%08X", crc)
				r.CRC32 = crcstr
			}
		}
		results = append(results, r)
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

			crc, _, err := prevkeyfile.ReadOrCreate(h, false)
			if err == nil {
				crcstr := fmt.Sprintf("%08X", crc)
				r.CRC32 = crcstr
			}
		}
		results = append(results, r)
	}

	if len(results) > 0 {
		resp.Value = results
	}
	return
}

func aesListSharedJSON(ct *geneos.Component, h *geneos.Host) (results instance.Responses, err error) {
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
				resp := aesListPathJSON(ct, h, "shared", config.KeyFile(ct.Shared(h, "keyfiles", dir.Name())))
				if resp.Value != nil {
					results = append(results, resp)
				}
			}
		}
	}
	return
}

func aesListInstanceJSON(i geneos.Instance, _ ...any) (result *instance.Response) {
	return aesListPathJSON(i.Type(), i.Host(), i.Name(), config.KeyFile(instance.PathOf(i, "keyfile")), config.KeyFile(instance.PathOf(i, "prevkeyfile")))
}
