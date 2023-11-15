package cmd

import (
	"html/template"
	"strings"

	"github.com/aymerick/douceur/inliner"

	"github.com/itrs-group/cordial/pkg/config"
)

func createHTML(cf *config.Config, data dv2emailData, htmlTemplate string, inlineCSS bool) (html string, err error) {
	ht, err := template.New("dataview").Parse(htmlTemplate)
	if err != nil {
		return
	}

	var body strings.Builder
	err = ht.Execute(&body, data)
	if err != nil {
		return
	}

	if inlineCSS {
		html, err = inliner.Inline(body.String())
	} else {
		html = body.String()
	}

	return
}
