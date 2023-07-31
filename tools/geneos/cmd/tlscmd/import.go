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

var importCmdCert, importCmdSigner, importCmdChain, importCmdCertKey, importCmdSignerKey string

func init() {
	tlsCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importCmdCert, "cert", "c", "", "Instance certificate to import, PEM format")
	importCmd.Flags().StringVarP(&importCmdCertKey, "privkey", "k", "", "Private key for instance certificate, PEM format")

	importCmd.Flags().StringVarP(&importCmdSigner, "signing", "S", "", "Signing certificate to import, PEM format")
	importCmd.Flags().StringVarP(&importCmdSignerKey, "signingkey", "K", "", "Signing keyto import, PEM format")

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
		"wildcard":     "false",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.CmdArgsParams(command)
		log.Debug().Msgf("ct=%s args=%v params=%v", ct, args, params)

		if importCmdSigner != "" {
			chain, cert, privkey, err := tlsDecompose(importCmdSigner, importCmdSignerKey)
			if err != nil {
				return err
			}
			instance.ForAllAny(ct, cmd.Hostname, tlsWriteInstance, args, cert, privkey)
			if importCmdChain == "" {
				tlsWriteChainLocal(chain)
			}
		}

		if importCmdCert != "" {
			chain, cert, privkey, err := tlsDecompose(importCmdCert, importCmdCertKey)
			if err != nil {
				return err
			}
			instance.ForAllAny(ct, cmd.Hostname, tlsWriteInstance, args, cert, privkey)
			if importCmdChain == "" {
				tlsWriteChainLocal(chain)
			}
		}

		if importCmdChain != "" {
			chain, _, _, err := tlsDecompose(importCmdChain, "")
			if err != nil {
				return err
			}
			tlsWriteChainLocal(chain)
		}

		return tlsImport(args...)
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

func tlsWriteInstance(c geneos.Instance, params ...any) (err error) {
	if len(params) != 2 {
		return geneos.ErrInvalidArgs
	}
	cert, ok := params[0].(*x509.Certificate)
	if !ok {
		return geneos.ErrInvalidArgs
	}

	key, ok := params[1].(*memguard.Enclave)
	if !ok {
		return geneos.ErrInvalidArgs
	}

	if err = instance.WriteCert(c, cert); err != nil {
		return
	}
	fmt.Printf("%s certificate written\n", c)

	if err = instance.WriteKey(c, key); err != nil {
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

// import root and signing certs
//
// a root cert is one where subject == issuer
//
// no support for instance certs (yet)
//
// instance cert has CA = false
func tlsImport(sources ...string) (err error) {
	err = geneos.LOCAL.MkdirAll(config.AppConfigDir(), 0755)
	if err != nil {
		return
	}

	// save certs and keys into memory, then check certs for root / etc.
	// and then validate private keys against certs before saving
	// anything to disk
	var certs []*x509.Certificate
	var keys []*memguard.Enclave
	var f []byte

	for _, source := range sources {
		log.Debug().Msgf("importing %s", source)
		if f, err = geneos.ReadFrom(source); err != nil {
			log.Error().Err(err).Msg("")
			err = nil
			continue
		}

		for {
			block, rest := pem.Decode(f)
			if block == nil {
				break
			}
			switch block.Type {
			case "CERTIFICATE":
				cert, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					return err
				}
				certs = append(certs, cert)
			case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
				keys = append(keys, memguard.NewEnclave(block.Bytes))
			default:
				return fmt.Errorf("unknown PEM type found: %s", block.Type)
			}
			f = rest
		}
	}

	var title, prefix string
	for _, cert := range certs {
		if bytes.Equal(cert.RawSubject, cert.RawIssuer) {
			// root cert
			title = "root"
			prefix = geneos.RootCAFile
		} else {
			// signing cert
			title = "signing"
			prefix = geneos.SigningCertFile
		}
		i, err := matchKey(cert, keys)
		if err != nil {
			log.Debug().Msgf("cert: no matching key found, ignoring %s", cert.Subject.String())
			continue
		}

		// pull out the matching key, write files
		key := keys[i]
		if len(keys) > i {
			keys = append(keys[:i], keys[i+1:]...)
		} else {
			keys = keys[:i]
		}

		if err = config.WriteCert(geneos.LOCAL, path.Join(config.AppConfigDir(), prefix+".pem"), cert); err != nil {
			return err
		}
		fmt.Printf("imported %s certificate to %q\n", title, path.Join(config.AppConfigDir(), prefix+".pem"))
		if err = config.WritePrivateKey(geneos.LOCAL, path.Join(config.AppConfigDir(), prefix+".key"), key); err != nil {
			return err
		}
		fmt.Printf("imported %s private key to %q\n", title, path.Join(config.AppConfigDir(), prefix+".pem"))
	}

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
