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

	importCmd.Flags().StringVarP(&importCmdCert, "cert", "c", "", "Instance certificate to import, PEM format")
	importCmd.Flags().StringVarP(&importCmdSigner, "signing", "s", "", "Signing certificate to import, PEM format")
	importCmd.Flags().StringVarP(&importCmdCertKey, "privkey", "k", "", "Private key for certificate, PEM format")

	importCmd.Flags().StringVarP(&importCmdChain, "chain", "C", "", "Certificate chain to import, PEM format")

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
		ct, names, params := cmd.TypeNamesParams(command)
		log.Debug().Msgf("ct=%s args=%v params=%v", ct, names, params)

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

			if err = config.WriteCert(geneos.LOCAL, geneos.LOCAL.PathTo("tls", geneos.SigningCertFile+".pem"), cert); err != nil {
				return err
			}

			if err = config.WritePrivateKey(geneos.LOCAL, geneos.LOCAL.PathTo("tls", geneos.SigningCertFile+".key"), privkey); err != nil {
				return err
			}

			if importCmdChain == "" && len(chain) > 0 {
				if err = tlsWriteChainLocal("", chain); err != nil {
					return err
				}
			}
		}

		if importCmdCert != "" {
			cert, privkey, chain, err := tlsDecompose(importCmdCert, importCmdCertKey)
			if err != nil {
				return err
			}
			responses := instance.DoWithValues(geneos.GetHost(cmd.Hostname), ct, names, tlsWriteInstance, cert, privkey, chain)
			responses.Write(os.Stdout)
		}

		if importCmdChain != "" {
			_, _, chain, err := tlsDecompose(importCmdChain, "")
			if err != nil {
				return err
			}
			if err = tlsWriteChainLocal("", chain); err != nil {
				return err
			}
			fmt.Println("local certificate chain written")
		}

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
		chainpath = path.Join(geneos.LOCAL.PathTo("tls"), geneos.ChainCertFile)
	}
	if err = config.WriteCertChain(geneos.LOCAL, chainpath, chain...); err != nil {
		return err
	}
	return
}

func tlsWriteInstance(c geneos.Instance, params ...any) (result instance.Response) {
	var chain []*x509.Certificate

	cf := c.Config()

	if len(params) < 2 {
		result.Err = geneos.ErrInvalidArgs
		return
	}

	cert, ok := params[0].(*x509.Certificate)
	if !ok {
		result.Err = geneos.ErrInvalidArgs
		return
	}

	key, ok := params[1].(*memguard.Enclave)
	if !ok {
		result.Err = geneos.ErrInvalidArgs
		return
	}

	if len(params) > 2 {
		c, ok := params[2].([]*x509.Certificate)
		if ok {
			chain = c
		}
	}

	if result.Err = instance.WriteCert(c, cert); result.Err != nil {
		return
	}
	result.Strings = append(result.Strings, fmt.Sprintf("%s certificate written", c))

	if result.Err = instance.WriteKey(c, key); result.Err != nil {
		return
	}
	result.Strings = append(result.Strings, fmt.Sprintf("%s private key written", c))

	if len(chain) > 0 {
		chainfile := path.Join(c.Home(), "chain.pem")
		if err := config.WriteCertChain(c.Host(), chainfile, chain...); err == nil {
			result.Strings = append(result.Strings, fmt.Sprintf("%s certificate chain written", c))
			if cf.GetString("certchain") == chainfile {
				return
			}
			cf.Set("certchain", chainfile)
			result.Err = instance.SaveConfig(c)
		}
	}

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
