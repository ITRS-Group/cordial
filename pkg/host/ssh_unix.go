//go:build !windows

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
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh/agent"
)

func sshConnectAgent() (agentClient agent.ExtendedAgent) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket != "" {
		if sshAgent, err := net.Dial("unix", socket); err == nil { // dial OK, return agent
			agentClient = agent.NewClient(sshAgent)
		}
	}
	return
}

// IsAbs run from unix will use path.IsAbs unless the remote is windows
// in which case it checks the volume name, stripping it and testing the
// rest of the path
func (h *SSHRemote) IsAbs(name string) bool {
	if strings.Contains(h.ServerVersion(), "windows") {
		n := filepath.VolumeName(name)
		if n == "" {
			return false
		}
		return filepath.IsAbs(filepath.ToSlash(strings.TrimPrefix(name, n)))
	}
	return filepath.IsAbs(name)
}
