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
func parseArgs(cmd *cobra.Command, rawargs []string) (err error) {
	var wild bool
	var newnames []string

	var ct *geneos.Component
	var args, params []string

	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}
	a := cmd.Annotations
	a["args"] = "[]"
	a["params"] = "[]"

	if len(rawargs) == 0 && a["wildcard"] != "true" {
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

	log.Debug().Msgf("rawargs %v, params %v, ct %s", rawargs, params, a["ct"])

	if _, ok := a["ct"]; !ok {
		a["ct"] = ""
	}
	jsonargs, _ := json.Marshal(params)
	a["params"] = string(jsonargs)

	if a["wildcard"] == "false" {
		if len(rawargs) == 0 {
			return nil
		}
		if ct = geneos.ParseComponentName(rawargs[0]); ct == nil {
			jsonargs, _ := json.Marshal(rawargs)
			a["args"] = string(jsonargs)
			return
		}
		if a["ct"] == "" {
			a["ct"] = rawargs[0]
		}
		args = rawargs[1:]
	} else {
		defaultComponent := a["ct"]
		if defaultComponent == "" && len(rawargs) > 0 {
			defaultComponent = rawargs[0]
		}
		// work through wildcard options
		// if len(rawargs) == 0 {
		// 	// nothing
		// } else
		if ct = geneos.ParseComponentName(defaultComponent); ct == nil {
			// first arg is not a known type, so treat the rest as instance names
			args = rawargs
		} else {
			if a["ct"] == "" {
				a["ct"] = rawargs[0]
				args = rawargs[1:]
			} else {
				args = rawargs
			}
		}

		log.Debug().Msgf("args = %+v", args)
		if len(args) == 0 || (len(args) == 1 && args[0] == "all") {
			// no args also means all instances
			wild = true
			args = instance.AllNames(geneos.ALL, ct)
		} else {
			// expand each arg and save results to a new slice
			// if local == "", then all instances on host (e.g. @host)
			// if host == "all" (or none given), then check instance on all hosts
			// @all is not valid - should be no arg
			var nargs []string
			for _, arg := range args {
				// check if not valid first and leave unchanged, skip
				if !(strings.HasPrefix(arg, "@") || instance.ValidInstanceName(arg)) {
					log.Debug().Msgf("leaving unchanged: %s", arg)
					nargs = append(nargs, arg)
					continue
				}
				_, local, r := instance.SplitName(arg, geneos.ALL)
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
						rargs := instance.AllNames(r, ct)
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
						for _, ct := range ct.Range(geneos.RealComponents()...) {
							if i, err := instance.Get(ct, name); err == nil && i.Loaded() {
								nargs = append(nargs, name)
								matched = true
							}
						}
					}
					if !matched && instance.ValidInstanceName(arg) {
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
		if !instance.ValidInstanceName(name) {
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
	a["args"] = string(jsonargs)
	jsonparams, _ := json.Marshal(params)
	a["params"] = string(jsonparams)

	if a["wildcard"] == "false" {
		return
	}

	// if args is empty, find them all again. ct == None too?
	if len(args) == 0 && geneos.Root() != "" && !wild {
		args = instance.AllNames(geneos.ALL, ct)
		jsonargs, _ := json.Marshal(args)
		a["args"] = string(jsonargs)
	}

	log.Debug().Msgf("ct %s, args %v, params %v", ct, args, params)
	return
}

func CmdArgs(cmd *cobra.Command) (ct *geneos.Component, args []string) {
	log.Debug().Msgf("%s %v", cmd.Annotations, ct)
	ct = geneos.ParseComponentName(cmd.Annotations["ct"])
	if err := json.Unmarshal([]byte(cmd.Annotations["args"]), &args); err != nil {
		log.Debug().Err(err).Msg("")
	}
	return
}

func CmdArgsParams(cmd *cobra.Command) (ct *geneos.Component, args, params []string) {
	ct, args = CmdArgs(cmd)
	if err := json.Unmarshal([]byte(cmd.Annotations["params"]), &params); err != nil {
		log.Debug().Err(err).Msg("")
	}
	return
}
