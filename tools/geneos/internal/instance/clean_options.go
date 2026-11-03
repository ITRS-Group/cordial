package instance

// CleanOption TODO: Describe its purpose and intended use.
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

// FullClean TODO: Document its behavior, inputs, and effects.
func FullClean(full bool) CleanOption {
	return func(o *cleanOptions) {
		o.full = full
	}
}

// ForceClean TODO: Document its behavior, inputs, and effects.
func ForceClean(force bool) CleanOption {
	return func(o *cleanOptions) {
		o.force = force
	}
}
