package cmd

import (
	"strings"
	"text/template"

	"github.com/itrs-group/cordial/pkg/config"
)

func createTextTemplate(cf *config.Config, data dv2emailData, textTemplate string) (text string, err error) {
	tt, err := template.New("dataview").Parse(textTemplate)
	if err != nil {
		return
	}

	var body strings.Builder
	err = tt.Execute(&body, data)
	if err != nil {
		return
	}
	text = body.String()
	return
}

func createTextTables(cf *config.Config, data dv2emailData) (text string, err error) {
	return
}
