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

package tlscmd

import (
	"crypto"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var importCmdCert, importCmdSigner, importCmdChain, importCmdCertKey string

func init() {
	tlsCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importCmdCert, "cert", "c", "", "Instance certificate `file` to import, PEM format")
	importCmd.Flags().StringVarP(&importCmdSigner, "signing", "s", "", "Signing certificate `file` to import, PEM format")
	importCmd.Flags().StringVarP(&importCmdCertKey, "privkey", "k", "", "Private key `file` for certificate, PEM format")

	importCmd.Flags().StringVarP(&importCmdChain, "chain", "C", "", "Certificate chain `file` to import, PEM format")

	importCmd.Flags().SortFlags = false
}

//go:embed _docs/import.md
var importCmdDescription string

var importCmd = &cobra.Command{
	Use:                   "import [flags] [TYPE] [NAME...]",
	Short:                 "Import certificates",
	Long:                  importCmdDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "explicit",
		cmd.AnnotationNeedsHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, names := cmd.ParseTypeNames(command)

		if importCmdCert != "" && importCmdSigner != "" {
			return errors.New("you can only import an instance *or* a signing certificate, not both")
		}

		if importCmdSigner != "" {
			cert, privkey, chain, err := tlsDecompose(importCmdSigner, importCmdCertKey)
			if err != nil {
				return err
			}
			// basic validation
			if !(cert.BasicConstraintsValid && cert.IsCA) {
				return geneos.ErrInvalidArgs
			}

			if err = config.WriteCert(geneos.LOCAL, path.Join(config.AppConfigDir(), geneos.SigningCertFile+".pem"), cert); err != nil {
				return err
			}

			if err = config.WritePrivateKey(geneos.LOCAL, path.Join(config.AppConfigDir(), geneos.SigningCertFile+".key"), privkey); err != nil {
				return err
			}

			if importCmdChain != "" {
				_, _, chain, err := tlsDecompose(importCmdChain, "")
				if err != nil {
					return err
				}
				if err = tlsWriteChainLocal("", chain); err != nil {
					return err
				}
				fmt.Printf("%s certificate chain written using %s\n", cmd.Execname, importCmdChain)
			} else if len(chain) > 0 {
				if err = tlsWriteChainLocal("", chain); err != nil {
					return err
				}
				fmt.Printf("%s certificate chain written using %s\n", cmd.Execname, importCmdSigner)
			}
			return err
		}

		if importCmdCert != "" {
			cert, privkey, chain, err := tlsDecompose(importCmdCert, importCmdCertKey)
			if err != nil {
				return err
			}
			instance.Do(geneos.GetHost(cmd.Hostname), ct, names, tlsWriteInstance, cert, privkey, chain).Write(os.Stdout)
		}

		if importCmdChain == "" {
			return
		}

		_, _, chain, err := tlsDecompose(importCmdChain, "")
		if err != nil {
			return err
		}
		if err = tlsWriteChainLocal("", chain); err != nil {
			return err
		}
		fmt.Printf("%s certificate chain written using %s\n", cmd.Execname, importCmdChain)

		return
	},
}

func tlsWriteChainLocal(chainpath string, chain []*x509.Certificate) (err error) {
	if len(chain) == 0 {
		return
	}
	tlsPath := geneos.LOCAL.PathTo("tls")
	if err = geneos.LOCAL.MkdirAll(tlsPath, 0775); err != nil {
		return err
	}
	if chainpath == "" {
		chainpath = path.Join(tlsPath, geneos.ChainCertFile)
	}
	if err = config.WriteCertChain(geneos.LOCAL, chainpath, chain...); err != nil {
		return err
	}
	return
}

