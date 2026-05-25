package values

// attribute - name=value
type Types []string

const TypesOptionsText = "A type NAME\n(Repeat as required, san only)"

func (i *Types) String() string {
	return ""
}

func (i *Types) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *Types) Type() string {
	return "NAME"
}
