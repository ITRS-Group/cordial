package process

type processOptions struct {
	checkFunc    func(checkArg any, cmdline []string) bool
	checkArg     any
	refreshCache bool
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
