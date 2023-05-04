/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package aes

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/spf13/cobra"
)

var aesPasswordCmdString, aesPasswordCmdSource string

func init() {
	AesCmd.AddCommand(aesPasswordCmd)

	aesPasswordCmd.Flags().StringVarP(&aesPasswordCmdString, "password", "p", "", "Password string to use")
	aesPasswordCmd.Flags().StringVarP(&aesPasswordCmdSource, "source", "s", "", "Source for password to use")
}

// aesPasswordCmd represents the password command
var aesPasswordCmd = &cobra.Command{
	Use:   "password [flags]",
	Short: "Encode a password using user's keyfile",
	Long: strings.ReplaceAll(`
Encode a password using the user's keyfile. If no keyfile exists it
is created. Output is in |Expand| format.

User is prompted to enter the password (twice, for validation) unless
on of the flags is set.
`, "|", "`"),
	Aliases:      []string{"passwd"},
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var plaintext []byte
		var match bool

		// check for existing keyfile, create if none
		if _, err := os.Stat(aesDefaultKeyfile); err != nil && err == fs.ErrNotExist {
			a, err := config.NewAESValues()
			if err != nil {
				return err
			}
			if err = os.MkdirAll(filepath.Dir(aesDefaultKeyfile), 0775); err != nil {
				return fmt.Errorf("failed to create keyfile directory %q: %w", filepath.Dir(aesDefaultKeyfile), err)
			}
			if err = os.WriteFile(aesDefaultKeyfile, []byte(a.String()), 0600); err != nil {
				return fmt.Errorf("failed to write keyfile to %q: %w", aesDefaultKeyfile, err)
			}
			var crc uint32

			crc, err = config.ChecksumString(a.String())
			if err != nil {
				return err
			}
			crcstr := fmt.Sprintf("%08X", crc)

			if aesNewCmdKeyfile != "" {
				fmt.Printf("%s created, checksum %s\n", aesDefaultKeyfile, crcstr)
			}
		}
		if aesPasswordCmdString != "" {
			plaintext = []byte(aesPasswordCmdString)
		} else if aesPasswordCmdSource != "" {
			plaintext, err = geneos.ReadFrom(aesPasswordCmdSource)
			if err != nil {
				return
			}
		} else {
			for i := 0; i < 3; i++ {
				plaintext = config.ReadPasswordPrompt()
				plaintext2 := config.ReadPasswordPrompt("Re-enter Password")
				if bytes.Equal(plaintext, plaintext2) {
					match = true
					break
				}
				fmt.Println("Passwords do not match. Please try again.")
			}
			if !match {
				return fmt.Errorf("too many attempts, giving up")
			}
		}
		e, err := config.EncodeWithKeyfile(plaintext, aesDefaultKeyfile, true)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", e)
		return nil
	},
}
