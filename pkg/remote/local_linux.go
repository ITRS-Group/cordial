package remote

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/rs/zerolog/log"
)

func procSetupOS(cmd *exec.Cmd, out *os.File) {
	var err error

	// if we've set-up privs at all, set the redirection output file to the same
	if cmd.SysProcAttr != nil && cmd.SysProcAttr.Credential != nil {
		if err = out.Chown(int(cmd.SysProcAttr.Credential.Uid), int(cmd.SysProcAttr.Credential.Gid)); err != nil {
			log.Error().Err(err).Msg("chown")
		}
	}
	// detach process by creating a session (fixed start + log)
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}
