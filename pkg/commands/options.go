package commands

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
