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
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

var listCmdCSV, listCmdJSON, listCmdIndent, listCmdToolkit bool
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
	listCmd.Flags().BoolVarP(&listCmdToolkit, "toolkit", "t", false, "Output Toolkit formatted CSV")
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
		cmd.CmdAllowRoot:     "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, names, _, err := cmd.FetchArgs(command)
		if err != nil {
			return
		}

		h := geneos.GetHost(cmd.Hostname)

		switch {
		case listCmdJSON, listCmdIndent:
			if listCmdShared {
				results, _ := aesListSharedJSON(ct, h)
				results.Formatted(os.Stdout, "json", nil, nil, responses.IndentJSON(listCmdIndent))
			} else {
				instance.Do(h, ct, names, aesListInstanceJSON).Formatted(os.Stdout, "json", nil, nil, responses.IndentJSON(listCmdIndent))
			}
		case listCmdToolkit:
			w := csv.NewWriter(os.Stdout)
			w.Write([]string{
				"ID",
				"type",
				"name",
				"host",
				"keyfile",
				"crc32",
				"modtime",
			})

			if listCmdShared {
				aesListSharedCSV(ct, h).Report(w)
			} else {
				instance.Do(h, ct, names, aesListInstanceCSV).Report(w)
			}
		case listCmdCSV:
			w := csv.NewWriter(os.Stdout)
			w.Write([]string{
				"Type",
				"Name",
				"Host",
				"Keyfile",
				"CRC32",
				"Modtime",
			})

			if listCmdShared {
				aesListSharedCSV(ct, h).Report(w)
			} else {
				instance.Do(h, ct, names, aesListInstanceCSV).Report(w)
			}
		default:
			columns := []string{"Type", "Name", "Host", "Keyfile", "CRC32", "Modtime"}

			if listCmdShared {
				responses, _ := aesListShared(ct, h)
				responses.Formatted(os.Stdout, "column", columns, nil)
			} else {
				instance.Do(h, ct, names, aesListInstance).Formatted(os.Stdout, "column", columns, nil)
			}
		}
		return
	},
}

func aesListPath(ct *geneos.Component, h *geneos.Host, name string, keyfile config.KeyFile, resp *responses.General) {
	if keyfile == "" {
		return
	}

	s, err := h.Stat(keyfile.String())
	if err != nil {
		resp.Dataview.Table = append(resp.Dataview.Table, []string{
			ct.String(),
			name,
			h.String(),
			keyfile.String(),
			"-",
			"-",
		})
		return
	}

	crc, err := keyfile.ReadCRC(h)
	if err != nil {
		resp.Dataview.Table = append(resp.Dataview.Table, []string{
			ct.String(),
			name,
			h.String(),
			keyfile.String(),
			"-",
			s.ModTime().Format(time.RFC3339),
		})
		return
	}

	resp.Dataview.Table = append(resp.Dataview.Table, []string{
		ct.String(),
		name,
		h.String(),
		keyfile.String(),
		fmt.Sprintf("%08X", crc),
		s.ModTime().Format(time.RFC3339),
	})
}

func aesListInstance(i geneos.Instance, _ ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)

	path := config.KeyFile(instance.PathTo(i, "keyfile"))
	if path != "" {
		aesListPath(i.Type(), i.Host(), i.Name(), path, resp)
	}

	prev := config.KeyFile(instance.PathTo(i, "prevkeyfile"))
	if path != "" {
		aesListPath(i.Type(), i.Host(), i.Name()+" (prev)", prev, resp)
	}
	return
}

func aesListShared(ct *geneos.Component, h *geneos.Host) (results responses.GeneralResponses, err error) {
	results = make(responses.GeneralResponses)
	results["shared"] = responses.New[responses.General](nil)
	resp := results["shared"]
	for h := range h.OrList() {
		for ct := range ct.OrList(geneos.UsesKeyFiles()...) {
			var dirs []fs.DirEntry
			dirs, err = h.ReadDir(ct.Shared(h, "keyfiles"))
			if err != nil {
				return
			}
			for _, dir := range dirs {
				if dir.IsDir() || !strings.HasSuffix(dir.Name(), ".aes") {
					continue
				}
				aesListPath(ct, h, "shared", config.KeyFile(ct.Shared(h, "keyfiles", dir.Name())), resp)
			}
		}
	}
	return
}

