package cmd

import "github.com/spf13/pflag"

// AddOutputFormatFlags registers the standard output-format flags.
func AddOutputFormatFlags(flags *pflag.FlagSet, json, pretty, csv, toolkit *bool) {
	flags.BoolVarP(json, "json", "j", false, "Output JSON")
	flags.BoolVarP(pretty, "pretty", "i", false, "Output indented JSON")
	flags.BoolVarP(csv, "csv", "c", false, "Output CSV")
	flags.BoolVarP(toolkit, "toolkit", "t", false, "Output Toolkit formatted CSV")
}
