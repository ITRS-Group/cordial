/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"fmt"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
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
func SetBasicAuth(username string, password *config.Plaintext) Options {
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
