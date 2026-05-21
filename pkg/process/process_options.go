package process

type processOptions struct {
	checkFunc    func(checkArg any, cmdline []string) bool
	checkArg     any
	refreshCache bool
	fetchLazy    bool
}

type ProcessOption func(*processOptions)

func evalProcessOptions(options ...ProcessOption) *processOptions {
	opts := &processOptions{}
	for _, o := range options {
		o(opts)
	}
	return opts
}

func RefreshCache() ProcessOption {
	return func(po *processOptions) {
		po.refreshCache = true
	}
}

func CustomChecker(checkFunc func(checkArg any, cmdline []string) bool, checkArg any) ProcessOption {
	return func(po *processOptions) {
		po.checkFunc = checkFunc
		po.checkArg = checkArg
	}
}

// FetchLazyFields is an option to indicate that any lazy fields in the
// ProcessInfo should be fetched. This is useful when the caller needs
// to access fields that are expensive to fetch, especially on a remote
// host, such as the open ports.
func FetchLazyFields() ProcessOption {
	return func(po *processOptions) {
		po.fetchLazy = true
	}
}
