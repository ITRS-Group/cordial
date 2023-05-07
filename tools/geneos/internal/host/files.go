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
	"sort"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
	"github.com/pkg/sftp"
	"github.com/rs/zerolog/log"
)

// file handling

var (
	ErrInvalidArgs  = fmt.Errorf("invalid argument")
	ErrNotSupported = fmt.Errorf("not supported")
)

// CopyFile copies a file between any combination of local or remote
// locations. Destination can be a directory or a file. Parent
// directories will be created. Any existing file will be overwritten.
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

// shim methods that test Host and direct to ssh / sftp / os
// at some point this should become interface based to allow other
// remote protocols cleanly
func (h *Host) Symlink(oldname, newname string) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Symlink(oldname, newname)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.Symlink(oldname, newname)
	}
}

func (h *Host) Readlink(file string) (link string, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Readlink(file)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.ReadLink(file)
	}
}

func (h *Host) MkdirAll(path string, perm os.FileMode) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.MkdirAll(path, perm)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.MkdirAll(path)
	}
}

func (h *Host) Chown(name string, uid, gid int) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Chown(name, uid, gid)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.Chown(name, uid, gid)
	}
}

// change the symlink ownership on local system, issue chown for remotes
func (h *Host) Lchown(name string, uid, gid int) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Lchown(name, uid, gid)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.Chown(name, uid, gid)
	}
}

func (h *Host) Create(path string, perms fs.FileMode) (out io.WriteCloser, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		var cf *os.File
		cf, err = os.Create(path)
		if err != nil {
			return
		}
		out = cf
		if err = cf.Chmod(perms); err != nil {
			return
		}
	default:
		var cf *sftp.File
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		if cf, err = s.Create(path); err != nil {
			return
		}
		out = cf
		if err = cf.Chmod(perms); err != nil {
			return
		}
	}
	return
}

func (h *Host) Remove(name string) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Remove(name)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.Remove(name)
	}
}

func (h *Host) RemoveAll(name string) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.RemoveAll(name)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}

		// walk, reverse order by prepending and remove
		// we could also just reverse sort strings...
		files := []string{}
		w := s.Walk(name)
		for w.Step() {
			if w.Err() != nil {
				continue
			}
			files = append([]string{w.Path()}, files...)
		}
		for _, file := range files {
			if err = s.Remove(file); err != nil {
				return
			}
		}
		return
	}
}

func (h *Host) Rename(oldpath, newpath string) (err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Rename(oldpath, newpath)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		// use PosixRename to overwrite oldpath
		return s.PosixRename(oldpath, newpath)
	}
}

// massaged file stats
type FileStat struct {
	St    os.FileInfo
	Uid   uint32
	Gid   uint32
	Mtime int64
}

// Stat wraps the os.Stat and sftp.Stat functions
func (h *Host) Stat(name string) (f fs.FileInfo, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Stat(name)
	default:
		var sf *sftp.Client
		if sf, err = h.DialSFTP(); err != nil {
			return
		}
		return sf.Stat(name)
	}
}

// Lstat wraps the os.Lstat and sftp.Lstat functions
func (h *Host) Lstat(name string) (f fs.FileInfo, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.Lstat(name)
	default:
		var sf *sftp.Client
		if sf, err = h.DialSFTP(); err != nil {
			return
		}
		return sf.Lstat(name)
	}
}

func (h *Host) Glob(pattern string) (paths []string, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return filepath.Glob(pattern)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		return s.Glob(pattern)
	}
}

func (h *Host) WriteFile(name string, data []byte, perm os.FileMode) (err error) {
	var s *sftp.Client
	var f *sftp.File

	if h == LOCAL {
		return os.WriteFile(name, data, perm)
	}
	if s, err = h.DialSFTP(); err != nil {
		return
	}
	if f, err = s.Create(name); err != nil {
		return
	}
	defer f.Close()
	f.Chmod(perm)
	_, err = f.Write(data)
	return
}

func (h *Host) ReadFile(name string) (b []byte, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.ReadFile(name)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		f, err := s.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		st, err := f.Stat()
		if err != nil {
			return nil, err
		}
		// force a block read as /proc doesn't give sizes
		sz := st.Size()
		if sz == 0 {
			sz = 8192
		}
		return io.ReadAll(f)
	}
}

// ReadDir reads the named directory and returns all its directory
// entries sorted by name.
func (h *Host) ReadDir(name string) (dirs []os.DirEntry, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		return os.ReadDir(name)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		f, err := s.ReadDir(name)
		if err != nil {
			return nil, err
		}
		sort.Slice(f, func(i, j int) bool {
			return f[i].Name() < f[j].Name()
		})
		for _, d := range f {
			dirs = append(dirs, fs.FileInfoToDirEntry(d))
		}
	}
	return
}

func (h *Host) Open(name string) (f io.ReadSeekCloser, err error) {
	switch h.GetString("name") {
	case LOCALHOST:
		f, err = os.Open(name)
	default:
		var s *sftp.Client
		if s, err = h.DialSFTP(); err != nil {
			return
		}
		f, err = s.Open(name)
	}
	return
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
