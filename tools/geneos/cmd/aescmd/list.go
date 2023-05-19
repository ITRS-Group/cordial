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

var aesLSTabWriter *tabwriter.Writer
var aesLsCmdCSV, aesLsCmdJSON, aesLsCmdIndent bool
var aesLsCmdShared bool

var aesLsCSVWriter *csv.Writer

type aesLsCmdType struct {
	Name    string `json:"name,omitempty"`
	Type    string `json:"type,omitempty"`
	Host    string `json:"host,omitempty"`
	Keyfile string `json:"keyfile,omitempty"`
	CRC32   string `json:"crc32,omitempty"`
	Modtime string `json:"modtime,omitempty"`
}

func init() {
	aesCmd.AddCommand(aesLsCmd)

	aesLsCmd.Flags().BoolVarP(&aesLsCmdShared, "shared", "S", false, "List shared key files")

	aesLsCmd.Flags().BoolVarP(&aesLsCmdJSON, "json", "j", false, "Output JSON")
	aesLsCmd.Flags().BoolVarP(&aesLsCmdIndent, "pretty", "i", false, "Output indented JSON")
	aesLsCmd.Flags().BoolVarP(&aesLsCmdCSV, "csv", "c", false, "Output CSV")
	aesLsCmd.Flags().SortFlags = false
}

var aesLsCmd = &cobra.Command{
	Use:   "ls [flags] [TYPE] [NAME...]",
	Short: "List key files",
	Long: strings.ReplaceAll(`
List details of the key files referenced by matching instances.

If given the |--shared|/|-S| flag then the key files in the shared
component directory are listed. This can be filtered by host with the
|--host|/|-H| and/or by component TYPE.

The default output is human-readable table format. You can select CSV
or JSON formats using the appropriate flags.
`, "|", "`"),
	Example: `
geneos aes ls gateway
geneos aes ls -S gateway -H localhost -c
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.CmdArgsParams(command)

		h := geneos.GetHost(cmd.Hostname)
		if aesLsCmdShared {

		}
		switch {
		case aesLsCmdJSON, aesLsCmdIndent:
			var results []interface{}
			if aesLsCmdShared {
				results, _ = aesLSSharedJSON(ct, h)
			} else {
				results, _ = instance.ForAllWithResults(ct, aesLSInstanceJSON, args, params)
			}
			var b []byte
			if aesLsCmdIndent {
				b, _ = json.MarshalIndent(results, "", "    ")
			} else {
				b, _ = json.Marshal(results)
			}
			fmt.Println(string(b))
		case aesLsCmdCSV:
			aesLsCSVWriter = csv.NewWriter(os.Stdout)
			aesLsCSVWriter.Write([]string{"Type", "Name", "Host", "Keyfile", "CRC32", "Modtime"})
			if aesLsCmdShared {
				aesLSSharedCSV(ct, h)
			} else {
				err = instance.ForAll(ct, cmd.Hostname, aesLSInstanceCSV, args, params)
			}
			aesLsCSVWriter.Flush()
		default:
			aesLSTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(aesLSTabWriter, "Type\tName\tHost\tKeyfile\tCRC32\tModtime\n")
			if aesLsCmdShared {
				aesLSShared(ct, h)
			} else {
				instance.ForAll(ct, cmd.Hostname, aesLSInstance, args, params)
			}
			aesLSTabWriter.Flush()
		}
		if err == os.ErrNotExist {
			err = nil
		}
		return
	},
}

func aesLSPath(ct *geneos.Component, h *geneos.Host, name string, path string) (err error) {
	if path == "" {
		return
	}
	s, err := h.Stat(path)
	if err != nil {
		fmt.Fprintf(aesLSTabWriter, "%s\t%s\t%s\t%s\t-\t-\n", ct, name, h, path)
		return nil
	}

	r, err := h.Open(path)
	if err != nil {
		fmt.Fprintf(aesLSTabWriter, "%s\t%s\t%s\t%s\t-\t%s\n", ct, name, h, path, s.ModTime().Format(time.RFC3339))
		return nil
	}
	defer r.Close()
	crc, err := config.Checksum(r)
	if err != nil {
		return
	}
	fmt.Fprintf(aesLSTabWriter, "%s\t%s\t%s\t%s\t%08X\t%s\n", ct, name, h, path, crc, s.ModTime().Format(time.RFC3339))
	return
}

func aesLSInstance(c geneos.Instance, params []string) (err error) {
	path := instance.Filepath(c, "keyfile")
	prev := instance.Filepath(c, "prevkeyfile")
	if path == "" {
		return
	}
	aesLSPath(c.Type(), c.Host(), c.Name(), path)
	aesLSPath(c.Type(), c.Host(), c.Name()+"**", prev)
	return
}

func aesLSShared(ct *geneos.Component, h *geneos.Host) (err error) {
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
				aesLSPath(ct, h, "shared", ct.SharedPath(h, "keyfiles", dir.Name()))
			}
		}
	}
	return
}

func aesLSPathCSV(ct *geneos.Component, h *geneos.Host, name string, path string) (err error) {
	if path == "" {
		return
	}
	s, err := h.Stat(path)
	if err != nil {
		aesLsCSVWriter.Write([]string{ct.String(), name, h.String(), path, "-", "-"})
		return nil
	}

	r, err := h.Open(path)
	if err != nil {
		aesLsCSVWriter.Write([]string{ct.String(), name, h.String(), path, "-", s.ModTime().Format(time.RFC3339)})
		return nil
	}
	defer r.Close()
	crc, err := config.Checksum(r)
	if err != nil {
		return
	}
	crcstr := fmt.Sprintf("%08X", crc)
	aesLsCSVWriter.Write([]string{ct.String(), name, h.String(), path, crcstr, s.ModTime().Format(time.RFC3339)})
	return
}

func aesLSInstanceCSV(c geneos.Instance, params []string) (err error) {
	path := instance.Filepath(c, "keyfile")
	prev := instance.Filepath(c, "prevkeyfile")
	aesLSPathCSV(c.Type(), c.Host(), c.Name(), path)
	aesLSPathCSV(c.Type(), c.Host(), c.Name()+"**", prev)
	return
}

func aesLSSharedCSV(ct *geneos.Component, h *geneos.Host) (err error) {
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
				aesLSPathCSV(ct, h, "shared", ct.SharedPath(h, "keyfiles", dir.Name()))
			}
		}
	}
	return
}

func aesLSPathJSON(ct *geneos.Component, h *geneos.Host, name string, path string) (result interface{}, err error) {
	if path == "" {
		return
	}
	s, err := h.Stat(path)
	if err != nil {
		err = nil
		result = aesLsCmdType{
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
		result = aesLsCmdType{
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
	result = aesLsCmdType{
		Name:    name,
		Type:    ct.String(),
		Host:    h.String(),
		Keyfile: path,
		CRC32:   crcstr,
		Modtime: s.ModTime().Format(time.RFC3339),
	}

	return
}

func aesLSSharedJSON(ct *geneos.Component, h *geneos.Host) (results []interface{}, err error) {
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
				result, err := aesLSPathJSON(ct, h, "shared", ct.SharedPath(h, "keyfiles", dir.Name()))
				if err != nil {
					continue
				}
				results = append(results, result)
			}
		}
	}
	return
}

func aesLSInstanceJSON(c geneos.Instance, params []string) (result interface{}, err error) {
	path := instance.Filepath(c, "keyfile")
	return aesLSPathJSON(c.Type(), c.Host(), c.Name(), path)
}
