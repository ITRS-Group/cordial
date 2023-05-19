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
	"encoding/json"
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
var aesListCmdCSV, aesListCmdJSON, aesListCmdIndent bool
var aesListCmdShared bool

var aesListCSVWriter *csv.Writer

type aesListCmdType struct {
	Name    string `json:"name,omitempty"`
	Type    string `json:"type,omitempty"`
	Host    string `json:"host,omitempty"`
	Keyfile string `json:"keyfile,omitempty"`
	CRC32   string `json:"crc32,omitempty"`
	Modtime string `json:"modtime,omitempty"`
}

func init() {
	aesCmd.AddCommand(aesListCmd)

	aesListCmd.Flags().BoolVarP(&aesListCmdShared, "shared", "S", false, "List shared key files")

	aesListCmd.Flags().BoolVarP(&aesListCmdJSON, "json", "j", false, "Output JSON")
	aesListCmd.Flags().BoolVarP(&aesListCmdIndent, "pretty", "i", false, "Output indented JSON")
	aesListCmd.Flags().BoolVarP(&aesListCmdCSV, "csv", "c", false, "Output CSV")
	aesListCmd.Flags().SortFlags = false
}

//go:embed _docs/list.md
var aesListCmdDescription string

var aesListCmd = &cobra.Command{
	Use:   "list [flags] [TYPE] [NAME...]",
	Short: "List key files",
	Long:  aesListCmdDescription,
	Example: `
geneos aes list gateway
geneos aes ls -S gateway -H localhost -c
`,
	Aliases:      []string{"ls"},
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.CmdArgsParams(command)

		h := geneos.GetHost(cmd.Hostname)
		if aesListCmdShared {

		}
		switch {
		case aesListCmdJSON, aesListCmdIndent:
			var results []interface{}
			if aesListCmdShared {
				results, _ = aesListSharedJSON(ct, h)
			} else {
				results, _ = instance.ForAllWithResults(ct, aesListInstanceJSON, args, params)
			}
			var b []byte
			if aesListCmdIndent {
				b, _ = json.MarshalIndent(results, "", "    ")
			} else {
				b, _ = json.Marshal(results)
			}
			fmt.Println(string(b))
		case aesListCmdCSV:
			aesListCSVWriter = csv.NewWriter(os.Stdout)
			aesListCSVWriter.Write([]string{"Type", "Name", "Host", "Keyfile", "CRC32", "Modtime"})
			if aesListCmdShared {
				aesListSharedCSV(ct, h)
			} else {
				err = instance.ForAll(ct, cmd.Hostname, aesListInstanceCSV, args, params)
			}
			aesListCSVWriter.Flush()
		default:
			aesListTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(aesListTabWriter, "Type\tName\tHost\tKeyfile\tCRC32\tModtime\n")
			if aesListCmdShared {
				aesListShared(ct, h)
			} else {
				instance.ForAll(ct, cmd.Hostname, aesListInstance, args, params)
			}
			aesListTabWriter.Flush()
		}
		if err == os.ErrNotExist {
			err = nil
		}
		return
	},
}

func aesListPath(ct *geneos.Component, h *geneos.Host, name string, path string) (err error) {
	if path == "" {
		return
	}
	s, err := h.Stat(path)
	if err != nil {
		fmt.Fprintf(aesListTabWriter, "%s\t%s\t%s\t%s\t-\t-\n", ct, name, h, path)
		return nil
	}

	r, err := h.Open(path)
	if err != nil {
		fmt.Fprintf(aesListTabWriter, "%s\t%s\t%s\t%s\t-\t%s\n", ct, name, h, path, s.ModTime().Format(time.RFC3339))
		return nil
	}
	defer r.Close()
	crc, err := config.Checksum(r)
	if err != nil {
		return
	}
	fmt.Fprintf(aesListTabWriter, "%s\t%s\t%s\t%s\t%08X\t%s\n", ct, name, h, path, crc, s.ModTime().Format(time.RFC3339))
	return
}

func aesListInstance(c geneos.Instance, params []string) (err error) {
	path := instance.Filepath(c, "keyfile")
	prev := instance.Filepath(c, "prevkeyfile")
	if path == "" {
		return
	}
	aesListPath(c.Type(), c.Host(), c.Name(), path)
	aesListPath(c.Type(), c.Host(), c.Name()+"**", prev)
	return
}

