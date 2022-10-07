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

type Options func(*Connection)

func evalOptions(c *Connection, options ...Options) {
	for _, opt := range options {
		opt(c)
	}
}

// configure basic authentication on the connection, given a username and password
func SetBasicAuth(username, password string) Options {
	return func(c *Connection) {
		if username != "" {
			c.AuthType = Basic
			c.Username = username
			c.Password = password
		}
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

func Timeout(timeout time.Duration) Options {
	return func(c *Connection) {
		c.Timeout = timeout
	}
}

type Args func(*CommandArgs)

func evalArgOptions(args *CommandArgs, options ...Args) {
	for _, opt := range options {
		opt(args)
	}
}

// Arg is a positional argument passed to a Geneos REST command. Use
// multiple Arg() options to [RunCommand] or [RunCommandAll] to set the
// required values.
func Arg(index int, value string) Args {
	return func(a *CommandArgs) {
		(*a)[fmt.Sprint(index)] = value
	}
}
