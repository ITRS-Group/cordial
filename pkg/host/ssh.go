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
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/awnumar/memguard"
	"github.com/pkg/sftp"
	"github.com/rs/zerolog/log"
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
	password    *memguard.Enclave // cannot use config.Plaintext because of import loop
	keys        []string
	failed      error
	lastAttempt time.Time
}

func NewSSHRemote(name string, options ...any) Host {
	r := &SSHRemote{
		name: name,
	}
	evalOptions(r, options...)
	return r
}

type SSHOptions func(*SSHRemote)

func evalOptions(r *SSHRemote, options ...any) {
	// defaults
	if u, err := user.Current(); err == nil {
		r.username = u.Username
	} else {
		r.username = os.Getenv("USER")
	}

	for _, opt := range options {
		switch o := opt.(type) {
		case SSHOptions:
			o(r)
		}
	}

}

func Username(username string) SSHOptions {
	return func(s *SSHRemote) {
		s.username = username
	}
}

func Password(password *memguard.Enclave) SSHOptions {
	return func(s *SSHRemote) {
		s.password = password
	}
}

func Port(port uint16) SSHOptions {
	return func(s *SSHRemote) {
		s.port = port
	}
}

func Hostname(hostname string) SSHOptions {
	return func(s *SSHRemote) {
		s.hostname = hostname
	}
}

// PrivateKeyFiles add the given paths as private key files to use for
// SSH connections. The files must (at this time) not be passphrase
// protected.
func PrivateKeyFiles(paths ...string) SSHOptions {
	return func(s *SSHRemote) {
		s.keys = append(s.keys, paths...)
	}
}

func (h *SSHRemote) Username() string {
	return h.username
}

func (s *SSHRemote) Hostname() string {
	return s.hostname
}

// load any/all the known private keys with no passphrase
func readSSHkeys(passphrase *memguard.Enclave, homedir string, files ...string) (signers []ssh.Signer) {
	for _, p := range files {
		key, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		signer, err := ssh.ParsePrivateKey(key)
		pperr := &ssh.PassphraseMissingError{}
		if err != nil && errors.As(err, &pperr) {
			err = nil
			pw, err := passphrase.Open()
			if err != nil {
				continue
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, pw.Bytes())
			pw.Destroy()

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
func sshConnect(dest, username string, password *memguard.Enclave, keyfiles ...string) (client *ssh.Client, err error) {
	var authmethods []ssh.AuthMethod
	var homedir string

	var u *user.User
	u, err = user.Current()
	if err != nil {
		return
	}

	homedir, err = os.UserHomeDir()
	if err != nil {
		homedir = u.HomeDir
	}

	if username == "" {
		username = u.Username
	}

	// XXX we need this because:
	// https://github.com/golang/go/issues/29286#issuecomment-1160958614
	kh, err := knownhosts.NewDB(path.Join(homedir, userSSHdir, "known_hosts"))
	if err != nil {
		return
	}

	keys := kh.HostKeys(dest)
	if len(keys) == 0 {
		log.Fatal().Msgf("no public key found for %s", dest)
	}

	if agentClient := sshConnectAgent(); agentClient != nil {
		authmethods = append(authmethods, ssh.PublicKeysCallback(agentClient.Signers))
	}

	// either private keys (using password as passphrase) or password
	// but not both
	if signers := readSSHkeys(password, homedir, keyfiles...); len(signers) > 0 {
		authmethods = append(authmethods, ssh.PublicKeys(signers...))
	} else if password != nil && password.Size() > 0 {
		l, _ := password.Open()
		defer l.Destroy()
		authmethods = append(authmethods, ssh.Password(l.String()))
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
		log.Fatal().Msgf("host key has changed for %s", dest)
	}
	return
}

// Dial connects to a remote host using ssh and returns an *ssh.Client
// on success. Each connection is cached and returned if found without
// checking if it is still valid. To remove a session call Close()
func (h *SSHRemote) Dial() (sc *ssh.Client, err error) {
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
// *sftp.Client is successful
func (h *SSHRemote) DialSFTP() (f *sftp.Client, err error) {
	if h == nil {
		err = ErrInvalidArgs
		return
	}

	val, ok := sftpSessions.Load(h.name)
	if ok {
		f = val.(*sftp.Client)
	} else {
		var s *ssh.Client
		if s, err = h.Dial(); err != nil {
			h.failed = err
			h.lastAttempt = time.Now()
			return
		}
		// disable concurrent reads as they mess with file offsets when using io.Copy()
		if f, err = sftp.NewClient(s, sftp.UseConcurrentReads(false)); err != nil {
			h.failed = err
			h.lastAttempt = time.Now()
			return
		}
		sftpSessions.Store(h.name, f)
	}
	return
}

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

// IsLocal returns true if h is local, which for SSH is always false
func (h *SSHRemote) IsLocal() bool {
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

	_, err = h.Dial()
	return err == nil, err
}

func (h *SSHRemote) String() string {
	return h.name
}

func (h *SSHRemote) Abs(dir string) (string, error) {
	if s, err := h.DialSFTP(); err != nil {
		return "", err
	} else {
		return s.RealPath(dir)
	}
}

func (h *SSHRemote) Getwd() (dir string, err error) {
	if s, err := h.DialSFTP(); err != nil {
		return "", err
	} else {
		return s.Getwd()
	}
}

func (h *SSHRemote) Symlink(oldname, newname string) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.Symlink(oldname, newname)
	}
}

func (h *SSHRemote) Readlink(file string) (string, error) {
	if s, err := h.DialSFTP(); err != nil {
		return "", err
	} else {
		return s.ReadLink(file)
	}
}

func (h *SSHRemote) MkdirAll(p string, perm os.FileMode) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.MkdirAll(p)
	}
}

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

