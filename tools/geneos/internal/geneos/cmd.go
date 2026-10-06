package geneos

// Command annotation types for command behaviour
//
// Annotations must be read-only.
const (
	// CmdWildcardNames should be "true" or "false". True will pass all
	// names through a path.Match style lookup
	CmdWildcardNames = "wildcard"

	// CmdNonInstanceArgsError should be "true" to cause a failure if
	// any args do not match any instances. This should prevent
	// misspelled instance names from dropping through as parameters.
	CmdNonInstanceArgsError = "noninstanceargserror"

	// CmdAllInstancesMustMatch should be "true" to cause a failure if
	// any instance name patterns do not match any instances. This
	// should prevent misspelled instance names from being ignored.
	CmdAllInstancesMustMatch = "allmustmatch"

	// CmdKeepHosts should be "true" to not expand "@host", for command
	// like copy/move
	CmdKeepHosts = "hosts"

	// CmdReplacedBy should be set to the new command that replaces
	// this one. It should be a full path without the executable, e.g.
	// "package install"
	CmdReplacedBy = "replacedby"

	// CmdRequireHome shouw be "true" if the command requires the Geneos
	// home directory to be set, initialised or not
	CmdRequireHome = "needshomedir"

	// CmdGlobal should be "true" if an empty list of instances should
	// mean all instances.
	CmdGlobal = "global"

	// CmdAllowRoot should be "true" to allow running as root, otherwise
	// the command will fail if the effective user ID is 0. This can be
	// overridden by the `--allow-root` flag, which will set this
	// annotation to "true" for the duration of the command, or the
	// global configuration option `allow-root` which will set this
	// annotation to "true" for all commands.
	CmdAllowRoot = "allowroot"

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
	CmdAuditCommand = "auditcommand"

	// CmdAuditActions is used to enable auditing of all the resulting
	// actions. To audit the primary command triggering the actions use
	// CmdAuditCommand. The default value comes from the global
	// configuration. Valid values are "always" or "never", to override
	// the global configuration, or "true" or "false" to set a default
	// if there is no global configuration.
	CmdAuditActions = "auditactions"

	// CmdProfileTrigger is a booleanused to specify if the command
	// should trigger the application of profiles. This can be used to
	// automatically apply profiles when specific conditions are met.
	CmdProfileTrigger = "profiletrigger"
)
