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
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// Command annotation types for command behaviour
//
// Annotations must be read-only.
const (
	// CmdWildcardNames should be "true" or "false". True will pass all
	// names through a path.Match style lookup
	CmdWildcardNames = geneos.CmdWildcardNames

	// CmdNonInstanceArgsError should be "true" to cause a failure if
	// any args do not match any instances. This should prevent
	// misspelled instance names from dropping through as parameters.
	CmdNonInstanceArgsError = geneos.CmdNonInstanceArgsError

	// CmdAllInstancesMustMatch should be "true" to cause a failure if
	// any instance name patterns do not match any instances. This
	// should prevent misspelled instance names from being ignored.
	CmdAllInstancesMustMatch = geneos.CmdAllInstancesMustMatch

	// CmdKeepHosts should be "true" to not expand "@host", for command
	// like copy/move
	CmdKeepHosts = geneos.CmdKeepHosts

	// CmdReplacedBy should be set to the new command that replaces
	// this one. It should be a full path without the executable, e.g.
	// "package install"
	CmdReplacedBy = geneos.CmdReplacedBy

	// CmdRequireHome shouw be "true" if the command requires the Geneos
	// home directory to be set, initialised or not
	CmdRequireHome = geneos.CmdRequireHome

	// CmdGlobal should be "true" if an empty list of instances should
	// mean all instances.
	CmdGlobal = geneos.CmdGlobal

	// CmdAllowRoot should be "true" to allow running as root, otherwise
	// the command will fail if the effective user ID is 0. This can be
	// overridden by the `--allow-root` flag, which will set this
	// annotation to "true" for the duration of the command, or the
	// global configuration option `allow-root` which will set this
	// annotation to "true" for all commands.
	CmdAllowRoot = geneos.CmdAllowRoot

	// CmdAuditCommand and CmdAuditActions are used for auditing
	// purposes. Their values are set in global config vars for the
	// duration of the command. If these global configuration values
	// already exist then they are not changed.
	//
	// see:
	//     config.Get(config.Global(), config.Join("audit", "commands"))
	//     config.Get(config.Global(), config.Join("audit", "actions"))

	// CmdAuditCommand is used to enable auditing the primary command.
	// To audit the resulting actions use CmdAuditActions. The default
	// value comes from the global configuration. Valid values are
	// "always" or "never", to override the global configuration, or
	// "true" or "false" to set a default if there is no global
	// configuration.
	CmdAuditCommand = geneos.CmdAuditCommand

	// CmdAuditActions is used to enable auditing of all the resulting
	// actions. To audit the primary command triggering the actions use
	// CmdAuditCommand. The default value comes from the global
	// configuration. Valid values are "always" or "never", to override
	// the global configuration, or "true" or "false" to set a default
	// if there is no global configuration.
	CmdAuditActions = geneos.CmdAuditActions

	// CmdProfileTrigger is a booleanused to specify if the command
	// should trigger the application of profiles. This can be used to
	// automatically apply profiles when specific conditions are met.
	CmdProfileTrigger = geneos.CmdProfileTrigger
)

// TODO: rewrite this
//
// command have the format:
//
// geneos CMD [FLAGS] [component] [instances] [flags] [params]
//
// component, if given, is always first
// instances can be empty to match everything for most commands, glob-wildcards for matching etc.
// flags are processed before parsing other args
// parameters are all items that are not valid instance names (syntax, not existence)
//