func (h *SSHRemote) Chtimes(path string, atime time.Time, mtime time.Time) (err error) {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.Chtimes(path, atime, mtime)
	}
}

func (h *SSHRemote) Lchtimes(path string, atime time.Time, mtime time.Time) (err error) {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		// try plain chtimes
		return s.Chtimes(path, atime, mtime)
	}
}

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

func (h *SSHRemote) Remove(name string) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		return s.Remove(name)
	}
}

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

func (h *SSHRemote) Rename(oldpath, newpath string) error {
	if s, err := h.DialSFTP(); err != nil {
		return err
	} else {
		// use PosixRename to overwrite oldpath
		return s.PosixRename(oldpath, newpath)
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

func (h *SSHRemote) Glob(pattern string) ([]string, error) {
	if s, err := h.DialSFTP(); err != nil {
		return []string{}, err
	} else {
		return s.Glob(pattern)
	}
}

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
	sort.Slice(f, func(i, j int) bool {
		return f[i].Name() < f[j].Name()
	})
	for _, d := range f {
		dirs = append(dirs, fs.FileInfoToDirEntry(d))
	}
	return
}

func (h *SSHRemote) Open(name string) (io.ReadSeekCloser, error) {
	if s, err := h.DialSFTP(); err != nil {
		return nil, err
	} else {
		return s.Open(name)
	}
}

func (h *SSHRemote) HostPath(p string) string {
	return fmt.Sprintf("%s:%s", h, p)
}

// TempDir returns a path on the remote to a temporary directory
//
// BUG This is currently broken - hardwired values for now
func (h *SSHRemote) TempDir() string {
	if strings.Contains(h.ServerVersion(), "windows") {
		return `C:\TEMP`
	}
	return "/tmp"
}

func (h *SSHRemote) LastError() error {
	// if the failure was a while back, try again (XXX crude)
	if h.failed != nil && !h.lastAttempt.IsZero() && time.Since(h.lastAttempt) > 5*time.Second {
		_, err := h.Dial()
		return err
	}
	return h.failed
}

func (h *SSHRemote) ServerVersion() string {
	remote, err := h.Dial()
	if err != nil {
		return ""
	}
	return string(remote.ServerVersion())
}

func (h *SSHRemote) GetFs() afero.Fs {
	client, err := h.DialSFTP()
	if err != nil {
		return nil
	}
	return sftpfs.New(client)
}

func (h *SSHRemote) Signal(pid int, signal syscall.Signal) (err error) {
	sess, err := h.NewSession()
	if err != nil {
		return
	}
	defer sess.Close()

	sess.CombinedOutput(fmt.Sprintf("kill -s %d %d", signal, pid))
	return
}

// NewSession wraps ssh.NewSession but does some retries
func (h *SSHRemote) NewSession() (sess *ssh.Session, err error) {
	rem, err := h.Dial()
	if err != nil {
		err = fmt.Errorf("Start: %w during Dial()", err)
		return
	}

	// the number of sessions is always limited by config on the remote
	// server, but we don't know what that limit is, so retry a few
	// times with a small delay
	var i int
	for i = 0; i < 10; i++ {
		sess, err = rem.NewSession()
		if err == nil {
			break
		}
		var ocerr *ssh.OpenChannelError
		if !errors.As(err, &ocerr) {
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
func (h *SSHRemote) Start(cmd *exec.Cmd, errfile string) (err error) {
	if strings.Contains(h.ServerVersion(), "windows") {
		err = errors.New("cannot run remote commands on windows")
	}

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
	pipe, err := sess.StdinPipe()
	if err != nil {
		return
	}

	if err = sess.Shell(); err != nil {
		return
	}
	fmt.Fprintf(pipe, "cd %q\n", cmd.Dir)
	for _, e := range cmd.Env {
		fmt.Fprintln(pipe, "export", e)
	}
	fmt.Fprintf(pipe, "%s >> %q 2>&1 &\n", cmdstr, errfile)
	fmt.Fprintln(pipe, "exit")
	return sess.Wait()
}

// Run starts a process on an SSH attached remote host h. It uses a
// shell and waits for the process status before returning. It returns
// the output and any error. errfile is an optional (remote) file for
// stderr output
func (h *SSHRemote) Run(cmd *exec.Cmd, errfile string) (output []byte, err error) {
	if strings.Contains(h.ServerVersion(), "windows") {
		err = errors.New("cannot run remote commands on windows")
	}

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

func (h *SSHRemote) Uname() (os, arch string, err error) {
	if strings.Contains(h.ServerVersion(), "windows") {
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
	for _, w := range bytes.Fields(out) {
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
		p, _ := filepath.Rel(dir, w.Path())
		if err = fn(p, fs.FileInfoToDirEntry(w.Stat()), err); err != nil {
			if err == fs.SkipDir {
				break
			}
			return err
		}
	}
	return nil
}
