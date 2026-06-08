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
	"os"
	"path"
	"slices"
	"strings"
	"text/template"

	zlog "github.com/rs/zerolog/log"

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

	zlog.Debug().Msgf("executing template %q for instance %s", name, i)

	cf := i.Config()
	h := i.Host()

	outputPathTmp := outputPath + ".new"

	t := template.New("").Funcs(fnmap).Option("missingkey=zero")
	if t, err = t.ParseGlob(h.PathTo(i.Type(), "templates", "*.gotmpl")); err != nil {
		zlog.Warn().Msgf("Cannot parse template(s) for %s: %v", i, err)
		t = template.New(name).Funcs(fnmap).Option("missingkey=zero")
		// if there are no templates, use internal as a fallback
		zlog.Warn().Msgf("No templates found in %s, using internal defaults", h.PathTo(i.Type(), "templates"))
		t = template.Must(t.Parse(string(defaultTemplate)))
	}

	zlog.Debug().Msgf("creating configuration file %q with permissions %o", outputPathTmp, perms)
	if out, err = h.Create(outputPathTmp, perms); err != nil {
		zlog.Warn().Msgf("Cannot create configuration file for %s %s", i, outputPathTmp)
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
	newVals := []map[string]string{}
	if variables, found := m["variables"]; found {
		switch vx := variables.(type) {
		case []map[string]string:
			newVals = vx
		case []any:
			for _, v := range vx {
				vMap, ok := v.(map[string]any)
				if !ok {
					zlog.Warn().Msgf("variable is not a map, got %T", v)
					return
					// continue
				}
				// the key is a kex string of the name to avoid case-sensitive
				// issues with the name
				nv := map[string]string{
					"type":  vMap["type"].(string),
					"name":  vMap["name"].(string),
					"value": vMap["value"].(string),
				}
				newVals = append(newVals, nv)
			}
		case []values.Variable:
			for _, v := range vx {
				nv := map[string]string{
					"type":  v.Type,
					"name":  v.Name,
					"value": v.Value,
				}
				newVals = append(newVals, nv)
			}
		default:
			zlog.Warn().Msgf("variables is in an unexpected format, got %T", variables)
			// drop through
		}

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
		// but we leave them as strings for toolkits etc to potentially
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
							zlog.Warn().Msgf("unexpected env variable format for %s: %v (type %T)", i, e, e)
						}
					}
				default:
					zlog.Warn().Msgf("unexpected env variable format for %s: %v (type %T)", i, env, env)
				}
				envsStr = slices.DeleteFunc(envsStr, func(e string) bool {
					name, value, found := strings.Cut(e, "=")
					if !found {
						zlog.Warn().Msgf("invalid env variable %q, expected format KEY=VALUE", e)
						return true
					}
					if strings.HasPrefix(value, "${enc:") {
						secret := cf.ExpandToPassword(value)
						defer clear(secret)
						if len(secret) > 0 {
							enc, err := k.Encode(h, secret, false)
							if err != nil {
								zlog.Warn().Msgf("Cannot re-encode environment variable %q for %s: %v", name, i, err)
								// but remove it anyway to avoid leaving secrets in plain text
								return true
							}
							newVals = append(newVals, map[string]string{
								"type":  "stdAESPassword",
								"name":  name,
								"value": "<stdAES>" + enc + "</stdAES>",
							})
						}
						// remove all encoded vars
						return true
					}

					return false
				})
				m["env"] = envsStr
			}

			newVals = slices.DeleteFunc(newVals, func(v map[string]string) bool {
				if v["type"] == "string" && strings.HasPrefix(v["value"], "${enc:") {
					secret := cf.ExpandToPassword(v["value"])
					defer clear(secret)
					if len(secret) == 0 {
						return true
					}

					enc, err := k.Encode(h, secret, false)
					if err != nil {
						zlog.Warn().Msgf("Cannot re-encode variable %q for %s: %v", v["name"], i, err)
						return true
					}
					v["type"] = "stdAESPassword"
					v["value"] = "<stdAES>" + enc + "</stdAES>"
				}

				return false
			})
		}
	}

	m["variables"] = newVals

	zlog.Debug().Msgf("executing template %q to create %q with data %#v", name, outputPathTmp, m)

	if err = t.ExecuteTemplate(out, name, m); err != nil {
		zlog.Error().Err(err).Msg("Cannot create configuration from template(s)")
		// close the file first so Windows systems do not break on Remove
		out.Close()
		h.Remove(outputPathTmp)
		return
	}

	zlog.Debug().Msgf("renaming %q to %q", outputPathTmp, outputPath)
	// close the file first before renaming, stops Windows systems breaking
	out.Close()
	if err = h.Rename(outputPathTmp, outputPath); err != nil {
		h.Remove(outputPathTmp)
	}
	return
}
