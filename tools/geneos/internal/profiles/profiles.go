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

// Package profiles provides functionality to manage and manipulate profiles.

package profiles

import (
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

var log = cordial.Logger

//go:embed "profiles.defaults.yaml"
var profilesDefault []byte

// Initialise sets up the profiles.yaml  file in the user's
// configuration directory, creating it from embedded defaults if it
// does not already exist.
func Initialise(appName string) {
	profilesPath, err := config.UserConfigPath(appName, "profiles.yaml")
	if err != nil {
		log.Error("failed to get user config path", slog.Any("err", err))
		return
	}

	if st, err := os.Stat(profilesPath); errors.Is(err, os.ErrNotExist) {
		// create the file from embedded defaults
		if err := os.WriteFile(profilesPath, profilesDefault, 0644); err != nil {
			log.Error("failed to create profiles file", slog.Any("err", err))
			return
		}
	} else if err != nil {
		log.Error("failed to stat profiles file", slog.Any("err", err))
		return
	} else if st.IsDir() {
		log.Error("profiles path is a directory", slog.String("path", profilesPath))
		return
	}
}

// LoadProfiles initializes the profiles by reading from the specified configuration file.
// If the file does not exist, it creates a new one with default values.
func Load(appName string) (pf *config.Config, err error) {
	pf, err = config.Read("profiles",
		config.AppName(appName),
		config.Format("yaml"),
		config.WithDefaults(profilesDefault, "yaml"),
	)
	if err != nil {
		return
	}

	return
}

type Profile struct {
	Name       string           // name of the profile
	Components []map[string]any // slice of components
}

type Gateway struct {
	Name string `yaml:"name"`
}

type Netprobe struct {
	Name string `yaml:"name"`
}

type Webserver struct {
	Name string `yaml:"name"`
}

type San struct {
	Name            string            `yaml:"name"`
	Gateways        map[string]string `yaml:"gateways,omitempty"`
	Types           []string          `yaml:"types,omitempty"`
	Attributes      values.NameValues `yaml:"attributes,omitempty"`
	Variables       []values.Variable `yaml:"variables,omitempty"`
	ManagedEntities []ManagedEntity   `yaml:"managed-entities,omitempty"`
}

type ManagedEntity struct {
	Name       string            `yaml:"name"`
	ForEach    string            `yaml:"for-each,omitempty"`
	Match      []string          `yaml:"match,omitempty"`  // filter only those instances that match one of the global patterns - Match is applied before Ignore
	Ignore     []string          `yaml:"ignore,omitempty"` // filter out those instances that match one of the global patterns - Ignore is applied after Match
	Types      []string          `yaml:"types,omitempty"`
	Attributes values.NameValues `yaml:"attributes,omitempty"`
	Variables  []values.Variable `yaml:"variables,omitempty"`
}

// Apply applies the specified profile to the local configuration.
func Apply(pf *config.Config, name string) error {
	profile, found := config.Lookup[map[string]any](pf, pf.Join("profiles", name))
	if !found {
		log.Error("profile not found", slog.String("profile", name))
		return fmt.Errorf("profile %q not found", name)
	}

	// each key is a component type, with or with a plural
	var gateways []Gateway
	var netprobes []Netprobe
	var sans []San
	var webservers []Webserver

	for key := range profile {
		switch key {
		case "san", "sans":
			// handle san component
			if err := pf.UnmarshalKey(pf.Join("profiles", name, key), &sans, config.NoExpand()); err != nil {
				log.Error("failed to unmarshal SANs", slog.Any("err", err))
				return fmt.Errorf("failed to unmarshal SANs for profile %q: %w", name, err)
			}
			// do something with the collected SANs
			log.Debug("collected SANs, applying them", slog.String("sans", fmt.Sprintf("%+v", sans)))
			applySans(pf, sans)
		case "gateway", "gateways":
			// handle gateway component
			if err := pf.UnmarshalKey(pf.Join("profiles", name, key), &gateways, config.NoExpand()); err != nil {
				return fmt.Errorf("failed to unmarshal Gateways for profile %q: %w", name, err)
			}
		case "netprobe", "netprobes":
			// handle netprobe component
			if err := pf.UnmarshalKey(pf.Join("profiles", name, key), &netprobes, config.NoExpand()); err != nil {
				return fmt.Errorf("failed to unmarshal Netprobes for profile %q: %w", name, err)
			}
		case "webserver", "webservers":
			// handle webserver component
			if err := pf.UnmarshalKey(pf.Join("profiles", name, key), &webservers, config.NoExpand()); err != nil {
				return fmt.Errorf("failed to unmarshal Webservers for profile %q: %w", name, err)
			}
		default:
			// handle unknown component
		}
	}

	return nil
}

func applySans(pf *config.Config, sans []San) {
	if len(sans) == 0 {
		return
	}

	ct := geneos.ParseComponent("san")

	// global lookup table, add values here as needed
	lookup := map[string]string{
		"hostname": geneos.LOCAL.String(),
	}

	for _, san := range sans {
		// does it already exist?
		name := config.Expand[string](pf, san.Name, config.LookupTable(lookup))
		instances := instance.Instances(geneos.LOCAL, ct, instance.MatchNames(name))

		v := values.Values{
			Gateways:   san.Gateways,
			Types:      san.Types,
			Attributes: san.Attributes,
			Variables:  san.Variables,
		}

		// now add managed-entities, prefixing each with the "name/"
		for _, me := range san.ManagedEntities {
			// start with clean values for each managed entity
			if nv, err := processSANManagedEntity(me,
				values.Values{
					Attributes: me.Attributes,
					Types:      me.Types,
					Variables:  me.Variables,
				},
			); err == nil {
				// merge into top-level values
				v.Attributes = append(v.Attributes, nv.Attributes...)
				v.Types = append(v.Types, nv.Types...)
				v.Variables = append(v.Variables, nv.Variables...)
			}
		}

		if len(instances) == 0 {
			// create here
			instance.Add(geneos.LOCAL, ct, name, 0, v)
			instance.Do(geneos.LOCAL, ct, []string{name}, func(i geneos.Instance, a ...any) (resp *responses.General) {
				resp = responses.New[responses.General](i)
				resp.Err = instance.Start(i)
				return
			}, v).Report(os.Stdout, responses.IgnoreErr(geneos.ErrRunning))
		} else {
			// update the first / only instance
			i := instances[0]
			cf := i.Config()
			keyfile := config.Get[config.KeyFile](cf, "keyfile")
			// TODO: remove existing managed-entity extras before setting new values
			config.Delete(cf, values.MANAGED_ENTITIES) // this may not work, may need to delete all keys

			if ncf, err := values.Set(i, v, keyfile); err == nil {
				i.SetConfig(ncf)
			}
			if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
				i.Log().Error("write failed", slog.Any("error", resp.Err))
				return
			}

			// reload config as instance data is not updated by Add() as an interface value
			i.Unload()
			i.Load()
			i.Rebuild(true)
			i.Reload()
		}
	}
}

