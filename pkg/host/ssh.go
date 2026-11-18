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

package host

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/sftp"
	"github.com/skeema/knownhosts"
	"github.com/spf13/afero"
	"github.com/spf13/afero/sftpfs"
	"golang.org/x/crypto/ssh"
)

const userSSHdir = ".ssh"

// timeouts for connecting to remote hosts. These are variables so that
// tests can shorten them.
var (
	sshDialTimeout      = 5 * time.Second
	sshHandshakeTimeout = 15 * time.Second
	sshSFTPTimeout      = 15 * time.Second

	// how long a failure is remembered before another attempt is made
	sshRetryInterval = 5 * time.Second
)

// sshSession holds the connections for a remote name. There is one per
// name, shared between all SSHRemote values with that name, and the
// mutex is held while connecting so that concurrent callers share one
// connection rather than each dialling their own.
type sshSession struct {
	mu   sync.Mutex
	ssh  *ssh.Client
	sftp *sftp.Client
}

var sshSessions sync.Map // name -> *sshSession

func sessionFor(name string) *sshSession {
	if s, ok := sshSessions.Load(name); ok {
		return s.(*sshSession)
	}
	s, _ := sshSessions.LoadOrStore(name, &sshSession{})
	return s.(*sshSession)
}

// An SSHRemote a type that satisfies the Host interface for SSH
// attached remote hosts
type SSHRemote struct {
	name     string
	username string
	hostname string
	port     uint16
	password []byte // cannot use *config.Secret because of import loop
	keys     []string

	// optionsErr is set if the options passed to NewSSHRemote could not
	// be used, and is returned when connecting
	optionsErr error

	mu          sync.Mutex // protects failed and lastAttempt
	failed      error
	lastAttempt time.Time
}

// NewSSHRemote returns a new SSHRemote with the given name and options.
// The name is used as the key for caching SSH and SFTP sessions and so
// should be unique for each remote host. Options are used to set the
// hostname, port, username and authentication method (password or
// private keys) for the remote host. If no options are given then the
// local username is used and the hostname is set to the name of the
// remote.
//
// Each option must be an [SSHOption] or a slice of them, so that
// `host.PrivateKeyFiles(...)` results can be passed with or without
// being spread. Anything else cannot be applied and is reported as an
// error, which is returned by [SSHRemote.LastError] and when connecting.
func NewSSHRemote(name string, options ...any) Host {
	r := &SSHRemote{
		name: name,
	}
	evalOptions(r, options...)
	return r
}

// SSHOption TODO: Describe its purpose and intended use.
type SSHOption func(*SSHRemote)

func evalOptions(r *SSHRemote, options ...any) {
	// defaults
	if u, err := user.Current(); err == nil {
		r.username = u.Username
	} else {
		r.username = os.Getenv("USER")
	}

	var errs []error
	apply := func(o SSHOption) {
		if o != nil {
			o(r)
		}
	}
	for _, opt := range options {
		switch o := opt.(type) {
		case nil:
		case SSHOption:
			apply(o)
		case func(*SSHRemote):
			apply(o)
		case []SSHOption:
			for _, so := range o {
				apply(so)
			}
		default:
			errs = append(errs, fmt.Errorf("%w: unsupported SSH option of type %T for remote %s", ErrInvalidArgs, opt, r.name))
		}
	}

	if r.hostname == "" {
		r.hostname = r.name
	}
	r.optionsErr = errors.Join(errs...)
}

// Username TODO: Document its behavior, inputs, and effects.
func Username(username string) SSHOption {
	return func(s *SSHRemote) {
		s.username = username
	}
}

// Password TODO: Document its behavior, inputs, and effects.
func Password(password []byte) SSHOption {
	return func(s *SSHRemote) {
		s.password = password
	}
}

// Port TODO: Document its behavior, inputs, and effects.
func Port(port uint16) SSHOption {
	return func(s *SSHRemote) {
		s.port = port
	}
}

// Hostname TODO: Document its behavior, inputs, and effects.
func Hostname(hostname string) SSHOption {
	return func(s *SSHRemote) {
		s.hostname = hostname
	}
}

