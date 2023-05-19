/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package commands

import (
	"fmt"
	"time"
)

// Options is an option type used for commands functions
type Options func(*Connection)

func evalOptions(c *Connection, options ...Options) {
	for _, opt := range options {
		opt(c)
	}
}

// SetBasicAuth configures basic authentication on the connection, given
// a username and password (both as plain strings)
func SetBasicAuth(username, password string) Options {
	return func(c *Connection) {
		if username != "" {
			c.AuthType = Basic
			c.Username = username
			c.Password = password
		}
	}
}

// AllowInsecureCertificates allows unverified connections over TLS to
// the gateway
func AllowInsecureCertificates(opt bool) Options {
	return func(c *Connection) {
		c.InsecureSkipVerify = opt
	}
}

// Ping overrides the built-in ping() function used to test the
// availability of the gateway when used with DialGateways() and
// Redial()
func Ping(ping func(*Connection) error) Options {
	return func(c *Connection) {
		c.ping = &ping
	}
}

// Timeout sets the timeout of the REST connection as a time.Duration
func Timeout(timeout time.Duration) Options {
	return func(c *Connection) {
		c.Timeout = timeout
	}
}

// Args is an option type used to add positional arguments to Geneos
// commands called via the REST API
type Args func(*CommandArgs)

func evalArgOptions(args *CommandArgs, options ...Args) {
	for _, opt := range options {
		opt(args)
	}
}

// Arg is a positional argument passed to a Geneos REST command. Use
// multiple Arg() options to [RunCommand] or [RunCommandAll] to set the
// required values. e.g.
//
// fn(..., Arg(1, "value"), Arg(5, "string"))
func Arg(index int, value string) Args {
	return func(a *CommandArgs) {
		(*a)[fmt.Sprint(index)] = value
	}
}