func processSANManagedEntity(entity ManagedEntity, v values.Values) (nv values.Values, err error) {
	entities := []ManagedEntity{entity}
	names := entity.Match

	ct := geneos.ParseComponent(entity.ForEach)

	if entity.ForEach != "" {
		// default is match all
		match := []string{"*"}
		if len(entity.Match) > 0 {
			match = entity.Match
		}

		names, err = instance.Match(geneos.LOCAL, ct, false, true, match...)
		if err != nil {
			log.Debug("no matching instances", slog.Any("error", err))
			return v, err
		}

		// now ignore instances that match the ignore patterns
		if len(entity.Ignore) > 0 {
			filtered := make([]string, 0, len(names))
			for _, name := range names {
				ignore := false
				for _, pattern := range entity.Ignore {
					if matched, _ := path.Match(pattern, name); matched {
						ignore = true
						break
					}
				}
				if !ignore {
					filtered = append(filtered, name)
				}
			}
			names = filtered
		}

		// expand the entities slice based on the for-each logic
		instances := instance.Instances(geneos.LOCAL,
			geneos.ParseComponent(entity.ForEach),
			instance.MatchNames(names...))

		entities = make([]ManagedEntity, len(instances))
		for idx, i := range instances {
			// global lookup table
			lookup := map[string]string{
				"hostname": i.Host().String(),
			}

			cf := i.Config()
			// build entity using lookup table for expansion, with custom functions

			variables := make(values.Variables, len(entity.Variables))
			for idx, v := range entity.Variables {
				variables[idx] = values.Variable{
					Type: v.Type,
					Name: v.Name,
					Value: config.Expand[string](cf, v.Value, config.LookupTable(lookup),
						config.Prefix("select", selectPrefix),
						config.Prefix("replace", replacePrefix),
					),
				}
			}

			name := config.Expand[string](cf, entity.Name, config.LookupTable(lookup),
				config.Prefix("select", selectPrefix),
				config.Prefix("replace", replacePrefix),
			)

			e := ManagedEntity{
				Name: name,
				Types: cf.ExpandStringSlice(entity.Types, config.LookupTable(lookup),
					config.Prefix("select", selectPrefix),
					config.Prefix("replace", replacePrefix),
				),
				Attributes: cf.ExpandStringSlice(entity.Attributes, config.LookupTable(lookup),
					config.Prefix("select", selectPrefix),
					config.Prefix("replace", replacePrefix),
				),
				Variables: variables,
			}
			// Variables:  entity.Variables,
			entities[idx] = e
		}
	}

	// append managed entity information to the view, prefixed with the entity name
	for _, entity := range entities {
		for i := range entity.Types {
			v.Types = append(v.Types, entity.Name+"/"+entity.Types[i])
		}
		for i := range entity.Attributes {
			v.Attributes = append(v.Attributes, entity.Name+"/"+entity.Attributes[i])
		}
		for i := range entity.Variables {
			v.Variables = append(v.Variables, values.Variable{
				Type:  entity.Variables[i].Type,
				Name:  entity.Name + "/" + entity.Variables[i].Name,
				Value: entity.Variables[i].Value,
			})
		}
	}

	return v, nil
}