func aesListShared(ct *geneos.Component, h *geneos.Host) (err error) {
	for _, ct := range ct.OrList(componentsWithKeyfiles...) {
		for _, h := range h.OrList(geneos.AllHosts()...) {
			var dirs []fs.DirEntry
			dirs, err = h.ReadDir(ct.SharedPath(h, "keyfiles"))
			if err != nil {
				return
			}
			for _, dir := range dirs {
				if dir.IsDir() || !strings.HasSuffix(dir.Name(), ".aes") {
					continue
				}
				aesListPath(ct, h, "shared", ct.SharedPath(h, "keyfiles", dir.Name()))
			}
		}
	}
	return
}

func aesListPathCSV(ct *geneos.Component, h *geneos.Host, name string, path string) (err error) {
	if path == "" {
		return
	}
	s, err := h.Stat(path)
	if err != nil {
		aesListCSVWriter.Write([]string{ct.String(), name, h.String(), path, "-", "-"})
		return nil
	}

	r, err := h.Open(path)
	if err != nil {
		aesListCSVWriter.Write([]string{ct.String(), name, h.String(), path, "-", s.ModTime().Format(time.RFC3339)})
		return nil
	}
	defer r.Close()
	crc, err := config.Checksum(r)
	if err != nil {
		return
	}
	crcstr := fmt.Sprintf("%08X", crc)
	aesListCSVWriter.Write([]string{ct.String(), name, h.String(), path, crcstr, s.ModTime().Format(time.RFC3339)})
	return
}

func aesListInstanceCSV(c geneos.Instance, params []string) (err error) {
	path := instance.Filepath(c, "keyfile")
	prev := instance.Filepath(c, "prevkeyfile")
	aesListPathCSV(c.Type(), c.Host(), c.Name(), path)
	aesListPathCSV(c.Type(), c.Host(), c.Name()+"**", prev)
	return
}

func aesListSharedCSV(ct *geneos.Component, h *geneos.Host) (err error) {
	for _, ct := range ct.OrList(componentsWithKeyfiles...) {
		for _, h := range h.OrList(geneos.AllHosts()...) {
			var dirs []fs.DirEntry
			dirs, err = h.ReadDir(ct.SharedPath(h, "keyfiles"))
			if err != nil {
				return
			}
			for _, dir := range dirs {
				if dir.IsDir() || !strings.HasSuffix(dir.Name(), ".aes") {
					continue
				}
				aesListPathCSV(ct, h, "shared", ct.SharedPath(h, "keyfiles", dir.Name()))
			}
		}
	}
	return
}

func aesListPathJSON(ct *geneos.Component, h *geneos.Host, name string, path string) (result interface{}, err error) {
	if path == "" {
		return
	}
	s, err := h.Stat(path)
	if err != nil {
		err = nil
		result = aesListCmdType{
			Name:    name,
			Type:    ct.String(),
			Host:    h.String(),
			Keyfile: path,
			CRC32:   "-",
			Modtime: "-",
		}
		return
	}

	r, err := h.Open(path)
	if err != nil {
		err = nil
		result = aesListCmdType{
			Name:    name,
			Type:    ct.String(),
			Host:    h.String(),
			Keyfile: path,
			CRC32:   "-",
			Modtime: s.ModTime().Format(time.RFC3339),
		}
		return
	}
	defer r.Close()
	crc, err := config.Checksum(r)
	if err != nil {
		return
	}
	crcstr := fmt.Sprintf("%08X", crc)
	result = aesListCmdType{
		Name:    name,
		Type:    ct.String(),
		Host:    h.String(),
		Keyfile: path,
		CRC32:   crcstr,
		Modtime: s.ModTime().Format(time.RFC3339),
	}

	return
}

func aesListSharedJSON(ct *geneos.Component, h *geneos.Host) (results []interface{}, err error) {
	results = []interface{}{}
	for _, ct := range ct.OrList(componentsWithKeyfiles...) {
		for _, h := range h.OrList(geneos.AllHosts()...) {
			var dirs []fs.DirEntry
			dirs, err = h.ReadDir(ct.SharedPath(h, "keyfiles"))
			if err != nil {
				return
			}
			for _, dir := range dirs {
				if dir.IsDir() || !strings.HasSuffix(dir.Name(), ".aes") {
					continue
				}
				result, err := aesListPathJSON(ct, h, "shared", ct.SharedPath(h, "keyfiles", dir.Name()))
				if err != nil {
					continue
				}
				results = append(results, result)
			}
		}
	}
	return
}

func aesListInstanceJSON(c geneos.Instance, params []string) (result interface{}, err error) {
	path := instance.Filepath(c, "keyfile")
	return aesListPathJSON(c.Type(), c.Host(), c.Name(), path)
}
