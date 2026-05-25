package values

import "strings"

// gateway - name:port
type Gateways map[string]string

const GatewaysOptionstext = "A gateway connection in the format HOSTNAME:PORT\n(Repeat as required, san and floating only)"

func (i *Gateways) String() string {
	return ""
}

func (i *Gateways) Set(value string) error {
	if *i == nil {
		*i = Gateways{}
	}
	host, port, found := strings.Cut(value, ":")
	if !found {
		port = "7039"
	}
	(*i)[host] = port
	return nil
}

func (i *Gateways) Type() string {
	return "HOSTNAME:PORT"
}
