package xmlrpc

type xmlrpcOptions struct {
	insecureSkipVerify bool
}

type Options func(*xmlrpcOptions)

func evalOptions(c *xmlrpcOptions, options ...Options) {
	for _, opt := range options {
		opt(c)
	}
}

// InsecureSkipVerify
func InsecureSkipVerify() Options {
	return func(c *xmlrpcOptions) {
		c.insecureSkipVerify = true
	}
}
