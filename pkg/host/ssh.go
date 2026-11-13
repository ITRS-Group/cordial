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
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
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

var sshSessions sync.Map
var sftpSessions sync.Map

// An SSHRemote a type that satisfies the Host interface for SSH
// attached remote hosts
type SSHRemote struct {
	name        string
	username    string
	hostname    string
	port        uint16
	password    []byte // cannot use *config.Secret because of import loop
	keys        []string
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

	for _, opt := range options {
		switch o := opt.(type) {
		case SSHOption:
			o(r)
		}
	}

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

// load any/all the known private keys with no passphrase
func readSSHkeys(passphrase []byte, homedir string, files ...string) (signers []ssh.Signer) {
	for _, p := range files {
		key, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		signer, err := ssh.ParsePrivateKey(key)
		pperr := &ssh.PassphraseMissingError{}
		if err != nil && errors.As(err, &pperr) {
			err = nil
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, passphrase)

			if err != nil {
				continue
			}
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

	if agentClient := sshConnectAgent(); agentClient != nil {
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
		Timeout:           5 * time.Second,
	}
	client, err = ssh.Dial("tcp", dest, config)
	if err != nil && knownhosts.IsHostKeyChanged(err) {
		return nil, fmt.Errorf("host key has changed for %s: %w", dest, err)
	}
	return
}

// DialSSH connects to a remote host using ssh and returns an *ssh.Client
// on success. Each connection is cached and returned if found without
// checking if it is still valid. To remove a session call Close()
func (h *SSHRemote) DialSSH() (sc *ssh.Client, err error) {
	if h == nil {
		err = ErrInvalidArgs
		return
	}

	if h.failed != nil && !h.lastAttempt.IsZero() && time.Since(h.lastAttempt) < 5*time.Second {
		err = h.failed
		return
	}

	if h.hostname == "" {
		return nil, fmt.Errorf("%w hostname not set for remote %s", ErrInvalidArgs, h)
	}
	if h.port == 0 {
		h.port = 22
	}

	dest := fmt.Sprintf("%s:%d", h.hostname, h.port)
	if val, ok := sshSessions.Load(h.name); ok {
		sc = val.(*ssh.Client)
	} else {
		sc, err = sshConnect(dest, h.username, h.password, h.keys...)
		if err != nil {
			h.failed = err
			h.lastAttempt = time.Now()
			return sc, fmt.Errorf("%w (note: you MUST add remote keys manually to known_hosts)", err)
		}
		sshSessions.Store(h.name, sc)
	}
	return
}

// Close a remote host connection
func (h *SSHRemote) Close() {
	if h == nil {
		return
	}

	h.CloseSFTP()

	val, ok := sshSessions.Load(h.name)
	if ok {
		s := val.(*ssh.Client)
		s.Close()
		sshSessions.Delete(h.name)
	}
}

// DialSFTP connects to the remote host using SSH and returns an
// *sftp.Client is successful. The connection is cached and returned if
// found, without checking if it is still valid.
func (h *SSHRemote) DialSFTP() (f *sftp.Client, err error) {
	if h == nil {
		err = ErrInvalidArgs
		return
	}

	if val, ok := sftpSessions.Load(h.name); ok {
		f = val.(*sftp.Client)
		return
	}

	var s *ssh.Client
	if s, err = h.DialSSH(); err != nil {
		h.failed = err
		h.lastAttempt = time.Now()
		return
	}
	if f, err = sftp.NewClient(s); err != nil {
		h.failed = err
		h.lastAttempt = time.Now()
		return
	}
	sftpSessions.Store(h.name, f)

	return
}

// CloseSFTP TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) CloseSFTP() {
	if h == nil {
		return
	}

	val, ok := sftpSessions.Load(h.name)
	if ok {
		f := val.(*sftp.Client)
		f.Close()
		sftpSessions.Delete(h.name)
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

	if h.failed != nil {
		// only retry every 5 seconds, otherwise return previous error with a time
		if !h.lastAttempt.IsZero() && time.Since(h.lastAttempt) < 5*time.Second {
			// not available for 5 seconds since last error
			return false, fmt.Errorf("%w (%v ago)", h.failed, time.Since(h.lastAttempt))
		}
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

// Mkdir TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Mkdir(p string, perm os.FileMode) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.Mkdir(p)
	}
}

// MkdirAll TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) MkdirAll(p string, perm os.FileMode) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.MkdirAll(p)
	}
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

// Create TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Create(p string, perms fs.FileMode) (out io.WriteCloser, err error) {
	var cf *sftp.File
	var s *sftp.Client
	if s, err = h.DialSFTP(); err != nil {
		return
	}
	if cf, err = s.Create(p); err != nil {
		return
	}
	out = cf
	if err = cf.Chmod(perms); err != nil {
		return
	}
	return
}

// Remove TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Remove(name string) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.Remove(name)
	}
}

// RemoveAll TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) RemoveAll(name string) (err error) {
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

// Rename TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Rename(oldpath, newpath string) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		// use PosixRename to overwrite oldpath
		return s.PosixRename(oldpath, newpath)
	}
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

// WriteFile TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) WriteFile(name string, data []byte, perm os.FileMode) (err error) {
	var s *sftp.Client
	var f *sftp.File

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

