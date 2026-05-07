package instance

type CleanOption func(*cleanOptions)

type cleanOptions struct {
	force bool
	full  bool
}

func evalCleanOptions(opts ...CleanOption) (options cleanOptions) {
	for _, o := range opts {
		if o != nil {
			o(&options)
		}
	}
	return
}

func FullClean(full bool) CleanOption {
	return func(o *cleanOptions) {
		o.full = full
	}
}

func ForceClean(force bool) CleanOption {
	return func(o *cleanOptions) {
		o.force = force
	}
}
