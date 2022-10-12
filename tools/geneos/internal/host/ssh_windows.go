package host

func sshConnectAgent() (agentClient agent.ExtendedAgent) {
	socket := `\\.\pipe\openssh-ssh-agent`
	if socket != "" {
		log.Debug().Msgf("connecting to agent on %s", socket)
		sshAgent, err := winio.DialPipe("unix", socket)
		if err != nil {
			log.Error().Msgf("Failed to connect to ssh agent: %v", err)
		} else {
			agentClient = agent.NewClient(sshAgent)
		}
	}
	return
}