// PrivateKeyFiles add the given paths as private key files to use for
// SSH connections. The files must (at this time) not be passphrase
// protected.
func PrivateKeyFiles(paths ...string) SSHOption {
	return func(s *SSHRemote) {
		s.keys = append(s.keys, paths...)
	}
}

// Username TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Username() string {
	return h.username
}

// Hostname TODO: Document its behavior, inputs, and effects.
func (s *SSHRemote) Hostname() string {
	return s.hostname
}

// load any/all the known private keys with no passphrase. A leading `~/`
// in a file name is replaced with homedir.
func readSSHkeys(passphrase []byte, homedir string, files ...string) (signers []ssh.Signer) {
	for _, p := range files {
		if rest, ok := strings.CutPrefix(p, "~/"); ok && homedir != "" {
			p = filepath.Join(homedir, rest)
		}
		key, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		signer, err := ssh.ParsePrivateKey(key)
		if _, ok := errors.AsType[*ssh.PassphraseMissingError](err); ok {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, passphrase)
		}
		if err != nil {
			continue
		}
		signers = append(signers, signer)
	}
	return
}

// sshConnect does the work of connecting to the given remote by
// assembling the authentication methods and dialling. dest is in the
// format of `HOST|IP[:PORT]` where PORT defaults to 22. username is the
// remote login to use and if empty defaults to the local username as
// found by [user.Current]
func sshConnect(dest, username string, password []byte, keyfiles ...string) (client *ssh.Client, err error) {
	var authmethods []ssh.AuthMethod
	var homedir string

	var u *user.User
	u, err = user.Current()
	if err != nil {
		return
	}

	homedir = u.HomeDir
	if home, homeErr := os.UserHomeDir(); homeErr == nil {
		homedir = home
	}

	if username == "" {
		username = u.Username
	}

	// XXX we need this because:
	// https://github.com/golang/go/issues/29286#issuecomment-1160958614
	kh, err := knownhosts.NewDB(filepath.Join(homedir, userSSHdir, "known_hosts"))
	if err != nil {
		return
	}

	keys := kh.HostKeys(dest)
	if len(keys) == 0 {
		return nil, fmt.Errorf("no public key found for %s", dest)
	}

	// the agent connection is only needed while authenticating
	if agentClient, closer := sshConnectAgent(); agentClient != nil {
		defer closer.Close()
		authmethods = append(authmethods, ssh.PublicKeysCallback(agentClient.Signers))
	}

	// either private keys (using password as passphrase) or password
	// but not both
	if signers := readSSHkeys(password, homedir, keyfiles...); len(signers) > 0 {
		authmethods = append(authmethods, ssh.PublicKeys(signers...))
	} else if len(password) > 0 {
		authmethods = append(authmethods, ssh.Password(string(password)))
	}

	config := &ssh.ClientConfig{
		User:              username,
		Auth:              authmethods,
		HostKeyCallback:   kh.HostKeyCallback(),
		HostKeyAlgorithms: kh.HostKeyAlgorithms(dest),
	}

	// as ssh.Dial, but with a deadline for the handshake, as the
	// ClientConfig Timeout only covers making the TCP connection
	conn, err := net.DialTimeout("tcp", dest, sshDialTimeout)
	if err != nil {
		return nil, err
	}
	if err = conn.SetDeadline(time.Now().Add(sshHandshakeTimeout)); err != nil {
		conn.Close()
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, dest, config)
	if err != nil {
		conn.Close()
		if knownhosts.IsHostKeyChanged(err) {
			return nil, fmt.Errorf("host key has changed for %s: %w", dest, err)
		}
		return nil, err
	}
	if err = conn.SetDeadline(time.Time{}); err != nil {
		c.Close()
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

// recentFailure returns the error from the last failed attempt to
// connect if it was less than sshRetryInterval ago.
func (h *SSHRemote) recentFailure() (since time.Duration, failed error, ok bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.failed != nil && !h.lastAttempt.IsZero() && time.Since(h.lastAttempt) < sshRetryInterval {
		return time.Since(h.lastAttempt), h.failed, true
	}
	return 0, nil, false
}

func (h *SSHRemote) setFailed(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failed = err
	h.lastAttempt = time.Now()
}

func (h *SSHRemote) clearFailed() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failed = nil
	h.lastAttempt = time.Time{}
}

// DialSSH connects to a remote host using ssh and returns an *ssh.Client
// on success. Each connection is cached and returned if found, but a
// connection that has been closed is replaced. Concurrent callers for
// the same remote share a single connection attempt. To remove a
// session call Close()
func (h *SSHRemote) DialSSH() (sc *ssh.Client, err error) {
	if h == nil {
		err = ErrInvalidArgs
		return
	}

	s := sessionFor(h.name)
	s.mu.Lock()
	defer s.mu.Unlock()
	return h.dialSSH(s)
}

// dialSSH returns the cached connection or makes a new one, and must be
// called with s.mu held.
func (h *SSHRemote) dialSSH(s *sshSession) (sc *ssh.Client, err error) {
	if h.optionsErr != nil {
		return nil, h.optionsErr
	}

	if s.ssh != nil {
		return s.ssh, nil
	}

	if _, failed, ok := h.recentFailure(); ok {
		return nil, failed
	}

	if h.hostname == "" {
		return nil, fmt.Errorf("%w hostname not set for remote %s", ErrInvalidArgs, h)
	}
	port := h.port
	if port == 0 {
		port = 22
	}

	dest := net.JoinHostPort(h.hostname, strconv.Itoa(int(port)))
	sc, err = sshConnect(dest, h.username, h.password, h.keys...)
	if err != nil {
		err = fmt.Errorf("%w (note: you MUST add remote keys manually to known_hosts)", err)
		h.setFailed(err)
		return nil, err
	}
	h.clearFailed()
	s.ssh = sc

	// forget the connection, and anything using it, when it closes
	go func() {
		sc.Wait()
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.ssh == sc {
			s.ssh = nil
			if s.sftp != nil {
				s.sftp.Close()
				s.sftp = nil
			}
		}
	}()
	return
}

// Close a remote host connection
func (h *SSHRemote) Close() {
	if h == nil {
		return
	}

	s := sessionFor(h.name)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.close()
}

// close drops both connections, and must be called with s.mu held
func (s *sshSession) close() {
	if s.sftp != nil {
		s.sftp.Close()
		s.sftp = nil
	}
	if s.ssh != nil {
		s.ssh.Close()
		s.ssh = nil
	}
}

// DialSFTP connects to the remote host using SSH and returns an
// *sftp.Client is successful. The connection is cached and returned if
// found, but one that has been closed is replaced.
func (h *SSHRemote) DialSFTP() (f *sftp.Client, err error) {
	if h == nil {
		err = ErrInvalidArgs
		return
	}

	s := sessionFor(h.name)
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sftp != nil {
		return s.sftp, nil
	}

	sc, err := h.dialSSH(s)
	if err != nil {
		return nil, err
	}

	// the SFTP handshake has no deadline of its own, so close the
	// connection if the server does not answer
	timer := time.AfterFunc(sshSFTPTimeout, func() { sc.Close() })
	f, err = sftp.NewClient(sc)
	if !timer.Stop() && err == nil {
		f.Close()
		err = fmt.Errorf("sftp handshake with %s timed out after %s", h.name, sshSFTPTimeout)
	}
	if err != nil {
		h.setFailed(err)
		return nil, err
	}
	s.sftp = f

	go func() {
		f.Wait()
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.sftp == f {
			s.sftp = nil
		}
	}()

	return
}

// CloseSFTP closes the cached SFTP connection to the remote host, if
// any. The SSH connection is not closed.
func (h *SSHRemote) CloseSFTP() {
	if h == nil {
		return
	}

	s := sessionFor(h.name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sftp != nil {
		s.sftp.Close()
		s.sftp = nil
	}
}

// IsLocalhost returns true if h is local, which for SSH is always false
func (h *SSHRemote) IsLocalhost() bool {
	return false
}

// IsAvailable returns true is the remote host can be contacted
func (h *SSHRemote) IsAvailable() (ok bool, err error) {
	if h == nil {
		return false, ErrInvalidArgs
	}

	// only retry every few seconds, otherwise return previous error with a time
	if since, failed, ok := h.recentFailure(); ok {
		return false, fmt.Errorf("%w (%v ago)", failed, since)
	}

	_, err = h.DialSSH()
	return err == nil, err
}

// String TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) String() string {
	return h.name
}

// Abs TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Abs(dir string) (string, error) {
	if s, err := h.DialSFTP(); err != nil {
		return "", err
	} else {
		return s.RealPath(dir)
	}
}

