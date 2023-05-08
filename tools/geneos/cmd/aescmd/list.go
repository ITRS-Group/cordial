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
	AesCmd.AddCommand(aesLsCmd)

	aesLsCmd.PersistentFlags().BoolVarP(&aesLsCmdJSON, "json", "j", false, "Output JSON")
	aesLsCmd.PersistentFlags().BoolVarP(&aesLsCmdIndent, "pretty", "i", false, "Output indented JSON")
	aesLsCmd.PersistentFlags().BoolVarP(&aesLsCmdCSV, "csv", "c", false, "Output CSV")
	aesLsCmd.Flags().SortFlags = false
}

var aesLsCmd = &cobra.Command{
	Use:   "ls [flags] [TYPE] [NAME...]",
	Short: "List configured keyfiles for instances",
	Long: strings.ReplaceAll(`
For matching instances list configured keyfiles, their location in
the filesystem and their CRC. 

The default output is human readable columns but can be in CSV
format using the |-c| flag or JSON with the |-j| or |-i| flags, the
latter "pretty" formatting the output over multiple, indented lines
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.CmdArgsParams(command)

		switch {
		case aesLsCmdJSON, aesLsCmdIndent:
			results, _ := instance.ForAllWithResults(ct, aesLSInstanceJSON, args, params)
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
			err = instance.ForAll(ct, aesLSInstanceCSV, args, params)
			aesLsCSVWriter.Flush()
		default:
			aesLSTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
			fmt.Fprintf(aesLSTabWriter, "Type\tName\tHost\tKeyfile\tCRC32\tModtime\n")
			instance.ForAll(ct, aesLSInstance, args, params)
			aesLSTabWriter.Flush()
		}
		if err == os.ErrNotExist {
			err = nil
		}
		return
	},
}

func aesLSInstance(c geneos.Instance, params []string) (err error) {
	path := instance.Filepath(c, "keyfile")
	if path == "" {
		return
	}
	s, err := c.Host().Stat(path)
	if err != nil {
		fmt.Fprintf(aesLSTabWriter, "%s\t%s\t%s\t%s\t-\t-\n", c.Type(), c.Name(), c.Host(), path)
		return nil
	}

	r, err := c.Host().Open(path)
	if err != nil {
		fmt.Fprintf(aesLSTabWriter, "%s\t%s\t%s\t%s\t-\t%s\n", c.Type(), c.Name(), c.Host(), path, s.ModTime().Format(time.RFC3339))
		return nil
	}
	defer r.Close()
	crc, err := config.Checksum(r)
	if err != nil {
		return
	}
	fmt.Fprintf(aesLSTabWriter, "%s\t%s\t%s\t%s\t%08X\t%s\n", c.Type(), c.Name(), c.Host(), path, crc, s.ModTime().Format(time.RFC3339))
	return
}

func aesLSInstanceCSV(c geneos.Instance, params []string) (err error) {
	path := instance.Filepath(c, "keyfile")
	if path == "" {
		return
	}
	s, err := c.Host().Stat(path)
	if err != nil {
		aesLsCSVWriter.Write([]string{c.Type().String(), c.Name(), c.Host().String(), path, "-", "-"})
		return nil
	}

	r, err := c.Host().Open(instance.Filepath(c, "keyfile"))
	if err != nil {
		aesLsCSVWriter.Write([]string{c.Type().String(), c.Name(), c.Host().String(), path, "-", s.ModTime().Format(time.RFC3339)})
		return nil
	}
	defer r.Close()
	crc, err := config.Checksum(r)
	if err != nil {
		return
	}
	crcstr := fmt.Sprintf("%08X", crc)
	aesLsCSVWriter.Write([]string{c.Type().String(), c.Name(), c.Host().String(), path, crcstr, s.ModTime().Format(time.RFC3339)})
	return
}

func aesLSInstanceJSON(c geneos.Instance, params []string) (result interface{}, err error) {
	path := instance.Filepath(c, "keyfile")
	if path == "" {
		return
	}
	s, err := c.Host().Stat(path)
	if err != nil {
		err = nil
		result = aesLsCmdType{
			Name:    c.Name(),
			Type:    c.Type().String(),
			Host:    c.Host().String(),
			Keyfile: path,
			CRC32:   "-",
			Modtime: "-",
		}
		return
	}

	r, err := c.Host().Open(instance.Filepath(c, "keyfile"))
	if err != nil {
		err = nil
		result = aesLsCmdType{
			Name:    c.Name(),
			Type:    c.Type().String(),
			Host:    c.Host().String(),
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
		Name:    c.Name(),
		Type:    c.Type().String(),
		Host:    c.Host().String(),
		Keyfile: path,
		CRC32:   crcstr,
		Modtime: s.ModTime().Format(time.RFC3339),
	}
	return
}
