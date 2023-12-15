/*
Copyright © 2023 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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
