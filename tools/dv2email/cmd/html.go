/*
Copyright © 2023 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/aymerick/douceur/inliner"
	"github.com/wneessen/go-mail"

	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
)

func createHTML(_ *config.Config, data any, htmlTemplate string, inlineCSS bool) (html string, err error) {
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

func buildHTMLAttachments(cf *config.Config, m *mail.Msg, d any, timestamp time.Time) (err error) {
	data, ok := d.(DV2EMailData)
	if !ok {
		err = os.ErrInvalid
		return
	}
	lookupDateTime := map[string]string{
		"date":     timestamp.Local().Format("20060102"),
		"time":     timestamp.Local().Format("150405"),
		"datetime": timestamp.Local().Format(time.RFC3339),
	}
	ht, err := template.New("dataview").Parse(cf.GetString("html.template"))
	if err != nil {
		return err
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
			lookup := map[string]string{
				"default":   "dataviews",
				"entity":    entity,
				"sampler":   "",
				"dataview":  "",
				"timestamp": timestamp.Local().Format("20060102150405"),
			}
			filename := buildName(cf.GetString("html.filename", config.LookupTable(lookupDateTime)), lookup) + ".html"
			m.AttachHTMLTemplate(filename, ht, many)
		}
	case "dataview":
		for _, d := range data.Dataviews {
			one := DV2EMailData{
				Dataviews: []*commands.Dataview{d},
				Env:       data.Env,
			}
			lookup := map[string]string{
				"default":   "dataviews",
				"entity":    d.XPath.Entity.Name,
				"sampler":   d.XPath.Sampler.Name,
				"dataview":  d.XPath.Dataview.Name,
				"timestamp": timestamp.Local().Format("20060102150405"),
			}
			filename := buildName(cf.GetString("html.filename", config.LookupTable(lookupDateTime)), lookup) + ".html"
			m.AttachHTMLTemplate(filename, ht, one)
		}
	default:
		lookup := map[string]string{
			"default":   "dataviews",
			"entity":    "",
			"sampler":   "",
			"dataview":  "",
			"timestamp": timestamp.Local().Format("20060102150405"),
		}
		filename := buildName(cf.GetString("html.filename", config.LookupTable(lookupDateTime)), lookup) + ".html"
		m.AttachHTMLTemplate(filename, ht, data)
	}

	return
}