func aesListPathCSV(ct *geneos.Component, h *geneos.Host, name, suffix string, kf config.KeyFile) (row []string) {
	if kf == "" {
		return
	}

	ts := ct.String()
	hs := h.String()
	id := ts + ":" + name + suffix
	if !listCmdToolkit {
		name += suffix
	}
	crcstr := "-"

	crc, crcerr := kf.ReadCRC(host.Localhost)
	if crcerr == nil {
		crcstr = fmt.Sprintf("%08X", crc)
	}

	if listCmdShared {
		id = ts + ":" + crcstr
	}

	if hs != geneos.LOCALHOST {
		id += "@" + hs
	}

	if listCmdToolkit {
		row = []string{id}
	}
	row = append(row, ts, name, hs, kf.String())

	s, err := h.Stat(kf.String())
	if err != nil {
		row = append(row, "-", "-")
		return
	}

	row = append(row, crcstr, s.ModTime().Format(time.RFC3339))
	return
}

func aesListInstanceCSV(i geneos.Instance, _ ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)

	path := config.KeyFile(instance.PathTo(i, "keyfile"))
	if path != "" {
		row := aesListPathCSV(i.Type(), i.Host(), i.Name(), "", path)
		resp.Dataview.Table = append(resp.Dataview.Table, row)
	}

	prev := config.KeyFile(instance.PathTo(i, "prevkeyfile"))
	if prev != "" {
		var row []string
		if listCmdToolkit {
			row = aesListPathCSV(i.Type(), i.Host(), i.Name(), " # prev", prev)
		} else {
			row = aesListPathCSV(i.Type(), i.Host(), i.Name(), " (prev)", prev)

		}
		resp.Dataview.Table = append(resp.Dataview.Table, row)
	}

	return
}

func aesListSharedCSV(ct *geneos.Component, h *geneos.Host) (rs responses.GeneralResponses) {
	rs = make(responses.GeneralResponses)
	var rows [][]string

	for h := range h.OrList() {
		for ct := range ct.OrList(geneos.UsesKeyFiles()...) {
			dirs, err := h.ReadDir(ct.Shared(h, "keyfiles"))
			if err != nil {
				continue
			}
			for _, dir := range dirs {
				if dir.IsDir() || !strings.HasSuffix(dir.Name(), ".aes") {
					continue
				}
				rows = append(rows, aesListPathCSV(ct, h, "shared", "", config.KeyFile(ct.Shared(h, "keyfiles", dir.Name()))))
			}
		}
	}
	rs["shared"] = &responses.General{Dataview: &responses.Dataview{Table: rows}}

	return
}

// aesListPathJSON fills in a listCmdType struct. paths is either one or
// two paths, the second being a previous keyfile.
func aesListPathJSON(i geneos.Instance, paths ...config.KeyFile) (resp *responses.General) {
	resp = responses.New[responses.General](i)

	// TODO: fixed shared list
	if i == nil {
		return
	}

	ct := i.Type()
	h := i.Host()
	name := i.Name()

	var keyfile, prevkeyfile *config.KeyFile
	results := []listCmdType{}

	if len(paths) == 0 {
		return
	}

	keyfile = &paths[0]
	if len(paths) > 1 {
		prevkeyfile = &paths[1]
	}

	if len(*keyfile) != 0 {
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

			crc, err := keyfile.ReadCRC(h)
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

			crc, err := prevkeyfile.ReadCRC(h)
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

func aesListSharedJSON(ct *geneos.Component, h *geneos.Host) (results responses.GeneralResponses, err error) {
	results = make(responses.GeneralResponses)
	var values []*responses.General

	for h := range h.OrList() {
		for ct := range ct.OrList(geneos.UsesKeyFiles()...) {
			var dirs []fs.DirEntry
			dirs, err = h.ReadDir(ct.Shared(h, "keyfiles"))
			if err != nil {
				return
			}
			for _, dir := range dirs {
				if dir.IsDir() || !strings.HasSuffix(dir.Name(), ".aes") {
					continue
				}
				resp := aesListPathJSON(nil, config.KeyFile(ct.Shared(h, "keyfiles", dir.Name())))
				if resp.Value != nil {
					values = append(values, resp)
				}
			}
		}
	}
	results["shared"] = &responses.General{Value: values}
	return
}

func aesListInstanceJSON(i geneos.Instance, _ ...any) (result *responses.General) {
	return aesListPathJSON(i, config.KeyFile(instance.PathTo(i, "keyfile")), config.KeyFile(instance.PathTo(i, "prevkeyfile")))
}
