/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package aescmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
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
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		var plaintext []byte

		crc, created, err := cmd.DefaultUserKeyfile.Check(true)
		if err != nil {
			return
		}

		if created {
			fmt.Printf("%s created, checksum %08X\n", cmd.DefaultUserKeyfile, crc)
		}

		if aesPasswordCmdString != "" {
			plaintext = []byte(aesPasswordCmdString)
		} else if aesPasswordCmdSource != "" {
			plaintext, err = geneos.ReadFrom(aesPasswordCmdSource)
			if err != nil {
				return
			}
		} else {
			plaintext, err = config.ReadPasswordInput(true, 3)
			if err != nil {
				return
			}
		}
		e, err := cmd.DefaultUserKeyfile.Encode(plaintext, true)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", e)
		return nil
	},
}