// tlsWriteInstance expects 3 params, of *x509.Certificate,
// *memguard.Enclave and a []*x509.Certificate or it will return an
// error or panic.
func tlsWriteInstance(i geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	cf := i.Config()

	if len(params) != 3 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	cert, key, chain := params[0].(*x509.Certificate), params[1].(*memguard.Enclave), params[2].([]*x509.Certificate)

	if resp.Err = instance.WriteCert(i, cert); resp.Err != nil {
		return
	}
	resp.Lines = append(resp.Lines, fmt.Sprintf("%s certificate written", i))

	if resp.Err = instance.WriteKey(i, key); resp.Err != nil {
		return
	}
	resp.Lines = append(resp.Lines, fmt.Sprintf("%s private key written", i))

	if len(chain) > 0 {
		chainfile := path.Join(i.Home(), "chain.pem")
		if err := config.WriteCertChain(i.Host(), chainfile, chain...); err == nil {
			resp.Lines = append(resp.Lines, fmt.Sprintf("%s certificate chain written", i))
			if cf.GetString("certchain") == chainfile {
				return
			}
			cf.SetString("certchain", chainfile, config.Replace("home"))
		}
	}

	resp.Err = instance.SaveConfig(i)
	return
}

// tlsDecompose reads a PEM file and extracts the first valid
// certificates and an optional PEM private key file path. It returns
// any CA certificates in chain, the certificate in cert and, if found,
// the DER encoded private key for the leaf certificate. If the key is
// found in the cert file then the keyfile arg is ignored. Only
// certificates with the BasicConstraints extension and valid are
// supported. All certificates in chain will have IsCA set. The cert may
// or may not be a leaf certificate.
//
// certfile may be a local file path, a url or '-' for stdin while keyfile
// must be a local file path
func tlsDecompose(certfile, keyfile string) (cert *x509.Certificate, der *memguard.Enclave, chain []*x509.Certificate, err error) {
	var certs []*x509.Certificate
	var leaf *x509.Certificate
	var derkeys []*memguard.Enclave
	var pembytes []byte

	if pembytes, err = geneos.ReadFrom(certfile); err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	for {
		block, rest := pem.Decode(pembytes)
		if block == nil {
			break
		}
		switch block.Type {
		case "CERTIFICATE":
			var c *x509.Certificate
			c, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return
			}
			if !c.BasicConstraintsValid {
				err = geneos.ErrInvalidArgs
				return
			}
			if c.IsCA {
				certs = append(certs, c)
			} else if leaf == nil {
				// save first leaf
				leaf = c
			}
		case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
			// save all private keys for later matching
			derkeys = append(derkeys, memguard.NewEnclave(block.Bytes))
		default:
			err = fmt.Errorf("unsupported PEM type found: %s", block.Type)
			return
		}
		pembytes = rest
	}

	if leaf == nil && len(certs) == 0 {
		err = fmt.Errorf("no certificates found in %s", certfile)
		return
	}

	// if we got this far then we can start setting returns
	cert = leaf
	chain = certs

	// if we have no leaf certificate then user the first cert from the
	// chain BUT leave do not remove from the chain. order is not checked
	if cert == nil {
		cert = chain[0]
	}

	var i int

	// are we good? check key and return a chain of valid CA certs
	i, err = matchKey(cert, derkeys)
	if err != nil {
		// try provided keyfile if no match in cert file
		// no keyfile arg is valid
		if keyfile != "" {
			der, err = config.ReadPrivateKey(geneos.LOCAL, keyfile)
			if err != nil {
				cert = nil
				return
			}
		}
	} else {
		der = derkeys[i]
	}

	err = nil
	return
}

func matchKey(cert *x509.Certificate, derkeys []*memguard.Enclave) (index int, err error) {
	for i, der := range derkeys {
		if pubkey, err := config.PublicKey(der); err == nil { // if ok then compare
			// ensure we have an Equal() method on the opaque key
			if k, ok := pubkey.(interface{ Equal(crypto.PublicKey) bool }); ok {
				if k.Equal(cert.PublicKey) {
					return i, nil
				}
			}
		}
	}
	return -1, os.ErrNotExist
}
