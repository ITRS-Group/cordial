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
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// Command annotation types for command behaviour
//
// Annotations must be read-only.
const (
	CmdWildcardNames = "wildcard"     // "true" or "false" - pass all names through a path.Match style lookup
	CmdKeepHosts     = "hosts"        // do not expand "@host", for command like copy/move
	CmdReplacedBy    = "replacedby"   // deprecated command alias
	CmdRequireHome   = "needshomedir" // "true" or "false"
	CmdGlobal        = "global"       // "true" if an empty list of instances should mean all instances.
)

// validNameRE is the test for what is a potentially valid instance name
// versus a parameter. spaces are valid - dumb, but valid - for now. If
// the name starts with number then the next character cannot be a
// number or '.' to help distinguish from versions.
//
// in addition to static names we also allow glob-style characters
// through
//
// look for "[flavour:]name[@host]" - only name can contain glob chars
var validNameRE = regexp.MustCompile(`^(\w+:)?([\w\.\-\ _\*\?\[\]\^\]]+)?(@[\w\-_\.]*)?$`)

// REFRESH:
//
// command have the format:
//
// geneos CMD [FLAGS] [component] [instances] [flags] [params]
//
// component, if given, is always first
// instances can be empty to match everything for most commands, glob-wildcards for matching etc.
// flags are processed before parsing other args
// parameters are everything after the first non instance name?
//

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
//

func ParseArgs(c *cobra.Command, args []string) (err error) {
	var ct *geneos.Component
	// default host from `-H` flag (default `localhost`)
	h := geneos.GetHost(Hostname)

	cmdGlobal := c.Annotations[CmdGlobal]
	cmdWildcardNames := c.Annotations[CmdWildcardNames]
	cmdKeepHosts := c.Annotations[CmdKeepHosts]

	cd := cmddata(c)
	if cd == nil {
		return fmt.Errorf("command context not found")
	}

	if cmdGlobal == "true" {
		cd.Lock()
		cd.globals = true
		cd.Unlock()
	}

	log.Debug().Msgf("args (%d): %v", len(args), args)

	// first, if there is at least one arg then try to consume the first
	// as a component type, then drop through
	if len(args) > 0 {
		if ct = geneos.ParseComponent(args[0]); ct != nil {
			log.Debug().Msgf("first arg is ct %q", ct)
			cd.Lock()
			cd.component = ct
			cd.Unlock()
			args = args[1:]
		}
	}

	// if there are no arguments check for "no args means all instances"
	// annotation and return
	if len(args) == 0 {
		if cmdGlobal == "true" {
			// return everything that matches, at this point any
			// hostname h and component ct are set
			names := instance.InstanceNames(h, ct)
			if len(names) > 0 {
				cd.Lock()
				cd.names = names
				cd.Unlock()
			}
		}
		// always return, with or without instance names set
		return
	}

	// now range over args, and as soon as any is not a valid name (inv
	// glob patterns) put then rest into params
	var names, params []string
	for i, a := range args {
		if cmdKeepHosts == "true" && strings.HasPrefix(a, "@") {
			names = append(names, a)
			continue
		}
		if !validNameRE.MatchString(a) {
			log.Debug().Msgf("not a valid instance name, moving %q to parameters", a)
			params = args[i:]
			break
		}
		names = append(names, a)
	}

	// names is now a list of instance names or patterns, process and remove dups
	if cmdWildcardNames == "true" {
		names = instance.Match(h, ct, cmdKeepHosts == "true", names...)
	}

	cd.Lock()
	cd.names = names
	cd.params = params
	cd.Unlock()

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
	ct = d.component
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
