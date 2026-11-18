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

package cmd

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/logger"
)

var cfgFile, confDir string

// ldapVerify enables the verification of the LDAP server certificate,
// which is skipped by default
var ldapVerify bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sso-diag",
	Short: "Diagnose a Geneos SSO Agent configuration",
	Long: `sso-diag reads the configuration of a Geneos SSO Agent, from the conf/sso-agent.conf
file in the SSO Agent directory, and runs a diagnostic server that uses the same
Kerberos and LDAP settings. Use the run command to start the server, then
browse to or use curl with SPNEGO against its /testuser endpoint to see the
results of authenticating a user and looking them up in LDAP.

Settings for sso-diag itself can be given as command line flags or in an
optional YAML configuration file (see --conf), with the keys:

  dir: SSO Agent directory
  ldap-verify: true | false`,
	Version: cordial.VERSION,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var execname string

// defaultConfigDirs returns the directories searched for the sso-diag
// configuration file, `sso-diag.yaml`, if none is given on the command
// line
func defaultConfigDirs() (dirs []string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	return []string{filepath.Join(home, ".config", "geneos")}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "conf", "c", "", "config file (default is $HOME/.config/geneos/sso-diag.yaml)")
	rootCmd.PersistentFlags().StringVarP(&confDir, "dir", "d", ".", "sso-agent directory for relative path resolution")
	rootCmd.PersistentFlags().BoolVar(&ldapVerify, "ldap-verify", false, "verify the LDAP server certificate (not verified by default)")

	execname = filepath.Base(os.Args[0])
	logger.Init(execname)
}

// initConfig reads in config file and ENV variables if set. Values from
// the file are only used for settings not given on the command line.
func initConfig() {
	viper.SetConfigType("yaml")
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("sso-diag")
		for _, dir := range defaultConfigDirs() {
			viper.AddConfigPath(dir)
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in. A file that was asked for
	// but cannot be read is an error, but not finding the default is not.
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if cfgFile != "" || !errors.As(err, &notFound) {
			cobra.CheckErr(err)
		}
		return
	}
	log.Info("using config file", slog.String("file", viper.ConfigFileUsed()))

	if f := rootCmd.PersistentFlags().Lookup("dir"); f != nil && !f.Changed && viper.IsSet("dir") {
		confDir = viper.GetString("dir")
	}
	if f := rootCmd.PersistentFlags().Lookup("ldap-verify"); f != nil && !f.Changed && viper.IsSet("ldap-verify") {
		ldapVerify = viper.GetBool("ldap-verify")
	}
}
