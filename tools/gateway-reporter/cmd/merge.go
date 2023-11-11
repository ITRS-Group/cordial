package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/host"
)

const gatewayBinary = "gateway2.linux_64"

// merge a gateway config and return the result for parsing
func mergeConfig(installDir, gatewayBin, setupFile string) (output []byte, err error) {
	// run a gateway with -dump-xml and consume the result, discard the heading
	gatewayPath := path.Join(installDir, gatewayBinary)
	if gatewayBin != "" {
		gatewayPath = gatewayBin
	}
	cmd := exec.Command(
		gatewayPath,
		"-resources-dir",
		path.Join(installDir, "resources"),
		"-nolog",
		"-skip-cache",
		"-setup",
		setupFile,
		"-dump-xml",
	)

	cmd.Dir = filepath.Dir(setupFile)
	cmd.Env = os.Environ()

	var err2 error
	output, err = host.Localhost.Run(cmd, "merge-errors.txt")
	if err != nil {
		// skip errors for now
		log.Debug().Err(err).Msg(string(output))
		err2 = err
		err = nil
	}

	i := bytes.Index(output, []byte("<?xml"))
	if i == -1 {
		err = err2
		log.Fatal().Err(err).Msg(string(output))
	}
	output = output[i:]
	return
}
