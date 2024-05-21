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
	htemplate "html/template"
	"os"
	"slices"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wneessen/go-mail"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/email"
)

func sendEmail(cf *config.Config, em *config.Config, data any, inlineCSS bool) (err error) {
	run := time.Now()

	m, err := email.UpdateEnvelope(em, inlineCSS)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	m.Subject(em.GetString("_subject"))

	// attachments here

	// if not a multipart/alternative then always attach a plain
	// text part as the main body
	tt, err := template.New("dataview").Parse(cf.GetString("text.template"))
	if err != nil {
		return
	}
	_ = m.SetBodyTextTemplate(tt, data)

	if slices.Contains(cf.GetStringSlice("email.contents"), "text+html") {
		ht, err := htemplate.New("dataview").Parse(cf.GetString("html.template"))
		if err != nil {
			return err
		}
		_ = m.AddAlternativeHTMLTemplate(ht, data)
	}

	if slices.Contains(cf.GetStringSlice("email.contents"), "texttable") {
		files, err := buildTextTableFiles(cf, data, run)
		if err != nil {
			return err
		}
		for _, file := range files {
			m.AttachReadSeeker(file.name, file.content)
		}
	}

	if slices.Contains(cf.GetStringSlice("email.contents"), "html") {
		if err := buildHTMLAttachments(cf, m, data, run); err != nil {
			return err
		}
	}

	if slices.Contains(cf.GetStringSlice("email.contents"), "xlsx") {
		files, err := buildXLSXFiles(cf, data, run)
		if err != nil {
			return err
		}

		for _, file := range files {
			m.AttachReadSeeker(file.name, file.content)
		}
	}

	if slices.Contains(cf.GetStringSlice("email.contents"), "images") {
		for name, path := range cf.GetStringMapString("images") {
			if _, err := os.Stat(path); err != nil {
				log.Error().Err(err).Msg("skipping")
				continue
			}
			m.EmbedFile(path, mail.WithFileName(name))
			m.SetGenHeader("X-Attachment-Id", name)
		}
	}

	// send
	d, err := email.Dial(em)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return d.DialAndSend(m)
}