// replacePrefix supports the "replace" prefix. It takes a strings with
// four components of the form: `${replace:param:/PATTERN/TEXT/}` (where
// the `/` can be any character except ':', but is then solely used to
// separate the pattern and the replacement text and must be the last
// character before the closing `}`) and runs regexp.ReplaceAllString().
// If the parameter is empty or not defined, an empty string is
// returned. If parsing the PATTERN fails then no substitution is
// performed.
func replacePrefix(ci map[string]any, s string, trim bool) (result string, err error) {
	s = strings.TrimPrefix(s, "replace:")
	sep := s[len(s)-1:] // last character as separator
	if sep == ":" || sep == "" {
		err = fmt.Errorf("invalid separator")
		return
	}

	params, expr, found := strings.Cut(s, sep)
	if !found || len(params) == 0 || len(expr) == 0 {
		err = fmt.Errorf("invalid args")
		return
	}
	params = strings.TrimSuffix(params, ":")
	expr = sep + expr

	// find first set param

	// create a new config instance and merge the current config map
	// into it, for [config.Expand] usage below
	cf := config.New()
	cf.MergeConfigMap(ci)

	for p := range strings.SplitSeq(params, ":") {
		if val, ok := ci[p]; ok {
			if str, ok := val.(string); ok && str != "" {
				result = config.Expand[string](cf, str)
				break
			}
		}
	}

	p := strings.SplitN(expr[1:], sep, 3)
	if len(p) != 3 || (len(p) == 3 && p[2] != "") {
		// there must be two more separators and nothing after the second
		err = fmt.Errorf("invalid args")
		return
	}
	pattern, text := p[0], p[1]

	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Error("failed to compile regex pattern", slog.Any("error", err), slog.String("pattern", pattern))
		return
	}

	result = re.ReplaceAllString(result, text)
	if trim {
		result = strings.TrimSpace(result)
	}
	return
}

// "select" accepts an expansion (after the enclosing `${}` is removed)
// in the form `select[:param...]:[DEFAULT]` and returns the value of
// the first parameter set or the last field as a static string. If the
// parameter is set but an empty string then it is treated as if it were
// not set. To return a blank string if no parameter is set use
// `${select:param:}` noting the colon just before the closing brace.
func selectPrefix(ci map[string]any, s string, trim bool) (result string, err error) {
	// const validSeparators = "+ /-"
	var r strings.Builder

	s = strings.TrimLeft(s, "select:")
	params := strings.Split(s, ":")
	if len(params) == 0 {
		return
	}
	last := len(params) - 1
	def := params[last]
	params = params[:last]

	// create a new config instance and merge the current config map
	// into it, for [config.Expand] usage below
	cf := config.New()
	cf.MergeConfigMap(ci)

	var p strings.Builder
	for _, param := range params {
		var paramWasSet bool
		p.Reset()

		for i := 0; i < len(param); i++ {
			switch param[i] {
			case '+', ' ', '-', '/':
				if p.Len() > 0 {
					if v, ok := ci[p.String()]; ok {
						if s, ok := v.(string); ok && len(s) > 0 {
							r.WriteString(config.Expand[string](cf, s))
							paramWasSet = true
						}
					}
					p.Reset()
				}
				// only add a '+' if it's doubles up
				if param[i] == '+' {
					if len(param) > i+1 && param[i+1] == '+' {
						r.WriteByte('+')
						i++
					}
				} else {
					// add the separator
					r.WriteByte(param[i])
				}
			default:
				p.WriteByte(param[i])
			}
		}

		if p.Len() > 0 {
			if v, ok := ci[p.String()]; ok {
				if s, ok := v.(string); ok && len(s) > 0 {
					r.WriteString(config.Expand[string](cf, s))
					paramWasSet = true
				}
			}
		}

		if paramWasSet {
			if trim {
				return strings.TrimSpace(r.String()), nil
			}
			return r.String(), nil
		}

		r.Reset()
	}

	return def, nil
}
