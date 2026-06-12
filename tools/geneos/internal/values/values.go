package values

import "github.com/itrs-group/cordial"

// package Values manages types for common repeatable flag types, such
// as gateways, includes, variables and attributes. These are used by
// various commands, such as set and unset, and also by the
// configuration file management code to manage the internal
// representation of these items in the configuration file. The types in
// this package implement the pflag.Value interface, so can be used
// directly as flag types for commands that need them.

// Values defined the set of configuration options that can be accepted
// by various commands
type Values struct {
	// Global instance variables

	// Params are key=value pairs set directly in the configuration
	// after checking, keys must be unique but can be hierarchical using
	// the config delimiter. These are valid for all instance types.
	Params []string

	// SecureParams parameters name[=value] where value will be prompted
	// for if not supplied and are encoded with a keyfile. These are
	// only useful on types where keyfiles are supported (by the Geneos
	// software).
	SecureParams SecureValues

	// Environment variables for all instances as name=value pairs
	Envs NameValues

	// SecureEnvs are environment variables in the form name[=value]
	// where value will be prompted for if not supplied and are encoded
	// with a keyfile. These are only useful on types where keyfiles are
	// supported (by the Geneos software).
	SecureEnvs SecureValues

	// Gateway instance variables

	// Includes are include files for Gateway templates, keyed by priority
	Includes Includes

	// SAN and Floating variables

	// Gateways are gateway connections for SAN / floating templates
	Gateways Gateways

	// SAN-only variables

	// Attributes are name=value pairs for attributes for SAN templates
	Attributes NameValues

	// Variables for SAN templates, keyed by variable name
	Variables Variables

	// Types for SAN templates
	Types Types

	// Other repeatable flags variables

	// Headers are name=value pairs for passing to URL requests from
	// various commands, they are not instance specific and are
	// otherwise ignored by the values package. See `geneos.Headers` for
	// more information.
	Headers NameValues
}

var log = cordial.Logger
