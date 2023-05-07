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

package host

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
	"github.com/rs/zerolog/log"
)

// CopyFile copies a file between two locations. Destination can be a
// directory or a file. Parent directories will be created as required.
// Any existing files will be overwritten.
func CopyFile(srcHost *Host, srcPath string, dstHost *Host, dstPath string) (err error) {
	if srcHost == ALL || dstHost == ALL {
		return ErrInvalidArgs
	}

	ss, err := srcHost.Stat(srcPath)
	if err != nil {
		return err
	}
	if ss.IsDir() {
		return fs.ErrInvalid
	}

	sf, err := srcHost.Open(srcPath)
	if err != nil {
		return err
	}
	defer sf.Close()

	ds, err := dstHost.Stat(dstPath)
	if err == nil {
		if ds.IsDir() {
			dstPath = path.Join(dstPath, filepath.Base(srcPath))
		}
	} else {
		dstHost.MkdirAll(utils.Dir(dstPath), 0775)
	}

	df, err := dstHost.Create(dstPath, ss.Mode())
	if err != nil {
		return err
	}
	defer df.Close()
	if _, err = io.Copy(df, sf); err != nil {
		return err
	}
	return
}

// CopyAll copies a directory between any combination of local or remote locations
func CopyAll(srcHost *Host, srcDir string, dstHost *Host, dstDir string) (err error) {
	if srcHost == ALL || dstHost == ALL {
		return ErrInvalidArgs
	}

	if srcHost == LOCAL {
		filesystem := os.DirFS(srcDir)
		fs.WalkDir(filesystem, ".", func(file string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Error().Err(err).Msg("")
				return nil
			}
			fi, err := d.Info()
			if err != nil {
				log.Error().Err(err).Msg("")
				return nil
			}
			dstPath := path.Join(dstDir, file)
			srcPath := path.Join(srcDir, file)
			return copyDirEntry(fi, srcHost, srcPath, dstHost, dstPath)
		})
		return
	}

	s, err := srcHost.DialSFTP()
	if err != nil {
		return err
	}

	w := s.Walk(srcDir)
	for w.Step() {
		if w.Err() != nil {
			log.Error().Err(err).Msg(w.Path())
			continue
		}
		fi := w.Stat()
		srcPath := w.Path()
		dstPath := path.Join(dstDir, strings.TrimPrefix(w.Path(), srcDir))
		if err = copyDirEntry(fi, srcHost, srcPath, dstHost, dstPath); err != nil {
			log.Error().Err(err).Msg("")
			continue
		}
	}
	return
}

func copyDirEntry(fi fs.FileInfo, srcHost *Host, srcPath string, dstHost *Host, dstPath string) (err error) {
	switch {
	case fi.IsDir():
		ds, err := srcHost.Stat(srcPath)
		if err != nil {
			log.Error().Err(err).Msg("")
			return err
		}
		if err = dstHost.MkdirAll(dstPath, ds.Mode()); err != nil {
			return err
		}
	case fi.Mode()&fs.ModeSymlink != 0:
		link, err := srcHost.Readlink(srcPath)
		if err != nil {
			return err
		}
		if err = dstHost.Symlink(link, dstPath); err != nil {
			return err
		}
	default:
		ss, err := srcHost.Stat(srcPath)
		if err != nil {
			return err
		}
		sf, err := srcHost.Open(srcPath)
		if err != nil {
			return err
		}
		defer sf.Close()
		df, err := dstHost.Create(dstPath, ss.Mode())
		if err != nil {
			return err
		}
		defer df.Close()
		if _, err = io.Copy(df, sf); err != nil {
			return err
		}
	}
	return nil
}

func nextRandom() string {
	return fmt.Sprint(rand.Uint32())
}

// based on os.CreatTemp, but allows for hosts and much simplified
// given a remote and a full path, create a file with a suffix
// and return an io.File
func (h *Host) CreateTempFile(path string, perms fs.FileMode) (f io.WriteCloser, name string, err error) {
	try := 0
	for {
		name = path + nextRandom()
		f, err = h.Create(name, perms)
		if os.IsExist(err) {
			if try++; try < 100 {
				continue
			}
			return nil, "", fs.ErrExist
		}
		return
	}
}

// given a path return a cleaned version. If the cleaning results in and
// absolute path or one that tries to ascend the tree then return an
// error
func CleanRelativePath(path string) (clean string, err error) {
	clean = filepath.Clean(path)
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "../") {
		log.Debug().Msgf("path %q must be relative and descending only", clean)
		return "", ErrInvalidArgs
	}

	return
}

// read a PEM encoded RSA private key from path. returns the first found as
// a parsed key
func (r *Host) ReadKey(path string) (key *rsa.PrivateKey, err error) {
	keyPEM, err := r.ReadFile(path)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(keyPEM)
		if p == nil {
			return nil, fmt.Errorf("cannot locate RSA private key in %s", path)
		}
		if p.Type == "RSA PRIVATE KEY" {
			return x509.ParsePKCS1PrivateKey(p.Bytes)
		}
		keyPEM = rest
	}
}

// write a private key as PEM to path. sets file permissions to 0600 (before umask)
func (r *Host) WriteKey(path string, key *rsa.PrivateKey) (err error) {
	log.Debug().Msgf("write key to %s", path)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	return r.WriteFile(path, keyPEM, 0600)
}

// read a PEM encoded cert from path, return the first found as a parsed certificate
func (r *Host) ReadCert(path string) (cert *x509.Certificate, err error) {
	certPEM, err := r.ReadFile(path)
	if err != nil {
		return
	}

	for {
		p, rest := pem.Decode(certPEM)
		if p == nil {
			return nil, fmt.Errorf("cannot locate certificate in %s", path)
		}
		if p.Type == "CERTIFICATE" {
			return x509.ParseCertificate(p.Bytes)
		}
		certPEM = rest
	}
}

// write cert as PEM to path
func (r *Host) WriteCert(path string, cert *x509.Certificate) (err error) {
	log.Debug().Msgf("write cert to %s", path)
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	return r.WriteFile(path, certPEM, 0644)
}

// concatenate certs and write to path
func (r *Host) WriteCerts(path string, certs ...*x509.Certificate) (err error) {
	log.Debug().Msgf("write certs to %s", path)
	var certsPEM []byte
	for _, cert := range certs {
		if cert == nil {
			continue
		}
		p := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		certsPEM = append(certsPEM, p...)
	}
	return r.WriteFile(path, certsPEM, 0644)
}
