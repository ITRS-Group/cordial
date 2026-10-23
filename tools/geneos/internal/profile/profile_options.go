package profile

import "github.com/itrs-group/cordial/tools/geneos/internal/values"

type profileOptions struct {
	clearKeys []string
	common    Common
	vals      values.Values
	rebuild   bool
}

type ProfileOption func(*profileOptions)

func evaluateOptions(opts ...ProfileOption) *profileOptions {
	pOpts := &profileOptions{}
	for _, opt := range opts {
		opt(pOpts)
	}
	return pOpts
}

func ClearKeys(keys []string) ProfileOption {
	return func(opts *profileOptions) {
		opts.clearKeys = keys
	}
}

func WithCommon(common Common) ProfileOption {
	return func(opts *profileOptions) {
		opts.common = common
	}
}

func WithValues(vals values.Values) ProfileOption {
	return func(opts *profileOptions) {
		opts.vals = vals
	}
}

func Rebuild(rebuild bool) ProfileOption {
	return func(opts *profileOptions) {
		opts.rebuild = rebuild
	}
}
