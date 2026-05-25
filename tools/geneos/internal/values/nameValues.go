package values

// attribute - name=value
type NameValues []string

const AttributesOptionsText = "Attribute in the format NAME=VALUE\n(Repeat as required, san only)"
const EnvsOptionsText = "Environment variable for instance start-up\n(Repeat as required)"
const HeadersOptionsText = "HTTP header in the format NAME=VALUE\n(Repeat as required)"

func (i *NameValues) String() string {
	return ""
}

func (i *NameValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *NameValues) Type() string {
	return "NAME=VALUE"
}
