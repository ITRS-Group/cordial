package process

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func setCredentialsFromUsername(cmd *exec.Cmd, username string) (err error) {
	if username == "" {
		return nil
	}
	current, err := user.Current()
	if err != nil {
		return err
	}
	currentName := current.Username
	if _, account, ok := strings.Cut(currentName, "\\"); ok {
		currentName = account
	}
	requestedName := username
	if _, account, ok := strings.Cut(requestedName, "\\"); ok {
		requestedName = account
	}
	if strings.EqualFold(username, current.Username) ||
		(!strings.ContainsAny(username, "\\@") && strings.EqualFold(requestedName, currentName)) {
		return nil
	}
	return fmt.Errorf("starting as a different Windows user requires credentials: %w", os.ErrPermission)
}
