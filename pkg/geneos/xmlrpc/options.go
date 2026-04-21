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
	return func(xo *xmlrpcOptions) {
		xo.insecureSkipVerify = true
	}
}

// Secure takes an argument to force checking of the Netprobe
// certificate if the connection is HTTPS. Unlike InsecureSkipVerify,
// this can be used with a boolean variable passed in from the command
// line or in a config file without further tests.
func Secure(secure bool) Options {
	return func(xo *xmlrpcOptions) {
		xo.insecureSkipVerify = !secure
	}
}
