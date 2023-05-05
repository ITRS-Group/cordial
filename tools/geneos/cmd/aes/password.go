/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package aes

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
		"wildcard": "false",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		var plaintext []byte

		crc, created, err := config.CheckKeyfile(cmd.DefaultUserKeyfile, true)
		if err != nil {
			return
		}
		crcstr := fmt.Sprintf("%08X", crc)

		if created {
			fmt.Printf("%s created, checksum %s\n", cmd.DefaultUserKeyfile, crcstr)
		}
		if aesPasswordCmdString != "" {
			plaintext = []byte(aesPasswordCmdString)
		} else if aesPasswordCmdSource != "" {
			plaintext, err = geneos.ReadFrom(aesPasswordCmdSource)
			if err != nil {
				return
			}
		} else {
			plaintext, err = config.PasswordPrompt(true, 3)
			if err != nil {
				return
			}
		}
		e, err := config.EncodeWithKeyfile(plaintext, cmd.DefaultUserKeyfile, true)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", e)
		return nil
	},
}
