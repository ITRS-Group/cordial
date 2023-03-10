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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/pkg/sftp"
	"github.com/rs/zerolog/log"
	"github.com/skeema/knownhosts"
	"golang.org/x/crypto/ssh"
)

const userSSHdir = ".ssh"

var sshSessions sync.Map
var sftpSessions sync.Map

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
func (h *Host) Dial() (s *ssh.Client, err error) {
	if h.failed != nil {
		err = h.failed
		return
	}
	username := h.GetString("username")
	if username == "" {
		log.Error().Msgf("username not set for remote %s", h)
		return nil, ErrInvalidArgs
	}
	hostname := h.GetString("hostname")
	port := h.GetString("port", config.Default("22"))
	if hostname == "" {
		log.Error().Msgf("hostname not set for remote %s", h)
		return nil, ErrInvalidArgs
	}

	dest := hostname + ":" + port
	val, ok := sshSessions.Load(username + "@" + dest)
	if ok {
		s = val.(*ssh.Client)
	} else {
		log.Debug().Msgf("ssh connect to %s as %s", dest, username)
		s, err =

			sshConnect(dest, username, h.GetByteSlice("password"), strings.Split(h.GetString("sshkeys"), ",")...)
		if err != nil {
			log.Debug().Err(err).Msg("")
			h.failed = err
			h.lastAttempt = time.Now()
			return
		}
		log.Debug().Msgf("host opened %s %s %s", h.GetString("name"), dest, username)
		sshSessions.Store(username+"@"+dest, s)
	}
	return
}

func (h *Host) Close() {
	h.CloseSFTP()

	port := h.GetString("port", config.Default("22"))
	dest := h.GetString("hostname") + ":" + port
	user := h.GetString("username")
	val, ok := sshSessions.Load(user + "@" + dest)
	if ok {
		s := val.(*ssh.Client)
		s.Close()
		sshSessions.Delete(user + "@" + dest)
	}
}

// DialSFTP connects to the remote host using SSH and returns an
// *sftp.Client is successful
func (h *Host) DialSFTP() (f *sftp.Client, err error) {
	if h.failed != nil {
		err = h.failed
		return
	}
	user := h.GetString("username")
	if user == "" {
		log.Error().Msgf("username not set for remote %s", h)
		return nil, ErrInvalidArgs
	}
	hostname := h.GetString("hostname")
	port := h.GetString("port", config.Default("22"))
	if hostname == "" {
		log.Error().Msgf("hostname not set for remote %s", h)
		return nil, ErrInvalidArgs
	}

	dest := hostname + ":" + port
	val, ok := sftpSessions.Load(user + "@" + dest)
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
		log.Debug().Msgf("remote opened %s", h.GetString("name"))
		sftpSessions.Store(user+"@"+dest, f)
	}
	return
}

func (h *Host) CloseSFTP() {
	port := h.GetString("port", config.Default("22"))
	dest := h.GetString("hostname") + ":" + port
	user := h.GetString("username")
	val, ok := sftpSessions.Load(user + "@" + dest)
	if ok {
		f := val.(*sftp.Client)
		f.Close()
		sftpSessions.Delete(user + "@" + dest)
	}
}