// ParseArgs does the heavy lifting of sorting out non-flag command line
// ares for the various commands. The results are passed in the command
// context value "data" as a CmdValType struct. The main RunE function
// for each command can then call ParseTypeNames or ParseTypeNamesParams
// to get the results.
func ParseArgs(c *cobra.Command, args []string) (err error) {
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

	var ct *geneos.Component
	// default host from `-H` flag (default `localhost`)
	h := geneos.GetHost(Hostname)

	cmdGlobal, _ := strconv.ParseBool(c.Annotations[CmdGlobal])

	cmdKeepHosts, _ := strconv.ParseBool(c.Annotations[CmdKeepHosts])
	cmdWildcardNames, _ := strconv.ParseBool(c.Annotations[CmdWildcardNames])
	cmdAllInstancesMustMatch, _ := strconv.ParseBool(c.Annotations[CmdAllInstancesMustMatch])

	// set audit configuration in the global configuration based on
	// command annotations

	if auditCommands, ok := c.Annotations[CmdAuditCommand]; ok {
		cf := config.Global()
		confKey := config.Join("audit", "commands")

		switch auditCommands {
		case "always":
			config.Set(cf, confKey, true)
		case "never":
			config.Set(cf, confKey, false)
		case "true":
			if !cf.IsSet(confKey) {
				config.Set(cf, confKey, true)
			}
		case "false":
			if !cf.IsSet(confKey) {
				config.Set(cf, confKey, false)
			}
		}
	}

	if config.Get[bool](config.Global(), config.Join("audit", "commands")) {
		defer func() {
			var cmdLine strings.Builder
			for _, a := range os.Args {
				if strings.Contains(a, " ") {
					cmdLine.WriteString(` "`)
					cmdLine.WriteString(a)
					cmdLine.WriteString(`"`)
				} else {
					cmdLine.WriteString(" ")
					cmdLine.WriteString(a)
				}
			}
			logger.Audit().Event(c.Name(), slog.String("command", strings.TrimSpace(cmdLine.String())))
		}()
	}

	if auditActions, ok := c.Annotations[CmdAuditActions]; ok {
		cf := config.Global()
		confKey := config.Join("audit", "actions")

		switch auditActions {
		case "always":
			config.Set(cf, confKey, true)
		case "never":
			config.Set(cf, confKey, false)
		case "true":
			if !cf.IsSet(confKey) {
				config.Set(cf, confKey, true)
			}
		case "false":
			if !cf.IsSet(confKey) {
				config.Set(cf, confKey, false)
			}
		}
	}

	log.Debug("command annotations", slog.Bool("cmdGlobal", cmdGlobal), slog.Bool("cmdKeepHosts", cmdKeepHosts), slog.Bool("cmdWildcardNames", cmdWildcardNames), slog.Bool("cmdAllInstancesMustMatch", cmdAllInstancesMustMatch))

	cd := cmddata(c)
	if cd == nil {
		return fmt.Errorf("command context not found")
	}

	if cmdGlobal {
		cd.Lock()
		cd.globals = true
		cd.Unlock()
	}

	log.Debug("args", slog.Int("len", len(args)), slog.Any("args", args))

	// first, if there is at least one arg then try to consume the first
	// as a component type, then drop through
	if len(args) > 0 {
		if ct = geneos.ParseComponent(args[0]); ct != nil {
			cd.Lock()
			cd.component = ct
			cd.Unlock()
			args = args[1:]
		}
	}

	// if there are no arguments check for "no args means all instances"
	// annotation and return
	if len(args) == 0 {
		if cmdGlobal {
			// return everything that matches, at this point any
			// hostname h and component ct are set
			names := instance.AllInstanceNames(h, ct)
			if len(names) > 0 {
				cd.Lock()
				cd.names = names
				cd.Unlock()
			}
		}
		// always return, with or without instance names set
		return
	}

	// now range over args, and as soon as any is not a valid name (this
	// includes unexpanded glob patterns) put them into params
	var names, params []string
	for i, a := range args {
		if cmdKeepHosts && strings.HasPrefix(a, "@") {
			names = append(names, a)
			continue
		}
		// if !validNameRE.MatchString(a) {
		if !instance.ValidName(a) {
			log.Debug("not a valid instance name, moving to parameters", slog.String("name", a))
			params = args[i:]
			break
		}
		names = append(names, a)
	}

	// names is now a list of instance names or patterns, process and remove dups
	if cmdWildcardNames {
		names, err = instance.Match(h, ct, cmdKeepHosts, cmdAllInstancesMustMatch, names...)
	}

	log.Debug("parsed args", slog.Any("ct", ct), slog.Any("args", args), slog.Any("params", params), slog.Any("names", names))

	cd.Lock()
	cd.names = names
	cd.params = params
	cd.Unlock()

	return
}

// FetchArgs parses the ct, args and params set by ParseArgs in a Pre
// run and returns the ct and a slice of names and a slice of params.
//
// If the command annotation CmdNonInstanceArgsError is "true" then an
// error will be returned if any params are found, otherwise they are
// returned as a slice of strings.
func FetchArgs(command *cobra.Command) (ct *geneos.Component, args, params []string, err error) {
	// ct, args = ParseTypeNames(command)
	d := cmddata(command)
	if d == nil {
		return
	}
	ct = d.component
	args = d.names
	params = d.params

	cmdNonInstanceArgsError, _ := strconv.ParseBool(command.Annotations[CmdNonInstanceArgsError])
	if cmdNonInstanceArgsError && len(params) > 0 {
		err = fmt.Errorf("unexpected parameters: %v", params)
	}

	return
}
