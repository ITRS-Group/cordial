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
	"bytes"
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
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, names, params := cmd.TypeNamesParams(command)
		log.Debug().Msgf("ct=%s args=%v params=%v", ct, names, params)

		if importCmdCert != "" && importCmdSigner != "" {
			return errors.New("you can only import an instance *or* a signing certificate, not both")
		}

		if importCmdSigner != "" {
			chain, cert, privkey, err := tlsDecompose(importCmdSigner, importCmdCertKey)
			if err != nil {
				return err
			}
			instance.DoWithValues(geneos.GetHost(cmd.Hostname), ct, names, tlsWriteInstance, cert, privkey)
			if importCmdChain == "" {
				if err = tlsWriteChainLocal(chain); err != nil {
					return err
				}
			}
		}

		if importCmdCert != "" {
			chain, cert, privkey, err := tlsDecompose(importCmdCert, importCmdCertKey)
			if err != nil {
				return err
			}
			instance.DoWithValues(geneos.GetHost(cmd.Hostname), ct, names, tlsWriteInstance, cert, privkey)
			if importCmdChain == "" {
				if err = tlsWriteChainLocal(chain); err != nil {
					return err
				}
			}
		}

		if importCmdChain != "" {
			chain, _, _, err := tlsDecompose(importCmdChain, "")
			if err != nil {
				return err
			}
			if err = tlsWriteChainLocal(chain); err != nil {
				return err
			}
		}

		return
	},
}

func tlsWriteChainLocal(chain []*x509.Certificate) (err error) {
	if len(chain) == 0 {
		return
	}
	tlsPath := geneos.LOCAL.PathTo("tls")
	if err = geneos.LOCAL.MkdirAll(tlsPath, 0775); err != nil {
		return err
	}
	chainpath := path.Join(geneos.LOCAL.PathTo("tls"), geneos.ChainCertFile)
	if err = config.WriteCertChain(geneos.LOCAL, chainpath, chain...); err != nil {
		return err
	}
	return
}

func tlsWriteInstance(c geneos.Instance, params ...any) (result instance.Result) {
	if len(params) != 2 {
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

	if result.Err = instance.WriteCert(c, cert); result.Err != nil {
		return
	}
	fmt.Printf("%s certificate written\n", c)

	if result.Err = instance.WriteKey(c, key); result.Err != nil {
		return
	}
	fmt.Printf("%s private key written\n", c)

	return
}

// tlsDecompose takes a certificate file and an optional key file path.
// It returns any parent certificates in chain, the (first) leaf
// certificate in cert and, if found, the private key for the leaf
// certificate. If the key is found in the cert file then the keyfile
// arg is ignored.
//
// privkey may be encrypted, the caller has to decrypt on return
//
// certfile may be a local file path, a url or '-' for stdin
// keyfile must be a local file path
func tlsDecompose(certfile, keyfile string) (chain []*x509.Certificate, cert *x509.Certificate, privkey *memguard.Enclave, err error) {
	// save certs and keys into memory, then check certs for root / etc.
	// and then validate private keys against certs before saving
	// anything to disk
	var certs []*x509.Certificate
	var keys []*memguard.Enclave
	var f []byte

	log.Debug().Msgf("importing %s", certfile)
	if f, err = geneos.ReadFrom(certfile); err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	for {
		block, rest := pem.Decode(f)
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
			certs = append(certs, c)
		case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
			keys = append(keys, memguard.NewEnclave(block.Bytes))
		default:
			err = fmt.Errorf("unsupported PEM type found: %s", block.Type)
			return
		}
		f = rest
	}

	if len(certs) == 0 {
		err = fmt.Errorf("no certificates found in %s", certfile)
		return
	}

	cert = certs[0]

	// is the first cert a leaf or self-signed?
	if bytes.Equal(cert.RawSubject, cert.RawIssuer) || !cert.IsCA {
		var i int

		// are we good? check key and return the rest as the chain
		i, err = matchKey(cert, keys)
		if err != nil {
			// try provided keyfile if no match in cert file
			privkey, err = config.ReadPrivateKey(geneos.LOCAL, keyfile)
			if err != nil {
				cert = nil
				return
			}
		} else {
			privkey = keys[i]
		}
		if len(certs) > 1 {
			chain = certs[1:]
		}
		return
	}

	// no leaf ? return a chain but no cert, no privkey
	chain = certs
	cert = nil

	return
}

func matchKey(cert *x509.Certificate, keys []*memguard.Enclave) (index int, err error) {
	for i, key := range keys {
		if pubkey, err := config.PublicKey(key); err == nil { // if ok then compare
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
