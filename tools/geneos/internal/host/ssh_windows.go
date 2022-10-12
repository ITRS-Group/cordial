package host

import (
	"github.com/Microsoft/go-winio"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh/agent"
)

func sshConnectAgent() (agentClient agent.ExtendedAgent) {
	socket := `\\.\pipe\openssh-ssh-agent`
	if socket != "" {
		log.Debug().Msgf("connecting to agent on %s", socket)
		sshAgent, err := winio.DialPipe(socket, nil)
		if err != nil {
			log.Error().Msgf("Failed to connect to ssh agent: %v", err)
		} else {
			agentClient = agent.NewClient(sshAgent)
		}
	}
	return
}
