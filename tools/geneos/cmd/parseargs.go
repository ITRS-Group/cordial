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

package cmd

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// Command annotation types for command behaviour
//
// Annotations should be read-only. Currently they are not completely.
const (
	CmdGlobNames    = "expand"       // "true" or "false" - pass all names through a path.Match style lookup
	CmdReplacedBy   = "replacedby"   // deprecated command alias
	CmdRequireHome  = "needshomedir" // "true" or "false"
	CmdNoneMeansAll = "wildcard"     // "true", "false", "explicit" (to match "all") or "none-or-all"
)

// type CommandFeatures struct {
// 	Glob         bool
// 	ReplacedBy   string
// 	RequireHome  bool
// 	NoneMeansAll string
// }

// ParseArgs does the heavy lifting of sorting out non-flag command line
// ares for the various commands. The results are passed back in the
// command Annotations map as `AnnotationNames` and `AnnotationsParams`
//
// given a list of args (after command has been seen), check if first
// arg is a component type and de-dup the names. A name of "all" will
// will override the rest and result in a lookup being done
//
// args with an '=' should be checked and only allowed if there are
// names?
//
// support glob style wildcards for instance names - allow through, let
// loopCommand* deal with them
//
// process command args in a standard way flags will have been handled
// by another function before this one any args with '=' are treated as
// parameters
//
// a bare argument with a '@' prefix means all instance of type on a
// host
func ParseArgs(command *cobra.Command, args []string) (err error) {
	var wild bool
	var ct *geneos.Component
	var names, params []string

	cd := cmddata(command)
	if cd == nil {
		return fmt.Errorf("command context not found")
	}

	// default host
	h := geneos.GetHost(Hostname)

	if command.Annotations == nil {
		command.Annotations = make(map[string]string)
	}

	an := command.Annotations

	if len(args) == 0 && an[CmdNoneMeansAll] != "true" {
		return nil
	}

	if an[CmdNoneMeansAll] == "true" {
		cd.Lock()
		cd.wildcards = true
		cd.Unlock()
	}

	// if the first non-component arg is "all" and the wildcard type is
	// "explicit" then treat it as a wildcard, but only when given "all"
	// as the first argument
	if len(args) > 0 {
		// check first non-component arg
		if an[CmdNoneMeansAll] == "explicit" || an[CmdNoneMeansAll] == "none-or-all" {
			if len(args) > 0 && args[0] == "all" {
				// an[CmdNoneMeansAll] = "true"
				cd.Lock()
				cd.wildcards = true
				cd.Unlock()
				args = args[1:]
			}
		}
	}

	log.Debug().Msgf("rawargs: %s", args)

	// filter in place - pull out all args that are not valid instance
	// names into params after rebuild this should only apply to
	// 'import'
	n := 0
	for _, a := range args {
		if !instance.ValidName(a) {
			params = append(params, a)
		} else {
			args[n] = a
			n++
		}
	}
	args = args[:n]

	log.Debug().Msgf("args %v, params %v, ct %q", args, params, ct)

	cd.Lock()
	cd.params = params
	cd.Unlock()

	if !cd.wildcards {
		if len(args) == 0 {
			return nil
		}

		// check the first arg for a component, or fall back to command
		// annotation, if set.
		if ct = geneos.ParseComponent(args[0]); ct == nil {
			cd.Lock()
			cd.names = args
			cd.Unlock()
			return
		}

		cd.Lock()
		cd.ct = geneos.ParseComponent(args[0])
		cd.Unlock()
		names = args[1:]

		if an[CmdGlobNames] == "true" {
			log.Debug().Msgf("matching %v", names)
			if len(names) > 0 {
				newnames := instance.Match(h, ct, names...)
				if len(newnames) == 0 {
					cd.Lock()
					cd.names = names
					cd.Unlock()
					return fmt.Errorf("%v - %w", names, geneos.ErrNotExist)
				} else {
					names = newnames
				}
			}
		}
	} else {
		defaultComponent := ""
		if len(args) > 0 {
			defaultComponent = args[0]
		}

		if ct = geneos.ParseComponent(defaultComponent); ct == nil {
			// first arg is not a known type, so treat the rest as instance names
			names = args
			if an[CmdGlobNames] == "true" {
				log.Debug().Msgf("matching %v", names)
				if len(names) == 1 && names[0] == "all" && (an[CmdNoneMeansAll] == "true" || an[CmdNoneMeansAll] == "explicit") {
					names = instance.InstanceNames(h, ct)
				} else if len(names) > 0 {
					newnames := instance.Match(h, ct, names...)
					log.Debug().Msgf("match returned %v", newnames)
					if len(newnames) == 0 {
						log.Debug().Msgf("no names match %v", names)
						cd.Lock()
						cd.names = names
						cd.Unlock()
						return fmt.Errorf("%v - %w", names, geneos.ErrNotExist)
					} else {
						names = newnames
					}
				}
			}
		} else {
			cd.Lock()
			cd.ct = geneos.ParseComponent(args[0])
			cd.Unlock()

			names = args[1:]
			if an[CmdGlobNames] == "true" {
				log.Debug().Msgf("matching %v", names)
				if len(names) > 0 {
					newnames := instance.Match(h, ct, names...)
					if len(newnames) == 0 {
						cd.Lock()
						cd.names = names
						cd.Unlock()
						return fmt.Errorf("%v - %w", names, geneos.ErrNotExist)
					} else {
						names = newnames
					}
				}
			}
		}

		log.Debug().Msgf("args = %+v", names)
		if len(names) == 0 || (len(names) == 1 && names[0] == "all") {
			// no args also means all instances
			wild = true
			names = instance.InstanceNames(h, ct)
		} else {
			// expand each arg and save results to a new slice
			// if local == "", then all instances on host (e.g. @host)
			// if host == "all" (or none given), then check instance on all hosts
			// @all is not valid - should be no arg
			var nargs []string
			for _, arg := range names {
				// check if not valid first and leave unchanged, skip
				if !(strings.HasPrefix(arg, "@") || instance.ValidName(arg)) {
					log.Debug().Msgf("leaving unchanged: %s", arg)
					nargs = append(nargs, arg)
					continue
				}
				_, local, r := instance.SplitName(arg, h)
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
						rargs := instance.InstanceNames(r, ct)
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
							if i, err := instance.Get(ct, name); err == nil && !i.Loaded().IsZero() {
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
			names = nargs
		}
	}

	log.Debug().Msgf("ct %s, args %v, params %v", ct, names, params)

	m := make(map[string]bool, len(names))
	var newnames []string

	// traditional loop because we can't modify args in a loop to skip
	for i := 0; i < len(names); i++ {
		name := names[i]
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
	names = newnames

	cd.Lock()
	cd.names = names
	cd.params = params
	cd.Unlock()

	if !cd.wildcards {
		return
	}

	// if args is empty, find them all again. ct == None too?
	if len(names) == 0 && (geneos.LocalRoot() != "" || len(geneos.RemoteHosts(false)) > 0) && !wild {
		names = instance.InstanceNames(h, ct)
		cd.Lock()
		cd.names = names
		cd.Unlock()
	}

	log.Debug().Msgf("ct %s, args %v, params %v", ct, names, params)
	return
}

// ParseTypeNames parses the ct, args and params set by ParseArgs in a
// Pre run and returns the ct and a slice of names. Parameters are
// ignored.
func ParseTypeNames(command *cobra.Command) (ct *geneos.Component, args []string) {
	d := cmddata(command)
	if d == nil {
		return
	}
	ct = d.ct
	args = d.names
	return
}

// ParseTypeNamesParams parses the ct, args and params set by ParseArgs
// in a Pre run and returns the ct and a slice of names and a slice of
// params.
func ParseTypeNamesParams(command *cobra.Command) (ct *geneos.Component, args, params []string) {
	ct, args = ParseTypeNames(command)
	d := cmddata(command)
	if d == nil {
		return
	}
	params = d.params
	return
}
