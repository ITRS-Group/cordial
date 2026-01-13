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

package tlscmd

import (
	"crypto/x509"
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

var migrateCmdDays int
var migrateCmdNewKey, migrateCmdPrepare, migrateCmdRoll, migrateCmdUnroll bool

func init() {
	tlsCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().BoolVarP(&migrateCmdPrepare, "prepare", "P", false, "Prepare migration without changing existing files")
	migrateCmd.Flags().BoolVarP(&migrateCmdRoll, "roll", "R", false, "Roll previously prepared migrated files and backup existing ones")
	migrateCmd.Flags().BoolVarP(&migrateCmdUnroll, "unroll", "U", false, "Unroll previously rolled migrated files to earlier backups")

	migrateCmd.MarkFlagsMutuallyExclusive("prepare", "roll", "unroll")
}

//go:embed _docs/migrate.md
var migrateCmdDescription string

var migrateCmd = &cobra.Command{
	Use:          "migrate [TYPE] [NAME...]",
	Short:        "Migrate certificates and related files to the updated layout",
	Long:         migrateCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:        "true",
		cmd.CmdRequireHome:   "true",
		cmd.CmdWildcardNames: "true",
	},
	Run: func(command *cobra.Command, _ []string) {
		ct, names := cmd.ParseTypeNames(command)
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, migrateInstance).Report(os.Stdout)
	},
}

// migrate instance TLS from old layout to new layout:
//
// * use certchain to create a full chain in certificate file (without root)
// * update params from certificate/privatekey/certchain to tls::certificate etc.
// * update trusted-roots with new roots found in certchain
//
// Private key file is unchanged, but the parameter is moved.
func migrateInstance(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	cf := i.Config()
	h := i.Host()

	// first check if already migrated
	if cf.IsSet(cf.Join("tls", "certificate")) {
		return
	}

	// some instances may already have multiple certificates in the
	// primary file, before migration
	instanceCertChain, err := instance.ReadCertificates(i)
	if err != nil && !os.IsNotExist(err) {
		resp.Err = err
		return
	}
	if len(instanceCertChain) == 0 {
		resp.Err = fmt.Errorf("no valid instance certificate found")
		return
	}

	if cf.IsSet("certchain") {
		chain, err := certs.ReadCertificates(h, cf.GetString("certchain"))
		if err != nil && !os.IsNotExist(err) {
			resp.Err = err
			return
		}
		instanceCertChain = append(instanceCertChain, chain...)
	}

	var haveRoot bool
	for _, c := range instanceCertChain {
		// prefer the root cert already in the chain
		if certs.IsValidRootCA(c) {
			haveRoot = true
			break
		}
	}

	if !haveRoot {
		// try to make sure we have a full chain by adding root cert
		rootCert, _, err := geneos.ReadRootCertificate()
		if err != nil {
			resp.Err = fmt.Errorf("cannot read root certificate: %w", err)
			return
		}

		instanceCertChain = append(instanceCertChain, rootCert)
	}

	leaf, intermediates, root, err := certs.ParseCertChain(instanceCertChain...)
	if err != nil {
		resp.Err = err
		return
	}

	// update trusted-roots file
	updated, err := certs.UpdatedCACertsFile(h, geneos.TrustedRootsPath(h), root)
	if err != nil {
		resp.Err = err
		return
	}
	if updated {
		resp.Completed = append(resp.Completed, "updated trusted roots")
	}

	// write fullchain to certificate file - this updates instance parameters for certificate
	err = instance.WriteCertificates(i, append([]*x509.Certificate{leaf}, intermediates...))
	if err != nil {
		resp.Err = err
		return
	}
	resp.Completed = append(resp.Completed, "wrote fullchain to instance certificate file")

	// update instance parameters to new layout
	cf.Set(cf.Join("tls", "privatekey"), cf.GetString("privatekey"))
	cf.Set(cf.Join("tls", "trusted-roots"), geneos.TrustedRootsPath(i.Host()))

	if !cf.GetBool("use-chain") {
		cf.Set(cf.Join("tls", "verify"), false)
	}

	// "certificate" is cleared by WriteCertificates above
	cf.Set("privatekey", "")
	cf.Set("certchain", "")
	cf.Set("use-chain", "")
	cf.Set("truststore", "")
	cf.Set("truststore-password", "")

	if err = instance.SaveConfig(i); err != nil {
		resp.Err = err
		return
	}
	resp.Completed = append(resp.Completed, "updated instance configuration")

	return
}
