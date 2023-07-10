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

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// given a list of args (after command has been seen), check if first
// arg is a component type and de-dup the names. A name of "all" will
// will override the rest and result in a lookup being done
//
// args with an '=' should be checked and only allowed if there are names?
//
// support glob style wildcards for instance names - allow through, let loopCommand*
// deal with them
//
// process command args in a standard way
// flags will have been handled by another function before this one
// any args with '=' are treated as parameters
//
// a bare argument with a '@' prefix means all instance of type on a host
func parseArgs(command *cobra.Command, rawargs []string) (err error) {
	var wild bool
	var newnames []string

	var ct *geneos.Component
	var args, params []string
	h := geneos.GetHost(Hostname)

	if command.Annotations == nil {
		command.Annotations = make(map[string]string)
	}
	annotations := command.Annotations
	annotations["args"] = "[]"
	annotations["params"] = "[]"

	if len(rawargs) == 0 && annotations["wildcard"] != "true" {
		return nil
	}

	log.Debug().Msgf("rawargs: %s", rawargs)

	// filter in place - pull out all args containing '=' into params
	// after rebuild this should only apply to 'import'
	n := 0
	for _, a := range rawargs {
		// if !instance.ValidInstanceName(a) {
		if strings.Contains(a, "=") {
			params = append(params, a)
		} else {
			rawargs[n] = a
			n++
		}
	}
	rawargs = rawargs[:n]

	log.Debug().Msgf("rawargs %v, params %v, ct %s", rawargs, params, annotations["ct"])

	if _, ok := annotations["ct"]; !ok {
		annotations["ct"] = ""
	}
	jsonargs, _ := json.Marshal(params)
	annotations["params"] = string(jsonargs)

	if annotations["wildcard"] == "false" {
		if len(rawargs) == 0 {
			return nil
		}
		if ct = geneos.ParseComponent(rawargs[0]); ct == nil {
			jsonargs, _ := json.Marshal(rawargs)
			annotations["args"] = string(jsonargs)
			return
		}
		if annotations["ct"] == "" {
			annotations["ct"] = rawargs[0]
		}
		args = rawargs[1:]
	} else {
		defaultComponent := annotations["ct"]
		if defaultComponent == "" && len(rawargs) > 0 {
			defaultComponent = rawargs[0]
		}
		// work through wildcard options
		// if len(rawargs) == 0 {
		// 	// nothing
		// } else
		if ct = geneos.ParseComponent(defaultComponent); ct == nil {
			// first arg is not a known type, so treat the rest as instance names
			args = rawargs
		} else {
			if annotations["ct"] == "" {
				annotations["ct"] = rawargs[0]
				args = rawargs[1:]
			} else {
				args = rawargs
			}
		}

		log.Debug().Msgf("args = %+v", args)
		if len(args) == 0 || (len(args) == 1 && args[0] == "all") {
			// no args also means all instances
			wild = true
			args = instance.Names(h, ct)
		} else {
			// expand each arg and save results to a new slice
			// if local == "", then all instances on host (e.g. @host)
			// if host == "all" (or none given), then check instance on all hosts
			// @all is not valid - should be no arg
			var nargs []string
			for _, arg := range args {
				// check if not valid first and leave unchanged, skip
				if !(strings.HasPrefix(arg, "@") || instance.ValidName(arg)) {
					log.Debug().Msgf("leaving unchanged: %s", arg)
					nargs = append(nargs, arg)
					continue
				}
				_, local, r := instance.NameParts(arg, h)
				if !r.Exists() {
					log.Debug().Msgf("%s - host not found", arg)
					// we have tried to match something and it may result in an empty list
					// so don't re-process
					return fmt.Errorf("host %q not found", r)
				}

				log.Debug().Msgf("split %s into: %s %s", arg, local, r.String())
				if local == "" {
					// only a '@host' in arg
					if r.Exists() {
						rargs := instance.Names(r, ct)
						nargs = append(nargs, rargs...)
						wild = true
					}
				} else if r == geneos.ALL {
					// no '@host' in arg
					var matched bool
					for _, rem := range geneos.AllHosts() {
						wild = true
						log.Debug().Msgf("checking host %s for %s", rem.String(), local)
						name := local + "@" + rem.String()
						for _, ct := range ct.OrList(geneos.RealComponents()...) {
							if i, err := instance.Get(ct, name); err == nil && i.Loaded() {
								nargs = append(nargs, name)
								matched = true
							}
						}
					}
					if !matched && instance.ValidName(arg) {
						// move the unknown unchanged - file or url - arg so it can later be pushed to params
						// do not set 'wild' though?
						log.Debug().Msgf("%s not found, saving to params", arg)
						nargs = append(nargs, arg)
					}
				} else {
					// save unchanged arg, may be param
					nargs = append(nargs, arg)
					// wild = true
				}
			}
			args = nargs
		}
	}

	log.Debug().Msgf("ct %s, args %v, params %v", ct, args, params)

	m := make(map[string]bool, len(args))
	// traditional loop because we can't modify args in a loop to skip
	for i := 0; i < len(args); i++ {
		name := args[i]
		// filter name here
		if !wild && instance.ReservedName(name) {
			log.Fatal().Msgf("%q is reserved name", name)
		}
		// move unknown args to params
		if !instance.ValidName(name) {
			params = append(params, name)
			continue
		}
		// ignore duplicates (not params above)
		if m[name] {
			continue
		}
		newnames = append(newnames, name)
		m[name] = true
	}
	args = newnames

	jsonargs, _ = json.Marshal(args)
	annotations["args"] = string(jsonargs)
	jsonparams, _ := json.Marshal(params)
	annotations["params"] = string(jsonparams)

	if annotations["wildcard"] == "false" {
		return
	}

	// if args is empty, find them all again. ct == None too?
	if len(args) == 0 && (geneos.Root() != "" || len(geneos.RemoteHosts(false)) > 0) && !wild {
		args = instance.Names(h, ct)
		jsonargs, _ := json.Marshal(args)
		annotations["args"] = string(jsonargs)
	}

	log.Debug().Msgf("ct %s, args %v, params %v", ct, args, params)
	return
}

func CmdArgs(command *cobra.Command) (ct *geneos.Component, args []string) {
	log.Debug().Msgf("%s %v", command.Annotations, ct)
	ct = geneos.ParseComponent(command.Annotations["ct"])
	if err := json.Unmarshal([]byte(command.Annotations["args"]), &args); err != nil {
		log.Debug().Err(err).Msg("")
	}
	return
}

func CmdArgsParams(command *cobra.Command) (ct *geneos.Component, args, params []string) {
	ct, args = CmdArgs(command)
	if err := json.Unmarshal([]byte(command.Annotations["params"]), &params); err != nil {
		log.Debug().Err(err).Msg("")
	}
	return
}
