package host

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
		logDebug.Println("loaded private key from", path)
		signers = append(signers, signer)
	}
	return
}

func sshConnect(dest, user string) (client *ssh.Client, err error) {
	var khCallback ssh.HostKeyCallback
	var authmethods []ssh.AuthMethod
	var signers []ssh.Signer
	var agentClient agent.ExtendedAgent
	var homedir string

	homedir, err = os.UserHomeDir()
	if err != nil {
		logError.Fatalln(err)
	}

	if khCallback == nil {
		k := filepath.Join(homedir, userSSHdir, "known_hosts")
		khCallback, err = knownhosts.New(k)
		if err != nil {
			logDebug.Println("cannot load ssh known_hosts file, ssh will not be available.")
			return
		}
	}

	if agentClient == nil {
		socket := os.Getenv("SSH_AUTH_SOCK")
		if socket != "" {
			sshAgent, err := net.Dial("unix", socket)
			if err != nil {
				log.Printf("Failed to open SSH_AUTH_SOCK: %v", err)
			} else {
				agentClient = agent.NewClient(sshAgent)
			}
		}
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
		s, err = sshConnect(dest, user)
		if err != nil {
			h.failed = err
			return
		}
		logDebug.Println("host opened", h.GetString("name"), dest, user)
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
		logDebug.Println("remote opened", h.GetString("name"))
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
