package commands

import "fmt"

type Options func(*Connection)

func evalOptions(c *Connection, options ...Options) {
	for _, opt := range options {
		opt(c)
	}
}

// configure basic authentication on the connection, given a username and password
func SetBasicAuth(username, password string) Options {
	return func(c *Connection) {
		c.AuthType = Basic
		c.Username = username
		c.Password = password
	}
}

// allow unverified connections over TLS to the gateway
func AllowInsecureCertificates(opt bool) Options {
	return func(c *Connection) {
		c.InsecureSkipVerify = opt
	}
}

// override the ping() function used to test the availability of
// the gateway when used with DialGateways() and Redial()
func Ping(ping func(*Connection) error) Options {
	return func(c *Connection) {
		c.ping = &ping
	}
}

type Args map[string]string

type ArgOptions func(*Args)

func evalArgOptions(args *Args, options ...ArgOptions) {
	for _, opt := range options {
		opt(args)
	}
}

func Arg(index int, value string) ArgOptions {
	return func(a *Args) {
		(*a)[fmt.Sprint(index)] = value
	}
}
