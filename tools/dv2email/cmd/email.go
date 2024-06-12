/*
Copyright © 2023 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
