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
	"bytes"
	_ "embed"
	"fmt"
	htemplate "html/template"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/wneessen/go-mail"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/email"
)

//go:embed icon_OurSolutions-geneos-image.png
var defaultLogo []byte

func sendEmail(cf *config.Config, em *config.Config, data any) (err error) {
	m, err := buildMessage(cf, em, data)
	if err != nil {
		return
	}

	d, err := email.Dial(em)
	if err != nil {
		return fmt.Errorf("failed to dial email server: %w", err)
	}

	return d.DialAndSend(m)
}

// embeddedImages returns the map of image names to file paths from the
// `images` configuration section.
//
// Names normally include a file extension, e.g. `logo.png`, which the
// configuration treats as a nested key, so the section has to be
// flattened to get the names back. Names are lower-cased by the
// configuration and so must be referenced in HTML as lower-case
// `cid:` values.
func embeddedImages(cf *config.Config) (images map[string]string) {
	images = map[string]string{}
	var flatten func(prefix string, v any)
	flatten = func(prefix string, v any) {
		switch v := v.(type) {
		case map[string]any:
			for k, v := range v {
				flatten(prefix+"."+k, v)
			}
		case string:
			images[strings.TrimPrefix(prefix, ".")] = v
		}
	}
	flatten("", config.Get[map[string]any](cf, "images"))
	return
}

// buildMessage builds the email message, with all body parts and
// attachments, from the configuration and data
func buildMessage(cf *config.Config, em *config.Config, data any) (m *mail.Msg, err error) {
	run := time.Now()

	m, err = email.UpdateEnvelope(em, false)
	if err != nil {
		return nil, fmt.Errorf("failed to update email envelope: %w", err)
	}
	m.Subject(config.Get[string](em, "_subject"))

	contents := config.Get[[]string](cf, cf.Join("email", "contents"))

	// if not a multipart/alternative then always attach a plain
	// text part as the main body
	tt, err := template.New("dataview").Parse(config.Get[string](cf, "text.template"))
	if err != nil {
		return nil, err
	}
	if err = m.SetBodyTextTemplate(tt, data); err != nil {
		return nil, fmt.Errorf("failed to build text body: %w", err)
	}

	if slices.Contains(contents, "text+html") {
		ht, err := htemplate.New("dataview").Parse(config.Get[string](cf, "html.template"))
		if err != nil {
			return nil, err
		}
		if err = m.AddAlternativeHTMLTemplate(ht, data); err != nil {
			return nil, fmt.Errorf("failed to build HTML body: %w", err)
		}
	}

	if slices.Contains(contents, "texttable") {
		files, err := buildTextTableFiles(cf, data, run)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			m.AttachReadSeeker(file.name, file.content)
		}
	}

	if slices.Contains(contents, "html") {
		if err := buildHTMLAttachments(cf, m, data, run); err != nil {
			return nil, err
		}
	}

	if slices.Contains(contents, "xlsx") {
		files, err := buildXLSXFiles(cf, data, run)
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			m.AttachReadSeeker(file.name, file.content)
		}
	}

	if slices.Contains(contents, "images") {
		images := embeddedImages(cf)
		if _, ok := images["logo.png"]; !ok {
			m.EmbedReadSeeker("logo.png", bytes.NewReader(defaultLogo))
		}
		for _, name := range slices.Sorted(maps.Keys(images)) {
			path := images[name]
			if _, err := os.Stat(path); err != nil {
				log.Error("skipping image", slog.Any("error", err), slog.String("path", path))
				continue
			}
			// the Content-ID, used for `cid:` references, is taken from the file name
			m.EmbedFile(path, mail.WithFileName(name))
		}
	}

	return m, nil
}