// Base returns the final slash-separated path element.
func (h *SSHRemote) Base(file string) string {
	return path.Base(file)
}

// Clean returns the shortest slash-separated path name equivalent to file.
func (h *SSHRemote) Clean(file string) string {
	return path.Clean(file)
}

// Dir returns the directory portion of a slash-separated path.
func (h *SSHRemote) Dir(file string) string {
	return path.Dir(file)
}

// Ext returns the extension of a slash-separated path.
func (h *SSHRemote) Ext(file string) string {
	return path.Ext(file)
}

// IsAbs reports whether name is an absolute slash-separated path.
func (h *SSHRemote) IsAbs(name string) bool {
	return path.IsAbs(name)
}

// Join joins slash-separated path elements.
func (h *SSHRemote) Join(elem ...string) string {
	return path.Join(elem...)
}

// Rel returns a slash-separated path from basepath to targpath.
func (h *SSHRemote) Rel(basepath, targpath string) (string, error) {
	basepath = path.Clean(basepath)
	targpath = path.Clean(targpath)
	if path.IsAbs(basepath) != path.IsAbs(targpath) {
		return "", fmt.Errorf("Rel: can't make %q relative to %q", targpath, basepath)
	}

	base := splitSlashPath(basepath)
	target := splitSlashPath(targpath)
	common := 0
	for common < len(base) && common < len(target) && base[common] == target[common] {
		common++
	}

	parts := make([]string, 0, len(base)-common+len(target)-common)
	for index := 0; index < len(base)-common; index++ {
		parts = append(parts, "..")
	}
	parts = append(parts, target[common:]...)
	if len(parts) == 0 {
		return ".", nil
	}
	return strings.Join(parts, "/"), nil
}

