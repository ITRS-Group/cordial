package san

import (
	"fmt"
	"path"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

type SanProfile struct {
	profile.Common  `mapstructure:",squash"`
	Gateways        []string          `yaml:"gateways,omitempty"`
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

func applyProfile(ct *geneos.Component, pf *config.Config, name, key string) (err error) {
	var sans []SanProfile

	err = pf.UnmarshalKey(pf.Join("profiles", name, key), &sans, config.NoExpand())
	if err != nil {
		return fmt.Errorf("failed to unmarshal SANs: %w", err)
	}

	// global lookup table, add values here as needed
	lookup := map[string]string{
		"hostname": geneos.LOCAL.String(),
	}

	// build param key delete list
	deleteKeys := []string{
		values.GATEWAYS,
		values.ATTRIBUTES,
		values.TYPES,
		values.VARIABLES,
		values.ENVIRONMENT,
		values.MANAGED_ENTITIES,
	}

	for _, san := range sans {
		// does it already exist?
		instanceName := config.Expand[string](pf, san.Name, config.LookupTable(lookup))

		vals := profile.ApplyCommonParams(pf, name, &san.Common, lookup)

		// expand all elements of gateways, join with commas and split again to catch already expanded values
		for gw := range strings.SplitSeq(strings.Join(pf.ExpandStringSlice(san.Gateways), ","), ",") {
			// san.Gateways expands to a map of host:port
			vals.Gateways = append(vals.Gateways, gw)
		}
		vals.Types = pf.ExpandStringSlice(san.Types)
		vals.Attributes = pf.ExpandStringSlice(san.Attributes)

		// vals.Variables = san.Variables
		for _, v := range san.Variables {
			vals.Variables = append(vals.Variables, values.Variable{
				Type:  v.Type,
				Name:  v.Name,
				Value: config.Expand[string](pf, v.Value),
			})
		}

		// now add managed-entities, prefixing each with the "name/"
		for _, me := range san.ManagedEntities {
			// start with clean values for each managed entity
			if extraVals, err := processSANManagedEntity(pf, me); err == nil {
				// merge into top-level values
				vals.Attributes = append(vals.Attributes, extraVals.Attributes...)
				vals.Types = append(vals.Types, extraVals.Types...)
				vals.Variables = append(vals.Variables, extraVals.Variables...)
			}
		}

		if err := profile.ApplyInstance(pf, ct, san.Common, instanceName, deleteKeys, vals); err != nil {
			return err
		}
	}
	return
}

func processSANManagedEntity(pf *config.Config, entity ManagedEntity) (vals values.Values, err error) {
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
			return vals, geneos.ErrNotExist
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
			cf := i.Config()

			// set the default options for the instance configuration,
			// which is used as a source of data for the managed entity
			// loop below
			lookup := map[string]string{
				"hostname": geneos.LOCAL.String(),
			}
			cf.SetDefaultExpandOptions(
				config.LookupTable(lookup),
				config.Prefix("select", profile.SelectPrefix),
				config.Prefix("replace", profile.ReplacePrefix),
				config.Prefix("flag", profile.Flag, config.Get[map[string]any](pf, "flags")),
				config.NoDecode(true),
			)

			// build entity using lookup table for expansion, with custom functions
			variables := make(values.Variables, len(entity.Variables))
			for idx, v := range entity.Variables {
				variables[idx] = values.Variable{
					Type:  v.Type,
					Name:  v.Name,
					Value: config.Expand[string](cf, v.Value),
				}
			}

			e := ManagedEntity{
				Name:       config.Expand[string](cf, entity.Name),
				Types:      cf.ExpandStringSlice(entity.Types),
				Attributes: cf.ExpandStringSlice(entity.Attributes),
				Variables:  variables,
			}
			entities[idx] = e
		}
	}

	// append managed entity information to the view, prefixed with the entity name
	for _, entity := range entities {
		for i := range entity.Types {
			vals.Types = append(vals.Types, entity.Name+"/"+entity.Types[i])
		}
		for i := range entity.Attributes {
			vals.Attributes = append(vals.Attributes, entity.Name+"/"+entity.Attributes[i])
		}
		for i := range entity.Variables {
			vals.Variables = append(vals.Variables, values.Variable{
				Type:  entity.Variables[i].Type,
				Name:  entity.Name + "/" + entity.Variables[i].Name,
				Value: entity.Variables[i].Value,
			})
		}
	}

	return vals, nil
}
