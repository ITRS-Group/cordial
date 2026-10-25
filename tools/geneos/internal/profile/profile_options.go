package profile

import (
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/config"

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

type ProfileDeployOption func(*profileDeployOptions)

func evaluateProfileDeployOptions(options ...ProfileDeployOption) *profileDeployOptions {
	pf := &profileDeployOptions{
		base:    "active_prod",
		version: "latest",
	}
	for _, opt := range options {
		opt(pf)
	}
	return pf
}

func WithPort(port uint16) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.port = port
	}
}

func WithValues(extras values.Values) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.vals = extras
	}
}

func WithBase(base string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.base = base
	}
}

func WithInsecure(insecure bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.insecure = insecure
	}
}

func WithCertBundle(certBundle string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.certBundle = certBundle
	}
}

func WithCertBundlePassword(certBundlePassword config.Secret) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.certBundlePassword = certBundlePassword
	}
}

func WithKeyfile(keyfile string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.keyfile = keyfile
	}
}

func WithKeyfileCRC(keyfileCRC string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.keyfileCRC = keyfileCRC
	}
}

func WithImportFiles(importFiles []string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.importFiles = importFiles
	}
}

func WithStartAfterAdd(startAfterAdd bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.startAfterAdd = startAfterAdd
	}
}

func WithTemplate(template string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.template = template
	}
}

func WithVersion(version string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.version = version
	}
}

func WithUsername(username string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.username = username
	}
}

func WithPassword(password config.Secret) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.password = password
	}
}

func WithLocal(local bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.local = local
	}
}

func WithNexus(nexus bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.nexus = nexus
	}
}

func WithSnapshot(snapshot bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.snapshot = snapshot
	}
}

func WithOverride(override string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.override = override
	}
}

func WithArchive(archive string) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.archive = archive
	}
}

func WithNoSave(noSave bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.noSave = noSave
	}
}

func WithFlags(flags *pflag.FlagSet) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.flags = flags
	}
}

func WithFollowLogs(followLogs bool) ProfileDeployOption {
	return func(o *profileDeployOptions) {
		o.followLogs = followLogs
	}
}
