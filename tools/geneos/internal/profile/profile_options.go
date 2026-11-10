package profile

import (
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

type profileDeployOptions struct {
	archive            string
	base               string
	certBundle         string
	certBundlePassword config.Secret
	flags              *pflag.FlagSet
	followLogs         bool
	importFiles        []string
	insecure           bool
	keyfile            string
	keyfileCRC         string
	local              bool
	nexus              bool
	noSave             bool
	override           string
	password           config.Secret
	port               uint16
	snapshot           bool
	startAfterAdd      bool
	template           string
	username           string
	vals               values.Values
	version            string
}

// ProfileDeployOption TODO: Describe its purpose and intended use.
type ProfileDeployOption func(*profileDeployOptions)

func evaluateProfileDeployOptions(options ...ProfileDeployOption) *profileDeployOptions {
	pf := &profileDeployOptions{
		base:    "active_prod",
		version: "latest",
	}
	for _, opt := range options {
		opt(pf)
	}
	if pf.archive != "" && !geneos.IsURL(pf.archive) {
		pf.local = true
	}
	return pf
}

// WithPort TODO: Document its behavior, inputs, and effects.
func WithPort(port uint16) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.port = port
	}
}

// WithValues TODO: Document its behavior, inputs, and effects.
func WithValues(extras values.Values) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.vals = extras
	}
}

// WithBase TODO: Document its behavior, inputs, and effects.
func WithBase(base string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.base = base
	}
}

// WithInsecure TODO: Document its behavior, inputs, and effects.
func WithInsecure(insecure bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.insecure = insecure
	}
}

// WithCertBundle TODO: Document its behavior, inputs, and effects.
func WithCertBundle(certBundle string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.certBundle = certBundle
	}
}

// WithCertBundlePassword TODO: Document its behavior, inputs, and effects.
func WithCertBundlePassword(certBundlePassword config.Secret) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.certBundlePassword = certBundlePassword
	}
}

// WithKeyfile TODO: Document its behavior, inputs, and effects.
func WithKeyfile(keyfile string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.keyfile = keyfile
	}
}

// WithKeyfileCRC TODO: Document its behavior, inputs, and effects.
func WithKeyfileCRC(keyfileCRC string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.keyfileCRC = keyfileCRC
	}
}

// WithImportFiles TODO: Document its behavior, inputs, and effects.
func WithImportFiles(importFiles []string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.importFiles = importFiles
	}
}

// WithStartAfterAdd TODO: Document its behavior, inputs, and effects.
func WithStartAfterAdd(startAfterAdd bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.startAfterAdd = startAfterAdd
	}
}

// WithTemplate TODO: Document its behavior, inputs, and effects.
func WithTemplate(template string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.template = template
	}
}

// WithVersion TODO: Document its behavior, inputs, and effects.
func WithVersion(version string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.version = version
	}
}

// WithUsername TODO: Document its behavior, inputs, and effects.
func WithUsername(username string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.username = username
	}
}

// WithPassword TODO: Document its behavior, inputs, and effects.
func WithPassword(password config.Secret) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.password = password
	}
}

// WithLocal TODO: Document its behavior, inputs, and effects.
func WithLocal(local bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.local = local
	}
}

// WithNexus TODO: Document its behavior, inputs, and effects.
func WithNexus(nexus bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.nexus = nexus
	}
}

// WithSnapshot TODO: Document its behavior, inputs, and effects.
func WithSnapshot(snapshot bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.snapshot = snapshot
	}
}

// WithOverride TODO: Document its behavior, inputs, and effects.
func WithOverride(override string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.override = override
	}
}

// WithArchive TODO: Document its behavior, inputs, and effects.
func WithArchive(archive string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.archive = archive
	}
}

// WithNoSave TODO: Document its behavior, inputs, and effects.
func WithNoSave(noSave bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.noSave = noSave
	}
}

// WithFlags TODO: Document its behavior, inputs, and effects.
func WithFlags(flags *pflag.FlagSet) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.flags = flags
	}
}

// WithFollowLogs TODO: Document its behavior, inputs, and effects.
func WithFollowLogs(followLogs bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.followLogs = followLogs
	}
}
