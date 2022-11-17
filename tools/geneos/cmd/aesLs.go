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

package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

var aesLSTabWriter *tabwriter.Writer
var aesLsCmdCSV, aesLsCmdJSON, aesLsCmdIndent bool

type aesLsCmdType struct {
	Type    string
	Name    string
	Host    string
	Keyfile string
	CRC32   string
	Modtime string
}

var aesLsCmdEntries []aesLsCmdType

func init() {
	aesCmd.AddCommand(aesLsCmd)

	aesLsCmd.PersistentFlags().BoolVarP(&aesLsCmdJSON, "json", "j", false, "Output JSON")
	aesLsCmd.PersistentFlags().BoolVarP(&aesLsCmdIndent, "pretty", "i", false, "Indent / pretty print JSON")
	aesLsCmd.PersistentFlags().BoolVarP(&aesLsCmdCSV, "csv", "c", false, "Output CSV")
	aesLsCmd.Flags().SortFlags = false
}

var aesLsCmd = &cobra.Command{
	Use:   "ls [flags] [TYPE] [NAME...]",
	Short: "List configured AES key files",
	Long: strings.ReplaceAll(`
List configured AES key files.

The AES key files are associated with instances (|gateway| instances 
to be precise) and typically carry the |.aes| file extension.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, args, params := cmdArgsParams(cmd)

		switch {
		case aesLsCmdJSON:
			aesLsCmdEntries = []aesLsCmdType{}
			err = instance.ForAll(ct, aesLSInstanceJSON, args, params)
			var b []byte
			if aesLsCmdIndent {
				b, _ = json.MarshalIndent(aesLsCmdEntries, "", "    ")
			} else {
				b, _ = json.Marshal(aesLsCmdEntries)
			}
			fmt.Println(string(b))
		case aesLsCmdCSV:
			csvWriter = csv.NewWriter(os.Stdout)
			csvWriter.Write([]string{"Type", "Name", "Host", "Keyfile", "CRC32", "Modtime"})
			err = instance.ForAll(ct, aesLSInstanceCSV, args, params)
			csvWriter.Flush()
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
		return
	}

	r, err := c.Host().Open(instance.Filepath(c, "keyfile"))
	if err != nil {
		return
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
		return
	}

	r, err := c.Host().Open(instance.Filepath(c, "keyfile"))
	if err != nil {
		return
	}
	defer r.Close()
	crc, err := config.Checksum(r)
	if err != nil {
		return
	}
	crcstr := fmt.Sprintf("%08X", crc)
	csvWriter.Write([]string{c.Type().String(), c.Name(), c.Host().String(), path, crcstr, s.ModTime().Format(time.RFC3339)})
	return
}

func aesLSInstanceJSON(c geneos.Instance, params []string) (err error) {
	path := instance.Filepath(c, "keyfile")
	if path == "" {
		return
	}
	s, err := c.Host().Stat(path)
	if err != nil {
		return
	}

	r, err := c.Host().Open(instance.Filepath(c, "keyfile"))
	if err != nil {
		return
	}
	defer r.Close()
	crc, err := config.Checksum(r)
	if err != nil {
		return
	}
	crcstr := fmt.Sprintf("%08X", crc)
	aesLsCmdEntries = append(aesLsCmdEntries, aesLsCmdType{c.Type().String(), c.Name(), c.Host().String(), path, crcstr, s.ModTime().Format(time.RFC3339)})
	return
}
