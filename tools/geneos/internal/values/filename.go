package values

// Filename fulfils the Var interface for pflag
type Filename []string

func (i *Filename) String() string {
	return ""
}

func (i *Filename) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *Filename) Type() string {
	return "[DEST=]PATH|URL"
}
