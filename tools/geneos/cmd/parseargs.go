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

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// Annotation types for command behaviour
const (
	AnnotationAliasFor   = "aliasfor"     // mapping alias
	AnnotationComponent  = "ct"           // specific component name
	AnnotationNames      = "names"        // json encoded array of instance names
	AnnotationNeedsHome  = "needshomedir" // "true" or "false"
	AnnotationParams     = "params"       // json encoded array of parameters
	AnnotationReplacedBy = "replacedby"   // deprecated command alias
	AnnotationWildcard   = "wildcard"     // "true", "false" or "explicit" (to match "all")
	AnnotationExpand     = "expand"       // "true" or "false" - pass all names through a path.Match style lookup
)

// ParseArgs does the heavy lifting of sorting out non-flag command line
// ares for the various commands.
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
	h := geneos.GetHost(Hostname)

	if command.Annotations == nil {
		command.Annotations = make(map[string]string)
	}
	annotations := command.Annotations
	// args and params are JSON arrays, initialise
	annotations[AnnotationNames] = "[]"
	annotations[AnnotationParams] = "[]"

	if len(args) == 0 && annotations[AnnotationWildcard] != "true" {
		return nil
	}

	// shortcut - if the first non-component arg is "all" and the
	// wildcard type is "explicit" then treat it as a wildcard, but only
	// when given "all" as the first argument
	if len(args) > 0 {
		// first strip component if given and specified
		if annotations[AnnotationComponent] == args[0] {
			args = args[1:]
		}
		// now check first non component arg
		if annotations[AnnotationWildcard] == "explicit" && args[0] == "all" {
			annotations[AnnotationWildcard] = "true"
			args = args[1:]
		}
	}

	log.Debug().Msgf("rawargs: %s", args)

	// filter in place - pull out all args containing '=' into params
	// after rebuild this should only apply to 'import'
	n := 0
	for _, a := range args {
		if strings.Contains(a, "=") {
			params = append(params, a)
		} else {
			args[n] = a
			n++
		}
	}
	args = args[:n]

	log.Debug().Msgf("args %v, params %v, ct %s", args, params, annotations[AnnotationComponent])

	jsonargs, _ := json.Marshal(params)
	annotations[AnnotationParams] = string(jsonargs)

	if annotations[AnnotationWildcard] != "true" {
		if len(args) == 0 {
			return nil
		}
		// check the first non-flag arg for a component, or fall back to
		// command annotation, if set.
		if ct = geneos.ParseComponent(args[0]); ct == nil {
			if annotations[AnnotationComponent] != "" {
				if ct = geneos.ParseComponent(annotations[AnnotationComponent]); ct == nil {
					jsonargs, _ := json.Marshal(args)
					annotations[AnnotationNames] = string(jsonargs)
					return
				}
			} else {
				jsonargs, _ := json.Marshal(args)
				annotations[AnnotationNames] = string(jsonargs)
				return
			}
		}
		if annotations[AnnotationComponent] == "" {
			annotations[AnnotationComponent] = args[0]
			names = args[1:]
		} else {
			names = args
		}
		if annotations[AnnotationExpand] == "true" {
			log.Debug().Msgf("matching %v", names)
			if newnames := instance.Match(h, ct, names...); len(newnames) > 0 {
				names = newnames
			}
		}
	} else {
		defaultComponent := annotations[AnnotationComponent]
		if defaultComponent == "" && len(args) > 0 {
			defaultComponent = args[0]
		}
		if ct = geneos.ParseComponent(defaultComponent); ct == nil {
			// first arg is not a known type, so treat the rest as instance names
			names = args
			if annotations[AnnotationExpand] == "true" {
				log.Debug().Msgf("matching %v", names)
				if newnames := instance.Match(h, ct, names...); len(newnames) > 0 {
					names = newnames
				}

			}
		} else {
			if annotations[AnnotationComponent] == "" {
				annotations[AnnotationComponent] = args[0]
				names = args[1:]
			} else {
				names = args
			}
			if annotations[AnnotationExpand] == "true" {
				log.Debug().Msgf("matching %v", names)
				if newnames := instance.Match(h, ct, names...); len(newnames) > 0 {
					names = newnames
				}
			}
		}

		log.Debug().Msgf("args = %+v", names)
		if len(names) == 0 || (len(names) == 1 && names[0] == "all") {
			// no args also means all instances
			wild = true
			names = instance.Names(h, ct)
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

	jsonargs, _ = json.Marshal(names)
	annotations[AnnotationNames] = string(jsonargs)
	jsonparams, _ := json.Marshal(params)
	annotations[AnnotationParams] = string(jsonparams)

	if annotations[AnnotationWildcard] == "false" {
		return
	}

	// if args is empty, find them all again. ct == None too?
	if len(names) == 0 && (geneos.Root() != "" || len(geneos.RemoteHosts(false)) > 0) && !wild {
		names = instance.Names(h, ct)
		jsonargs, _ := json.Marshal(names)
		annotations[AnnotationNames] = string(jsonargs)
	}

	log.Debug().Msgf("ct %s, args %v, params %v", ct, names, params)
	return
}

// ParseTypeNames parses the ct, args and params set by ParseArgs in a
// Pre run and returns the ct and a slice of names. Parameters are
// ignored.
func ParseTypeNames(command *cobra.Command) (ct *geneos.Component, args []string) {
	ct = geneos.ParseComponent(command.Annotations[AnnotationComponent])
	if err := json.Unmarshal([]byte(command.Annotations[AnnotationNames]), &args); err != nil {
		log.Debug().Err(err).Msg("")
	}
	return
}

// ParseTypeNamesParams parses the ct, args and params set by ParseArgs
// in a Pre run and returns the ct and a slice of names and a slice of
// params.
func ParseTypeNamesParams(command *cobra.Command) (ct *geneos.Component, args, params []string) {
	ct, args = ParseTypeNames(command)
	if err := json.Unmarshal([]byte(command.Annotations[AnnotationParams]), &params); err != nil {
		log.Debug().Err(err).Msg("")
	}
	return
}
