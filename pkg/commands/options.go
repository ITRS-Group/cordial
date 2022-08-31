package commands

import "fmt"

type CommandOptions func(*Connection)

func evalOptions(c *Connection, options ...CommandOptions) {
	for _, opt := range options {
		opt(c)
	}
}

func SetBasicAuth(username, password string) CommandOptions {
	return func(c *Connection) {
		c.AuthType = Basic
		c.Username = username
		c.Password = password
	}
}

func AllowInsecureCertificates(opt bool) CommandOptions {
	return func(c *Connection) {
		c.InsecureSkipVerify = opt
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
