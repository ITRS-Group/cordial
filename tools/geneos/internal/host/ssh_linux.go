package host

import (
	"net"
	"os"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh/agent"
)

func sshConnectAgent() (agentClient agent.ExtendedAgent) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket != "" {
		log.Debug().Msgf("connecting to agent on %s", socket)
		sshAgent, err := net.Dial("unix", socket)
		if err != nil {
			log.Error().Msgf("Failed to connect to ssh agent: %v", err)
		} else {
			agentClient = agent.NewClient(sshAgent)
		}
	}
	return
}
