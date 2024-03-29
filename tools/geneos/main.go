/*
Copyright © 2022 ITRS Group

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
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/san"
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
