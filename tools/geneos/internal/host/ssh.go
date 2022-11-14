package host

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

const userSSHdir = ".ssh"

var sshSessions sync.Map
var sftpSessions sync.Map

var privateKeys = ""

// load any/all the known private keys with no passphrase
func readSSHkeys(homedir string) (signers []ssh.Signer) {
	for _, keyfile := range strings.Split(privateKeys, ",") {
		path := filepath.Join(homedir, userSSHdir, keyfile)
		key, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			continue
		}
		log.Debug().Msgf("loaded private key from %s", path)
		signers = append(signers, signer)
	}
	return
}

func sshConnect(dest, user string) (client *ssh.Client, err error) {
	var khCallback ssh.HostKeyCallback
	var authmethods []ssh.AuthMethod
	var signers []ssh.Signer
	var agentClient agent.ExtendedAgent // XXX this should be package global?
	var homedir string

	homedir, err = os.UserHomeDir()
	if err != nil {
		return
	}

	if khCallback == nil {
		k := filepath.Join(homedir, userSSHdir, "known_hosts")
		khCallback, err = knownhosts.New(k)
		if err != nil {
			log.Debug().Msg("cannot load ssh known_hosts file, ssh will not be available.")
			return
		}
	}

	if agentClient == nil {
		agentClient = sshConnectAgent()
	}

	if signers == nil {
		signers = readSSHkeys(homedir)
	}

	if agentClient != nil {
		authmethods = append(authmethods, ssh.PublicKeysCallback(agentClient.Signers))
	}
	if signers == nil {
		authmethods = append(authmethods, ssh.PublicKeys(signers...))
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authmethods,
		HostKeyCallback: khCallback,
		Timeout:         5 * time.Second,
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoED25519, ssh.CertAlgoED25519v01,

			ssh.CertSigAlgoRSASHA2512v01, ssh.CertSigAlgoRSASHA2256v01,
			ssh.CertSigAlgoRSAv01, ssh.CertAlgoDSAv01, ssh.CertAlgoECDSA256v01,
			ssh.CertAlgoECDSA384v01, ssh.CertAlgoECDSA521v01,

			ssh.KeyAlgoECDSA256, ssh.KeyAlgoECDSA384, ssh.KeyAlgoECDSA521,
			ssh.SigAlgoRSASHA2512, ssh.SigAlgoRSASHA2256,
			ssh.SigAlgoRSA, ssh.KeyAlgoDSA,
		},
	}
	return ssh.Dial("tcp", dest, config)
}

func (h *Host) Dial() (s *ssh.Client, err error) {
	if h.failed != nil {
		err = h.failed
		return
	}
	dest := h.GetString("hostname") + ":" + h.GetString("port")
	user := h.GetString("username")
	val, ok := sshSessions.Load(user + "@" + dest)
	if ok {
		s = val.(*ssh.Client)
	} else {
		log.Debug().Msgf("ssh connect to %s as %s", dest, user)
		s, err = sshConnect(dest, user)
		if err != nil {
			log.Debug().Err(err).Msg("")
			h.failed = err
			return
		}
		log.Debug().Msgf("host opened %s %s %s", h.GetString("name"), dest, user)
		sshSessions.Store(user+"@"+dest, s)
	}
	return
}

func (h *Host) Close() {
	h.CloseSFTP()

	dest := h.GetString("hostname") + ":" + h.GetString("port")
	user := h.GetString("username")
	val, ok := sshSessions.Load(user + "@" + dest)
	if ok {
		s := val.(*ssh.Client)
		s.Close()
		sshSessions.Delete(user + "@" + dest)
	}
}

// succeed or fatal
func (h *Host) DialSFTP() (f *sftp.Client, err error) {
	if h.failed != nil {
		err = h.failed
		return
	}
	dest := h.GetString("hostname") + ":" + h.GetString("port")
	user := h.GetString("username")
	val, ok := sftpSessions.Load(user + "@" + dest)
	if ok {
		f = val.(*sftp.Client)
	} else {
		var s *ssh.Client
		if s, err = h.Dial(); err != nil {
			h.failed = err
			return
		}
		if f, err = sftp.NewClient(s); err != nil {
			h.failed = err
			return
		}
		log.Debug().Msgf("remote opened %s", h.GetString("name"))
		sftpSessions.Store(user+"@"+dest, f)
	}
	return
}

func (h *Host) CloseSFTP() {
	dest := h.GetString("hostname") + ":" + h.GetString("port")
	user := h.GetString("username")
	val, ok := sftpSessions.Load(user + "@" + dest)
	if ok {
		f := val.(*sftp.Client)
		f.Close()
		sftpSessions.Delete(user + "@" + dest)
	}
}
