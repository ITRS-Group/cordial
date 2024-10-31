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

package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/awnumar/memguard"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"

	// import subsystems here for command registration
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/aescmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/cfgcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/hostcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/initcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/pkgcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/tlscmd"

	// each component type registers itself when imported here
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ac2"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ca3"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/fileagent"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/floating"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/minimal"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/san"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ssoagent"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/webserver"
)

func init() {
	// strip any whitespace from the embedded VERSION value as early as
	// possible
	cordial.VERSION = strings.TrimSpace(cordial.VERSION)
}

func main() {
	memguard.CatchInterrupt()
	defer memguard.Purge()

	execname := path.Base(os.Args[0])

	// if the executable does not have a `ctl` suffix then execute the
	// underlying code directly
	if !strings.HasSuffix(execname, "ctl") {
		cmd.Execute()
		memguard.SafeExit(0)
	}

	// otherwise emulate core ctl commands
	ct := geneos.ParseComponent(strings.TrimSuffix(execname, "ctl"))
	if len(os.Args) > 1 {
		name := os.Args[1]
		switch name {
		case "list":
			os.Args = []string{execname, "ls", ct.String()}
			cmd.Execute()
		case "create":
			fmt.Printf("create not support, please use 'geneos add %s ...'\n", ct)
			os.Exit(1)
		default:
			if len(os.Args) > 2 {
				function := os.Args[2]
				switch function {
				case "start", "stop", "restart", "command", "log", "details", "refresh", "status", "delete":
					os.Args = []string{execname, function, ct.String(), name}
					cmd.Execute()
				default:
					fmt.Printf("'%s' not supported\n", function)
				}
			} else {
				fmt.Println("unknown command")
			}
		}
	}
}
