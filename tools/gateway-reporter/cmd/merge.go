/*
Copyright © 2023 ITRS Group

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
func mergeConfig(install, setup string) (output []byte, err error) {
	var gatewayPath string

	// run a gateway with -dump-xml and consume the result, discard the heading
	st, err := os.Stat(install)
	if err != nil {
		return
	}
	if st.IsDir() {
		gatewayPath = path.Join(install, gatewayBinary)
	} else {
		gatewayPath = install
		install = filepath.Dir(install)
	}
	cmd := exec.Command(
		gatewayPath,
		"-resources-dir",
		path.Join(install, "resources"),
		"-nolog",
		"-skip-cache",
		"-setup",
		setup,
		"-dump-xml",
	)

	cmd.Dir = filepath.Dir(setup)
	cmd.Env = os.Environ()

	var err2 error
	output, err = host.Localhost.Run(cmd, "merge-errors.txt")
	if err != nil {
		// skip errors for now
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
