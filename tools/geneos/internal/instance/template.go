/*
Copyright © 2022 ITRS Group

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

package instance

import (
	"io"
	"io/fs"
	"path"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// return the KEY from "[TYPE:]KEY=VALUE"
func nameOf(s string, sep string) string {
	r := strings.SplitN(s, sep, 2)
	return r[0]
}

// return the VALUE from "[TYPE:]KEY=VALUE"
func valueOf(s string, sep string) string {
	r := strings.SplitN(s, sep, 2)
	if len(r) > 0 {
		return r[1]
	}
	return ""
}

// first returns the first non-empty string argument
func first(d ...interface{}) string {
	for _, f := range d {
		if s, ok := f.(string); ok {
			if s != "" {
				return s
			}
		}
	}
	return ""
}

var fnmap = template.FuncMap{
	"first":   first,
	"join":    path.Join,
	"nameOf":  nameOf,
	"valueOf": valueOf,
}

// ExecuteTemplate loads templates from TYPE/templates/[tmpl]* and parse them,
// using the instance data write it out to a single file. If tmpl is
// empty, load all files. An optional perms parameter can be passed to
// set the file permissions, defaulting to 0660.
func ExecuteTemplate(i geneos.Instance, p string, name string, defaultTemplate []byte, perms ...fs.FileMode) (err error) {
	var out io.WriteCloser
	// var t *template.Template

	perm := fs.FileMode(0660)
	if len(perms) > 0 {
		perm = perms[0]
	}

	cf := i.Config()

	t := template.New("").Funcs(fnmap).Option("missingkey=zero")
	if t, err = t.ParseGlob(i.Host().PathTo(i.Type(), "templates", "*.gotmpl")); err != nil {
		t = template.New(name).Funcs(fnmap).Option("missingkey=zero")
		// if there are no templates, use internal as a fallback
		log.Warn().Msgf("No templates found in %s, using internal defaults", i.Host().PathTo(i.Type(), "templates"))
		t = template.Must(t.Parse(string(defaultTemplate)))
	}

	if out, err = i.Host().Create(p, perm); err != nil {
		log.Warn().Msgf("Cannot create configuration file for %s %s", i, p)
		return err
	}
	defer out.Close()
	m := cf.ExpandAllSettings(config.NoDecode(true))
	// viper insists this is a float64, manually override
	m["port"] = uint16(cf.GetUint("port"))
	// set high level defaults
	m["root"] = i.Host().GetString("geneos")
	m["name"] = i.Name()
	m["home"] = i.Home()
	// remove aliases and expand the rest
	for _, k := range cf.AllKeys() {
		if _, ok := i.Type().LegacyParameters[k]; ok {
			delete(m, k)
		}
	}
	// log.Debug().Msgf("template data: %#v", m)

	if err = t.ExecuteTemplate(out, name, m); err != nil {
		log.Error().Err(err).Msg("Cannot create configuration from template(s)")
	}

	return
}