func splitSlashPath(file string) []string {
	file = strings.Trim(file, "/")
	if file == "" || file == "." {
		return nil
	}
	return strings.Split(file, "/")
}

// ToSlash converts backslash separators to slashes.
func (h *SSHRemote) ToSlash(file string) string {
	return strings.ReplaceAll(file, `\`, "/")
}

// VolumeName returns an empty volume for slash-based SSH paths.
func (h *SSHRemote) VolumeName(string) string {
	return ""
}

// Split separates a slash-separated path into its directory and final element.
func (h *SSHRemote) Split(file string) (dir, base string) {
	return path.Split(file)
}

// Getwd TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Getwd() (dir string, err error) {
	if s, err := h.DialSFTP(); err != nil {
		return "", err
	} else {
		return s.Getwd()
	}
}

// Symlink TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Symlink(oldname, newname string) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.Symlink(oldname, newname)
	}
}

// Readlink TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Readlink(file string) (string, error) {
	if s, err := h.DialSFTP(); err != nil {
		return "", err
	} else {
		return s.ReadLink(file)
	}
}

// Mkdir creates a directory on the remote host with permissions perm.
// SFTP creates directories with the server's default permissions, so
// they are set afterwards and are not subject to the remote umask.
func (h *SSHRemote) Mkdir(p string, perm os.FileMode) error {
	s, err := h.DialSFTP()
	if err != nil {
		return err
	}
	if err = s.Mkdir(p); err != nil {
		return err
	}
	return s.Chmod(p, perm)
}

// MkdirAll creates a directory and any missing parents on the remote
// host, giving each directory it creates the permissions perm, like
// [os.MkdirAll].
func (h *SSHRemote) MkdirAll(p string, perm os.FileMode) error {
	s, err := h.DialSFTP()
	if err != nil {
		return err
	}
	return sftpMkdirAll(s, path.Clean(p), perm)
}

func sftpMkdirAll(s *sftp.Client, p string, perm os.FileMode) error {
	if fi, err := s.Stat(p); err == nil {
		if fi.IsDir() {
			return nil
		}
		return &fs.PathError{Op: "mkdir", Path: p, Err: syscall.ENOTDIR}
	}

	if parent := path.Dir(p); parent != p {
		if err := sftpMkdirAll(s, parent, perm); err != nil {
			return err
		}
	}

	if err := s.Mkdir(p); err != nil {
		// lost a race to create it
		if fi, serr := s.Lstat(p); serr == nil && fi.IsDir() {
			return nil
		}
		return err
	}
	return s.Chmod(p, perm)
}

// Chown TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Chown(name string, uid, gid int) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.Chown(name, uid, gid)
	}
}

// Lchown just down a Chown() for remote files
func (h *SSHRemote) Lchown(name string, uid, gid int) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.Chown(name, uid, gid)
	}
}

// Chtimes TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Chtimes(path string, atime time.Time, mtime time.Time) (err error) {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.Chtimes(path, atime, mtime)
	}
}

// Lchtimes TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Lchtimes(path string, atime time.Time, mtime time.Time) (err error) {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		// try plain chtimes
		return s.Chtimes(path, atime, mtime)
	}
}

// Create opens p for writing on the remote host, creating or truncating
// it. SFTP cannot set permissions when creating a file, so perms are set
// immediately afterwards and a new file is briefly created with the
// server's default permissions.
func (h *SSHRemote) Create(p string, perms fs.FileMode) (out io.WriteCloser, err error) {
	s, err := h.DialSFTP()
	if err != nil {
		return nil, err
	}
	cf, err := s.Create(p)
	if err != nil {
		return nil, err
	}
	if err = cf.Chmod(perms); err != nil {
		cf.Close()
		return nil, err
	}
	return cf, nil
}

// Remove TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Remove(name string) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.Remove(name)
	}
}

// RemoveAll removes name and any children it contains, without following
// symbolic links. Like [os.RemoveAll] it is not an error if name does
// not exist.
func (h *SSHRemote) RemoveAll(name string) (err error) {
	s, err := h.DialSFTP()
	if err != nil {
		return err
	}
	if err = s.RemoveAll(name); errors.Is(err, fs.ErrNotExist) {
		err = nil
	}
	return
}

// Rename moves oldpath to newpath, replacing newpath if it exists. This
// uses the posix-rename SFTP extension and falls back to a plain rename,
// which fails if newpath exists, for servers that do not support it.
func (h *SSHRemote) Rename(oldpath, newpath string) error {
	s, err := h.DialSFTP()
	if err != nil {
		return err
	}
	err = s.PosixRename(oldpath, newpath)
	if se, ok := errors.AsType[*sftp.StatusError](err); ok && se.FxCode() == sftp.ErrSSHFxOpUnsupported {
		return s.Rename(oldpath, newpath)
	}
	return err
}

// Link TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Link(oldname, newname string) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.Link(oldname, newname)
	}
}

// Stat wraps the os.Stat and sftp.Stat functions
func (h *SSHRemote) Stat(name string) (fs.FileInfo, error) {
	if s, err := h.DialSFTP(); err != nil {
		return nil, err
	} else {
		return s.Stat(name)
	}
}

// Lstat wraps the os.Lstat and sftp.Lstat functions
func (h *SSHRemote) Lstat(name string) (fs.FileInfo, error) {
	if s, err := h.DialSFTP(); err != nil {
		return nil, err
	} else {
		return s.Lstat(name)
	}
}

// Glob TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Glob(pattern string) ([]string, error) {
	if s, err := h.DialSFTP(); err != nil {
		return []string{}, err
	} else {
		return s.Glob(pattern)
	}
}

// WriteFile writes data to name on the remote host, creating or
// truncating it, and sets its permissions to perm.
func (h *SSHRemote) WriteFile(name string, data []byte, perm os.FileMode) (err error) {
	s, err := h.DialSFTP()
	if err != nil {
		return
	}
	f, err := s.Create(name)
	if err != nil {
		return
	}
	defer func() {
		// the data is only known to be written when the close succeeds
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	if err = f.Chmod(perm); err != nil {
		return
	}
	_, err = f.Write(data)
	return
}

// ReadFile reads the complete contents of name on the remote host.
func (h *SSHRemote) ReadFile(name string) (b []byte, err error) {
	s, err := h.DialSFTP()
	if err != nil {
		return
	}
	f, err := s.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// do not rely on the size, as /proc doesn't give them
	return io.ReadAll(f)
}

// ReadDir reads the named directory and returns all its directory
// entries sorted by name.
func (h *SSHRemote) ReadDir(name string) (dirs []os.DirEntry, err error) {
	var s *sftp.Client
	if s, err = h.DialSFTP(); err != nil {
		return
	}
	f, err := s.ReadDir(name)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(f, func(a, b os.FileInfo) int {
		return strings.Compare(a.Name(), b.Name())
	})
	for _, d := range f {
		dirs = append(dirs, fs.FileInfoToDirEntry(d))
	}
	return
}

// Open TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Open(name string) (io.ReadSeekCloser, error) {
	if s, err := h.DialSFTP(); err != nil {
		return nil, err
	} else {
		return s.Open(name)
	}
}

// HostPath TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) HostPath(p string) string {
	return fmt.Sprintf("%s:%s", h, p)
}

// TempDir returns a path on the remote to a temporary directory
//
// BUG This is currently broken - hardwired values for now
func (h *SSHRemote) TempDir() string {
	if h.OS() == "windows" {
		return `C:\TEMP`
	}
	return "/tmp"
}

// LastError returns the error from the last failed attempt to connect to
// the remote host, if there has been one since it last succeeded, or the
// error from invalid options passed to [NewSSHRemote]. It does not try
// to connect.
func (h *SSHRemote) LastError() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.failed != nil {
		return h.failed
	}
	return h.optionsErr
}

// ServerVersion TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) ServerVersion() string {
	remote, err := h.DialSSH()
	if err != nil {
		return ""
	}
	return string(remote.ServerVersion())
}

// OS returns "windows" if the SSH server identifies itself as running on
// Windows, and "linux" otherwise, including if the server cannot be
// reached.
func (h *SSHRemote) OS() string {
	if strings.Contains(strings.ToLower(h.ServerVersion()), "windows") {
		return "windows"
	}
	return "linux"
}

// GetFs returns an afero filesystem for the remote host. If the host
// cannot be reached then the returned filesystem returns that error from
// every operation, so the result is never nil.
func (h *SSHRemote) GetFs() afero.Fs {
	client, err := h.DialSFTP()
	if err != nil {
		return errFs{err}
	}
	return sftpfs.New(client)
}

// errFs is an [afero.Fs] where every operation fails with err
type errFs struct{ err error }

func (e errFs) Create(name string) (afero.File, error) { return nil, e.err }
func (e errFs) Mkdir(name string, perm os.FileMode) error {
	return e.err
}
func (e errFs) MkdirAll(path string, perm os.FileMode) error { return e.err }
func (e errFs) Open(name string) (afero.File, error)         { return nil, e.err }
func (e errFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, e.err
}
func (e errFs) Remove(name string) error                          { return e.err }
func (e errFs) RemoveAll(path string) error                       { return e.err }
func (e errFs) Rename(oldname, newname string) error              { return e.err }
func (e errFs) Stat(name string) (os.FileInfo, error)             { return nil, e.err }
func (e errFs) Name() string                                      { return "errFs" }
func (e errFs) Chmod(name string, mode os.FileMode) error         { return e.err }
func (e errFs) Chown(name string, uid, gid int) error             { return e.err }
func (e errFs) Chtimes(name string, atime, mtime time.Time) error { return e.err }

// Signal sends a signal to the remote pid, which must be greater than
// zero, and returns nil on success. If the process is not found then
// an error matching [os.ErrProcessDone] is returned and if it cannot be
// signalled for lack of permission then an error matching
// [syscall.EPERM] is returned.
func (h *SSHRemote) Signal(pid int, signal syscall.Signal) (err error) {
	if pid <= 0 {
		return fmt.Errorf("%w: invalid pid %d", ErrInvalidArgs, pid)
	}

	sess, err := h.NewSession()
	if err != nil {
		return
	}
	defer sess.Close()

	// kill has the same exit status for all failures, so check what it
	// says, in a locale we can match
	var stderr bytes.Buffer
	sess.Stderr = &stderr
	err = sess.Run(fmt.Sprintf("LC_ALL=C kill -s %d %d", signal, pid))
	if err != nil {
		if exitErr, ok := errors.AsType[*ssh.ExitError](err); ok {
			return killError(pid, exitErr.ExitStatus(), stderr.String())
		}
	}
	return
}

// killError converts the exit status and output of kill to an error
func killError(pid, status int, output string) error {
	if status == 0 {
		return nil
	}
	output = strings.TrimSpace(output)
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "no such process"):
		return os.ErrProcessDone
	case strings.Contains(lower, "not permitted"):
		return fmt.Errorf("signalling pid %d: %w", pid, syscall.EPERM)
	default:
		return fmt.Errorf("signalling pid %d failed with status %d: %s", pid, status, output)
	}
}

// NewSession wraps ssh.NewSession but does some retries
func (h *SSHRemote) NewSession() (sess *ssh.Session, err error) {
	rem, err := h.DialSSH()
	if err != nil {
		err = fmt.Errorf("Start: %w during Dial()", err)
		return
	}

	// the number of sessions is always limited by config on the remote
	// server, but we don't know what that limit is, so retry a few
	// times with a small delay
	for range 10 {
		sess, err = rem.NewSession()
		if err == nil {
			break
		}
		if _, ok := errors.AsType[*ssh.OpenChannelError](err); !ok {
			// not a limit, so the connection is probably dead; forget
			// it so that the next call connects again
			h.forget(rem)
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if err != nil {
		err = fmt.Errorf("Start: %w during NewSession()", err)
		return
	}
	return
}

// forget closes and drops the cached connection if it is still rem
func (h *SSHRemote) forget(rem *ssh.Client) {
	s := sessionFor(h.name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ssh == rem {
		s.close()
	}
}

// shellQuote quotes s for a POSIX shell, so that it is passed on as a
// single word with no expansion of any kind. Single quotes protect
// everything except a single quote, which is closed, escaped with a
// backslash and reopened.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

var envNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// remoteCommand builds the command to run cmd in a shell on a remote
// host, with every argument and environment value quoted so that it
// reaches the program exactly as given. Any cmd.Env entries are set,
// in addition to the remote environment, using env(1). We cannot use
// sess.Setenv() because the openssh daemon blocks non-whitelisted
// variables from being set. Note that cmd.Args already has the command
// as Args[0], so no Path is required. Entries in cmd.Env without an
// `=` are ignored, as they are by [exec.Cmd].
func remoteCommand(cmd *exec.Cmd) (string, error) {
	if len(cmd.Args) == 0 {
		return "", fmt.Errorf("%w: no command", ErrInvalidArgs)
	}

	var words []string
	for _, e := range cmd.Env {
		k, _, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		if !envNameRE.MatchString(k) {
			return "", fmt.Errorf("%w: invalid environment variable name %q", ErrInvalidArgs, k)
		}
		words = append(words, shellQuote(e))
	}
	if len(words) > 0 {
		words = slices.Insert(words, 0, "env")
	}
	for _, a := range cmd.Args {
		words = append(words, shellQuote(a))
	}
	return strings.Join(words, " "), nil
}

// Start starts a process on an SSH attached remote host h. It uses a
// shell and backgrounds and redirects. May not work on all remotes and
// for all processes. errfile has stdout/stderr appended to it, and
// defaults to '/dev/null'. Arguments and environment values are passed
// to the program as given, they are not expanded by the remote shell.
func (h *SSHRemote) Start(cmd *exec.Cmd, options ...ProcessOption) (pid int, err error) {
	if h.OS() == "windows" {
		return 0, fmt.Errorf("%w: cannot run remote commands on windows", ErrNotSupported)
	}

	po := evalProcessOptions(options...)
	errfile := po.errfile
	if errfile == "" {
		errfile = "/dev/null"
	}

	cmdstr, err := remoteCommand(cmd)
	if err != nil {
		return 0, err
	}
	if cmd.Dir != "" {
		cmdstr = fmt.Sprintf("cd %s || exit 1; %s", shellQuote(cmd.Dir), cmdstr)
	}
	cmdstr = fmt.Sprintf("%s </dev/null >>%s 2>&1 & echo $!", cmdstr, shellQuote(errfile))

	sess, err := h.NewSession()
	if err != nil {
		return
	}
	defer sess.Close()

	var output []byte
	if output, err = sess.Output(cmdstr); err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(output))
	pid, err = strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse PID from remote output: %w", err)
	}
	return pid, nil
}

// Run starts a process on an SSH attached remote host h. It uses a
// shell and waits for the process status before returning. It returns
// the output and any error. errfile is an optional (remote) file for
// stderr output, which is replaced if it exists. Arguments and
// environment values are passed to the program as given, they are not
// expanded by the remote shell.
func (h *SSHRemote) Run(cmd *exec.Cmd, options ...ProcessOption) (output []byte, err error) {
	if h.OS() == "windows" {
		return nil, fmt.Errorf("%w: cannot run remote commands on windows", ErrNotSupported)
	}

	po := evalProcessOptions(options...)
	errfile := po.errfile

	cmdstr, err := remoteCommand(cmd)
	if err != nil {
		return nil, err
	}
	if cmd.Dir != "" {
		cmdstr = fmt.Sprintf("cd %s && %s", shellQuote(cmd.Dir), cmdstr)
	}

	sess, err := h.NewSession()
	if err != nil {
		return
	}
	defer sess.Close()

	if errfile != "" {
		if !h.IsAbs(errfile) {
			errfile = path.Join(cmd.Dir, errfile)
		}
		e, cerr := h.Create(errfile, 0664)
		if cerr != nil {
			return []byte{}, cerr
		}
		defer func() {
			if cerr := e.Close(); err == nil {
				err = cerr
			}
		}()
		sess.Stderr = e
	}

	return sess.Output(cmdstr)
}

// Uname TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Uname() (os, arch string, err error) {
	if h.OS() == "windows" {
		err = fmt.Errorf("%w: cannot run remote commands on windows", ErrNotSupported)
		return
	}

	sess, err := h.NewSession()
	if err != nil {
		return
	}
	defer sess.Close()
	out, err := sess.Output("/usr/bin/uname -s -m")
	if err != nil {
		return
	}
	for w := range bytes.FieldsSeq(out) {
		switch string(bytes.ToLower(w)) {
		case "linux":
			os = "linux"
		case "x86_64":
			arch = "x86_64"
		default:
			// ignore for now
		}
	}

	return
}

// WalkDir walks the tree below dir on the remote host, calling fn with
// paths relative to dir, exactly as [fs.WalkDir] does for a local
// directory, including the handling of [fs.SkipDir] and [fs.SkipAll].
func (h *SSHRemote) WalkDir(dir string, fn fs.WalkDirFunc) error {
	s, err := h.DialSFTP()
	if err != nil {
		return err
	}
	return fs.WalkDir(sftpFS{c: s, root: dir}, ".", fn)
}

// sftpFS is a read-only [fs.FS] for the tree below root on an SFTP
// server, which allows the standard library to do the work of walking it.
type sftpFS struct {
	c    *sftp.Client
	root string
}

var (
	_ fs.ReadDirFS = sftpFS{}
	_ fs.StatFS    = sftpFS{}
)

func (f sftpFS) resolve(op, name string) (string, error) {
	if !fs.ValidPath(name) {
		return "", &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}
	return path.Join(f.root, name), nil
}

func (f sftpFS) Open(name string) (fs.File, error) {
	p, err := f.resolve("open", name)
	if err != nil {
		return nil, err
	}
	file, err := f.c.Open(p)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (f sftpFS) Stat(name string) (fs.FileInfo, error) {
	p, err := f.resolve("stat", name)
	if err != nil {
		return nil, err
	}
	return f.c.Stat(p)
}

func (f sftpFS) ReadDir(name string) ([]fs.DirEntry, error) {
	p, err := f.resolve("readdir", name)
	if err != nil {
		return nil, err
	}
	infos, err := f.c.ReadDir(p)
	if err != nil {
		return nil, err
	}
	entries := make([]fs.DirEntry, 0, len(infos))
	for _, fi := range infos {
		entries = append(entries, fs.FileInfoToDirEntry(fi))
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})
	return entries, nil
}