// ReadFile TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) ReadFile(name string) (b []byte, err error) {
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

// LastError TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) LastError() error {
	// if the failure was a while back, try again (XXX crude)
	if h.failed != nil && !h.lastAttempt.IsZero() && time.Since(h.lastAttempt) > 5*time.Second {
		_, err := h.DialSSH()
		return err
	}
	return h.failed
}

// ServerVersion TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) ServerVersion() string {
	remote, err := h.DialSSH()
	if err != nil {
		return ""
	}
	return string(remote.ServerVersion())
}

// OS TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) OS() string {
	if strings.Contains(h.ServerVersion(), "windows") {
		return "windows"
	}
	return "linux"
}

// GetFs TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) GetFs() afero.Fs {
	client, err := h.DialSFTP()
	if err != nil {
		return nil
	}
	return sftpfs.New(client)
}

// Signal sends a signal to the remote pid and returns nil on success or
// os.ProcessDone if the process is not found.
func (h *SSHRemote) Signal(pid int, signal syscall.Signal) (err error) {
	sess, err := h.NewSession()
	if err != nil {
		return
	}
	defer sess.Close()

	err = sess.Run(fmt.Sprintf("kill -s %d %d", signal, pid))
	if err != nil {
		if exitErr, ok := errors.AsType[*ssh.ExitError](err); ok {
			switch exitErr.Waitmsg.ExitStatus() {
			case 0:
				err = nil
			case 1:
				err = os.ErrProcessDone
			default:
				// leave as is for now, but could be a permission error or something else
			}
		}
	}
	return
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

// Start starts a process on an SSH attached remote host h. It uses a
// shell and backgrounds and redirects. May not work on all remotes and
// for all processes. errfile has stdout/stderr appended to it, use
// '/dev/null' if no errfile is wanted.
func (h *SSHRemote) Start(cmd *exec.Cmd, options ...ProcessOption) (pid int, err error) {
	if h.OS() == "windows" {
		err = errors.New("cannot run remote commands on windows")
	}

	po := evalProcessOptions(options...)
	errfile := po.errfile

	sess, err := h.NewSession()
	if err != nil {
		return
	}

	// we have to convert cmd to a string ourselves as we have to quote any args
	// with spaces (like "Demo Gateway")
	//
	// given this is sent to a shell, we can quote everything blindly ?
	//
	// note that cmd.Args already has the command as Args[0], so no Path required
	var cmdstr = ""
	for _, a := range cmd.Args {
		cmdstr = fmt.Sprintf("%s %q", cmdstr, a)
	}

	// we can't use sess.Setenv() because openssh daemon blocks
	// non-whitelisted envs from being set
	envs := []string{}
	for _, e := range cmd.Env {
		if k, v, ok := strings.Cut(e, "="); ok {
			envs = append(envs, fmt.Sprintf("%s=%q", k, v))
		}
	}
	cmdstr = fmt.Sprintf(`cd %q; %s %s </dev/null >%q 2>&1 & echo $!`, cmd.Dir, strings.Join(envs, " "), cmdstr, errfile)

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
// stderr output
func (h *SSHRemote) Run(cmd *exec.Cmd, options ...ProcessOption) (output []byte, err error) {
	if h.OS() == "windows" {
		err = errors.New("cannot run remote commands on windows")
	}

	po := evalProcessOptions(options...)
	errfile := po.errfile

	sess, err := h.NewSession()
	if err != nil {
		return
	}

	// we have to convert cmd to a string ourselves as we have to quote any args
	// with spaces (like "Demo Gateway")
	//
	// given this is sent to a shell, we can quote everything blindly ?
	//
	// note that cmd.Args hosts the command as Args[0], so no Path required
	var cmdstr = ""
	for _, a := range cmd.Args {
		cmdstr = fmt.Sprintf("%s %q", cmdstr, a)
	}
	// pipe, err := sess.StdinPipe()
	// if err != nil {
	// 	return
	// }

	if errfile != "" {
		if !h.IsAbs(errfile) {
			errfile = path.Join(cmd.Dir, errfile)
		}
		e, err := h.Create(errfile, 0664)
		if err != nil {
			return []byte{}, err
		}
		defer e.Close()
		sess.Stderr = e
	}

	envs := []string{}
	for _, e := range cmd.Env {
		envs = append(envs, strconv.Quote(e))
	}
	cmdstr = fmt.Sprintf("cd %q && %s %s", cmd.Dir, strings.Join(cmd.Env, " "), cmdstr)

	return sess.Output(cmdstr)
}

// Uname TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) Uname() (os, arch string, err error) {
	if h.OS() == "windows" {
		err = errors.New("cannot run remote commands on windows")
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

// WalkDir TODO: Document its behavior, inputs, and effects.
func (h *SSHRemote) WalkDir(dir string, fn fs.WalkDirFunc) error {
	s, err := h.DialSFTP()
	if err != nil {
		return err
	}

	w := s.Walk(dir)
	for w.Step() {
		if w.Err() != nil {
			return w.Err()
		}
		p, _ := h.Rel(dir, w.Path())
		if err = fn(p, fs.FileInfoToDirEntry(w.Stat()), err); err != nil {
			if err == fs.SkipDir {
				break
			}
			return err
		}
	}
	return nil
}
