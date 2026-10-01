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
	"log/slog"
	"os"
	"path"
	"slices"
	"strings"
	"text/template"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

// return the KEY from "[TYPE:]KEY=VALUE"
func nameOf(s string, sep string) string {
	key, _, _ := strings.Cut(s, sep)
	return key
}

// return the VALUE from "[TYPE:]KEY=VALUE"
func valueOf(s string, sep string) string {
	_, value, _ := strings.Cut(s, sep)
	return value
}

// first returns the first non-empty string argument
func first(d ...any) string {
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

// ExecuteTemplate loads the template name from the component
// `templates` directory on the host for the instance i, parses it and
// executes it, writing the results to outputPath with the given
// permissions. If a template file is not found on the host, the
// defaultTemplate is used instead.
//
// The output file is first written to a temporary file with a ".new"
// suffix, which is then renamed to the final outputPath with the
// permissions perms.
//
// If an error occurs, any temporary file is removed and the error
// returned. The existing outputPath file is not modified until the
// final rename step.
func ExecuteTemplate(i geneos.Instance, outputPath string, name string, defaultTemplate []byte, perms os.FileMode) (err error) {
	var out io.WriteCloser
	// var t *template.Template

	cf := i.Config()
	h := i.Host()

	outputPathTmp := outputPath + ".new"

	t := template.New("").Funcs(fnmap).Option("missingkey=zero")
	if t, err = t.ParseGlob(h.PathTo(i.Type(), "templates", "*.gotmpl")); err != nil {
		i.Log().Warn("Cannot parse template(s)", slog.Any("error", err))
		t = template.New(name).Funcs(fnmap).Option("missingkey=zero")
		// if there are no templates, use internal as a fallback
		i.Log().Warn("No templates found, using internal defaults", slog.String("path", h.PathTo(i.Type(), "templates")))
		t = template.Must(t.Parse(string(defaultTemplate)))
	}

	i.Log().Debug("creating configuration file", slog.String("path", outputPathTmp), slog.Any("permissions", perms))
	if out, err = h.Create(outputPathTmp, perms); err != nil {
		i.Log().Warn("Cannot create configuration file", slog.Any("error", err), slog.String("path", outputPathTmp))
		return err
	}

	m := cf.ExpandAllSettings(config.NoDecode(true))

	m["port"] = config.Get[uint16](cf, "port")

	// set high level defaults
	m["root"] = config.Get[string](h.Config, "geneos")
	m["name"] = i.Name()
	m["home"] = i.Home()

	// remove aliases and expand the rest
	for _, k := range cf.AllKeys() {
		if _, ok := i.Type().LegacyParameters[k]; ok {
			delete(m, k)
		}
	}

	// convert "variables" (if any) from slice of structs to slice of
	// maps for lower case keys in templates
	variables := []map[string]string{}

	nv, _ := values.NormaliseVars(m[values.VARIABLES])

	for _, v := range nv {
		variables = append(variables, map[string]string{
			values.MAPKEY_NAME:      v.Name,
			values.VARMAP_KEY_TYPE:  v.Type,
			values.VARMAP_KEY_VALUE: v.Value,
		})
	}

	// tls migration, for now lift new settings up to old names
	c, c_ok := m[CERTIFICATE].(string)
	p, p_ok := m[PRIVATEKEY].(string)
	if (!c_ok || c == "") && (!p_ok || p == "") {
		if t, ok := m[TLSBASE]; ok {
			if ts, ok := t.(map[string]any); ok {
				if ts[CERTIFICATE] != nil && ts[PRIVATEKEY] != nil {
					m[CERTIFICATE] = ts[CERTIFICATE]
					m[PRIVATEKEY] = ts[PRIVATEKEY]
				}
			}
		}
	}

	if i.Type().IsA("gateway") {
		// finally, for gateways, go through environment variables and
		// variables for encoded values, attempt to decode them using
		// given keyfile and re-encode them using the instance keyfile
		// (if any) and move them into a new list so they can be pulled
		// out into Geneos AES256 encoded variable types and not plain
		// strings.
		//
		// Self-announcing netprobes do not support password variables,
		// so we leave them as strings for toolkits etc to potentially
		// decode
		if k, _, _, err := ReadAESKeyFile(i); err == nil {
			if env, ok := m["env"]; ok {
				var envsStr []string

				switch ev := env.(type) {
				case []string:
					envsStr = ev
				case []any:
					for _, e := range ev {
						if es, ok := e.(string); ok {
							envsStr = append(envsStr, es)
						} else {
							i.Log().Warn("unexpected env variable format", slog.Any("variable", e))
						}
					}
				default:
					i.Log().Warn("unexpected env variable format", slog.Any("env", env))
				}
				envsStr = slices.DeleteFunc(envsStr, func(e string) bool {
					name, value, found := strings.Cut(e, "=")
					if !found {
						i.Log().Warn("invalid env variable, expected format KEY=VALUE", slog.String("variable", e))
						return true
					}
					if strings.HasPrefix(value, "${enc:") {
						secret := cf.ExpandToPassword(value)
						defer clear(secret)
						if len(secret) > 0 {
							enc, err := k.Encode(h, secret, false)
							if err != nil {
								i.Log().Warn("Cannot re-encode environment variable", slog.String("name", name), slog.Any("error", err))
								// but remove it anyway to avoid leaving secrets in plain text
								return true
							}
							variables = append(variables, map[string]string{
								values.MAPKEY_NAME:      name,
								values.VARMAP_KEY_TYPE:  "stdAESPassword",
								values.VARMAP_KEY_VALUE: "<stdAES>" + enc + "</stdAES>",
							})
						}
						// remove all encoded vars
						return true
					}

					return false
				})
				m["env"] = envsStr
			}

			variables = slices.DeleteFunc(variables, func(v map[string]string) bool {
				if v[values.VARMAP_KEY_TYPE] == "string" && strings.HasPrefix(v[values.VARMAP_KEY_VALUE], "${enc:") {
					secret := cf.ExpandToPassword(v[values.VARMAP_KEY_VALUE])
					defer clear(secret)
					if len(secret) == 0 {
						return true
					}

					enc, err := k.Encode(h, secret, false)
					if err != nil {
						i.Log().Warn("Cannot re-encode variable", slog.String("name", v[values.MAPKEY_NAME]), slog.Any("error", err))
						return true
					}
					v[values.VARMAP_KEY_TYPE] = "stdAESPassword"
					v[values.VARMAP_KEY_VALUE] = "<stdAES>" + enc + "</stdAES>"
				}

				return false
			})
		}
	}

	// convert SAN managed-entity variables to move from struct to map
	// with lower case keys so templates can access them consistently
	if i.Type().IsA("san") {
		ents, ok := m[values.MANAGED_ENTITIES] // map[string]map[string]any
		if ok {
			if entities, ok := ents.(map[string]any); ok {
				for key, entitiesMap := range entities {
					// do processing per entity, as required

					entity, ok := entitiesMap.(map[string]any)
					if !ok {
						continue
					}

					for _, k := range []string{values.TYPES, values.ATTRIBUTES, values.VARIABLES} {
						if _, ok := entity[k]; !ok {
							continue
						}

						// delete keys marked as unset
						if !cf.IsSet(cf.Join(values.MANAGED_ENTITIES, key, k)) {
							delete(entity, k)
						}
					}

					variables := []map[string]string{}

					nv, _ := values.NormaliseVars(entity[values.VARIABLES])
					for _, v := range nv {
						variables = append(variables, map[string]string{
							values.MAPKEY_NAME:      v.Name,
							values.VARMAP_KEY_TYPE:  v.Type,
							values.VARMAP_KEY_VALUE: v.Value,
						})
					}

					if len(variables) == 0 {
						delete(entity, values.VARIABLES)
					} else {
						entity[values.VARIABLES] = variables
					}

					entities[key] = entity
				}

				m[values.MANAGED_ENTITIES] = entities
			}
		}
	}

	// overwrite the top-level "variables" key with the processed
	// variables slice
	m[values.VARIABLES] = variables

	if err = t.ExecuteTemplate(out, name, m); err != nil {
		i.Log().Error("Cannot create configuration from template(s)", slog.Any("error", err))
		// close the file first so Windows systems do not break on Remove
		out.Close()
		h.Remove(outputPathTmp)
		return
	}

	i.Log().Debug("renaming template", slog.String("from", outputPathTmp), slog.String("to", outputPath))
	// close the file first before renaming, stops Windows systems breaking
	out.Close()
	if err = h.Rename(outputPathTmp, outputPath); err != nil {
		h.Remove(outputPathTmp)
	}
	return
}
