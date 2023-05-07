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

package remote

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/spf13/afero"
	"github.com/spf13/afero/sftpfs"

	"github.com/pkg/sftp"
	"github.com/rs/zerolog/log"
	"github.com/skeema/knownhosts"
	"golang.org/x/crypto/ssh"
)

const userSSHdir = ".ssh"

var sshSessions sync.Map
var sftpSessions sync.Map

type SSHRemote struct {
	name        string
	username    string
	hostname    string
	port        uint16
	password    []byte
	keys        []string
	failed      error
	lastAttempt time.Time
}

func NewSSHRemote(name string, options ...SSHOptions) Remote {
	r := &SSHRemote{
		name: name,
	}
	evalOptions(r, options...)
	return r
}

type SSHOptions func(*SSHRemote)

func evalOptions(r *SSHRemote, options ...SSHOptions) {
	// defaults
	u, _ := user.Current()
	r.username = u.Username

	for _, opt := range options {
		opt(r)
	}

}

func Username(username string) SSHOptions {
	return func(s *SSHRemote) {
		s.username = username
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

func Keys(paths ...string) SSHOptions {
	return func(s *SSHRemote) {
		s.keys = append(s.keys, paths...)
	}
}

// load any/all the known private keys with no passphrase
func readSSHkeys(homedir string, morekeys ...string) (signers []ssh.Signer) {
	files := strings.Split(config.GetString("privateKeys"), ",")
	for i, f := range files {
		files[i] = filepath.Join(homedir, userSSHdir, f)
	}
	for _, k := range morekeys {
		if k != "" {
			files = append(files, k)
		}
	}

	for _, path := range files {
		log.Debug().Msgf("trying to read private key %s", path)
		key, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			log.Debug().Err(err).Msg("")
			continue
		}
		log.Debug().Msgf("loaded private key from %s", path)
		signers = append(signers, signer)
	}
	return
}

func sshConnect(dest, user string, password []byte, keyfiles ...string) (client *ssh.Client, err error) {
	var authmethods []ssh.AuthMethod
	var homedir string

	homedir, err = os.UserHomeDir()
	if err != nil {
		log.Debug().Msg("user has no home directory, ssh will not be available.")
		return
	}

	// XXX we need this because https://github.com/golang/go/issues/29286#issuecomment-1160958614
	kh, err := knownhosts.New(filepath.Join(homedir, userSSHdir, "known_hosts"))
	if err != nil {
		log.Debug().Msg("cannot load ssh known_hosts file, ssh will not be available.")
		return
	}

	if agentClient := sshConnectAgent(); agentClient != nil {
		authmethods = append(authmethods, ssh.PublicKeysCallback(agentClient.Signers))
		log.Debug().Msg("added ssh agent to auth methods")
	}

	if signers := readSSHkeys(homedir, keyfiles...); len(signers) > 0 {
		authmethods = append(authmethods, ssh.PublicKeys(signers...))
		log.Debug().Msgf("added %d private key(s) to auth methods", len(signers))
	}

	if len(password) > 0 {
		authmethods = append(authmethods, ssh.Password(string(password)))
		log.Debug().Msg("added password to auth methods")
	}

	config := &ssh.ClientConfig{
		User:              user,
		Auth:              authmethods,
		HostKeyCallback:   kh.HostKeyCallback(),
		HostKeyAlgorithms: kh.HostKeyAlgorithms(dest),
		Timeout:           5 * time.Second,
	}
	return ssh.Dial("tcp", dest, config)
}

// Dial connects to a remote host using ssh and returns an *ssh.Client
// on success. Each connection is cached and returned if found without
// checking if it is still valid. To remove a session call Close()
func (h *SSHRemote) Dial() (s *ssh.Client, err error) {
	if h == nil {
		err = ErrInvalidArgs
		return
	}

	if h.failed != nil {
		err = h.failed
		return
	}
	if h.username == "" {
		log.Error().Msgf("username not set for remote %s", h)
		return nil, ErrInvalidArgs
	}
	if h.hostname == "" {
		log.Error().Msgf("hostname not set for remote %s", h)
		return nil, ErrInvalidArgs
	}
	if h.port == 0 {
		h.port = 22
	}

	dest := fmt.Sprintf("%s:%d", h.hostname, h.port)
	val, ok := sshSessions.Load(h.name)
	if ok {
		s = val.(*ssh.Client)
	} else {
		log.Debug().Msgf("ssh connect to %s as %s", dest, h.username)
		s, err = sshConnect(dest, h.username, h.password, h.keys...)
		if err != nil {
			log.Debug().Err(err).Msg("")
			h.failed = err
			h.lastAttempt = time.Now()
			return
		}
		log.Debug().Msgf("host opened %s %s %s", h.name, dest, h.username)
		sshSessions.Store(h.name, s)
	}
	return
}

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
		if f, err = sftp.NewClient(s); err != nil {
			h.failed = err
			h.lastAttempt = time.Now()
			return
		}
		log.Debug().Msgf("remote opened %s", h.name)
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

func (h *SSHRemote) IsLocal() bool {
	return false
}

func (h *SSHRemote) IsAvailable() bool {
	_, err := h.Dial()
	return err == nil
}

func (h *SSHRemote) String() string {
	return h.name
}

func (h *SSHRemote) Symlink(oldname, newname string) (err error) {
	var s *sftp.Client

	if s, err = h.DialSFTP(); err != nil {
		return
	}
	return s.Symlink(oldname, newname)
}

func (h *SSHRemote) Readlink(file string) (link string, err error) {
	var s *sftp.Client
	if s, err = h.DialSFTP(); err != nil {
		return
	}
	return s.ReadLink(file)
}

func (h *SSHRemote) MkdirAll(path string, perm os.FileMode) (err error) {
	var s *sftp.Client
	if s, err = h.DialSFTP(); err != nil {
		return
	}
	return s.MkdirAll(path)
}

func (h *SSHRemote) Chown(name string, uid, gid int) (err error) {
	var s *sftp.Client
	if s, err = h.DialSFTP(); err != nil {
		return
	}
	return s.Chown(name, uid, gid)
}

// change the symlink ownership on local system, issue chown for remotes
func (h *SSHRemote) Lchown(name string, uid, gid int) (err error) {
	var s *sftp.Client
	if s, err = h.DialSFTP(); err != nil {
		return
	}
	return s.Chown(name, uid, gid)
}

func (h *SSHRemote) Create(path string, perms fs.FileMode) (out io.WriteCloser, err error) {
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
	return
}

func (h *SSHRemote) Remove(name string) (err error) {
	var s *sftp.Client
	if s, err = h.DialSFTP(); err != nil {
		return
	}
	return s.Remove(name)
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

func (h *SSHRemote) Rename(oldpath, newpath string) (err error) {
	var s *sftp.Client
	if s, err = h.DialSFTP(); err != nil {
		return
	}
	// use PosixRename to overwrite oldpath
	return s.PosixRename(oldpath, newpath)
}

// Stat wraps the os.Stat and sftp.Stat functions
func (h *SSHRemote) Stat(name string) (f fs.FileInfo, err error) {
	var sf *sftp.Client
	if sf, err = h.DialSFTP(); err != nil {
		return
	}
	return sf.Stat(name)
}

// Lstat wraps the os.Lstat and sftp.Lstat functions
func (h *SSHRemote) Lstat(name string) (f fs.FileInfo, err error) {
	var sf *sftp.Client
	if sf, err = h.DialSFTP(); err != nil {
		return
	}
	return sf.Lstat(name)
}

func (h *SSHRemote) Glob(pattern string) (paths []string, err error) {
	var s *sftp.Client
	if s, err = h.DialSFTP(); err != nil {
		return
	}
	return s.Glob(pattern)
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

func (h *SSHRemote) Open(name string) (f io.ReadSeekCloser, err error) {
	var s *sftp.Client
	if s, err = h.DialSFTP(); err != nil {
		return
	}
	f, err = s.Open(name)
	return
}

func (h *SSHRemote) Run(name string, args ...string) (output []byte, err error) {
	remote, err := h.Dial()
	if err != nil {
		return
	}
	session, err := remote.NewSession()
	if err != nil {
		return
	}
	return session.Output(name)
}

func (h *SSHRemote) Path(path string) string {
	return fmt.Sprintf("%s:%s", h, path)
}

func (h *SSHRemote) Failed() bool {
	if h == nil {
		return false
	}
	// if the failure was a while back, try again (XXX crude)
	if !h.lastAttempt.IsZero() && time.Since(h.lastAttempt) > 5*time.Second {
		return false
	}
	return h.failed != nil
}

func (h *SSHRemote) ServerVersion() string {
	remote, err := h.Dial()
	if err != nil {
		return ""
	}
	return string(remote.ServerVersion())
}

func (h *SSHRemote) NewAferoFS() afero.Fs {
	client, err := h.DialSFTP()
	if err != nil {
		log.Error().Msgf("connection to %s failed", h)
		return nil
	}
	return sftpfs.New(client)
}

func (h *SSHRemote) Signal(pid int, signal syscall.Signal) (err error) {
	rem, err := h.Dial()
	if err != nil {
		return
	}
	sess, err := rem.NewSession()
	if err != nil {
		return
	}
	defer sess.Close()

	_, err = sess.CombinedOutput(fmt.Sprintf("kill -s %d %d", signal, pid))
	return
}

func (h *SSHRemote) Start(cmd *exec.Cmd, env []string, username, home, errfile string) (err error) {
	// rUsername := r.GetString("username")
	// if rUsername != username && username != "" {
	// 	return 0, fmt.Errorf("cannot run remote process as a different user (%q != %q)", rUsername, username)
	// }
	rem, err := h.Dial()
	if err != nil {
		return
	}
	sess, err := rem.NewSession()
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
	pipe, err := sess.StdinPipe()
	if err != nil {
		return
	}

	if err = sess.Shell(); err != nil {
		return
	}
	fmt.Fprintln(pipe, "cd", home)
	for _, e := range env {
		fmt.Fprintln(pipe, "export", e)
	}
	fmt.Fprintf(pipe, "%s > %q 2>&1 &", cmdstr, errfile)
	fmt.Fprintln(pipe, "exit")
	sess.Close()
	// wait a short while for remote to catch-up
	time.Sleep(250 * time.Millisecond)

	return
}
