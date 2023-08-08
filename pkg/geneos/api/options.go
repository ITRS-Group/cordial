package api

type apiOptions struct {
	insecureSkipVerify bool
}

type Options func(*apiOptions)

func evalOptions(options ...Options) *apiOptions {
	opts := &apiOptions{}
	for _, opt := range options {
		opt(opts)
	}
	return opts
}

// InsecureSkipVerify
func InsecureSkipVerify() Options {
	return func(c *apiOptions) {
		c.insecureSkipVerify = true
	}
}
