package cmd

import (
	"bytes"
	"html/template"
	"strings"
	"time"

	"github.com/aymerick/douceur/inliner"

	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
)

func createHTML(cf *config.Config, data DV2EMailData, htmlTemplate string, inlineCSS bool) (html string, err error) {
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

func buildHTMLFiles(cf *config.Config, data DV2EMailData, timestamp time.Time, inlineCSS bool) (files []dataFile, err error) {
	lookupDateTime := map[string]string{
		"date":     timestamp.Local().Format("20060102"),
		"time":     timestamp.Local().Format("150405"),
		"datetime": timestamp.Local().Format(time.RFC3339),
	}
	switch cf.GetString("html.split") {
	case "entity":
		entities := map[string][]*commands.Dataview{}
		for _, d := range data.Dataviews {
			if len(entities[d.XPath.Entity.Name]) == 0 {
				entities[d.XPath.Entity.Name] = []*commands.Dataview{}
			}
			entities[d.XPath.Entity.Name] = append(entities[d.XPath.Entity.Name], d)
		}
		for entity, e := range entities {
			many := DV2EMailData{
				Dataviews: e,
				Env:       data.Env,
			}
			htmlString, err := createHTML(cf, many, cf.GetString("html.template"), inlineCSS)
			if err != nil {
				return files, err
			}
			lookup := map[string]string{
				"default":   "dataviews",
				"entity":    entity,
				"sampler":   "",
				"dataview":  "",
				"timestamp": timestamp.Local().Format("20060102150405"),
			}
			filename := buildName(cf.GetString("html.filename", config.LookupTable(lookupDateTime)), lookup) + ".html"
			files = append(files, dataFile{
				name:    filename,
				content: bytes.NewBufferString(htmlString),
			})
			// m.AttachReader(filename, bytes.NewBufferString(htmlString))
		}
	case "dataview":
		for _, d := range data.Dataviews {
			one := DV2EMailData{
				Dataviews: []*commands.Dataview{d},
				Env:       data.Env,
			}
			htmlString, err := createHTML(cf, one, cf.GetString("html.template"), inlineCSS)
			if err != nil {
				return files, err
			}
			lookup := map[string]string{
				"default":   "dataviews",
				"entity":    d.XPath.Entity.Name,
				"sampler":   d.XPath.Sampler.Name,
				"dataview":  d.XPath.Dataview.Name,
				"timestamp": timestamp.Local().Format("20060102150405"),
			}
			filename := buildName(cf.GetString("html.filename", config.LookupTable(lookupDateTime)), lookup) + ".html"
			files = append(files, dataFile{
				name:    filename,
				content: bytes.NewBufferString(htmlString),
			})
			// m.AttachReader(filename, bytes.NewBufferString(htmlString))
		}
	default:
		htmlString, err := createHTML(cf, data, cf.GetString("html.template"), inlineCSS)
		if err != nil {
			return files, err
		}
		lookup := map[string]string{
			"default":   "dataviews",
			"entity":    "",
			"sampler":   "",
			"dataview":  "",
			"timestamp": timestamp.Local().Format("20060102150405"),
		}
		filename := buildName(cf.GetString("html.filename", config.LookupTable(lookupDateTime)), lookup) + ".html"
		files = append(files, dataFile{
			name:    filename,
			content: bytes.NewBufferString(htmlString),
		})
		// m.AttachReader(filename, bytes.NewBufferString(htmlString))
	}

	return
}
