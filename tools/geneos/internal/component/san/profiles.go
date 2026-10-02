package san

import (
	"log/slog"
	"path"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profiles"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

type SanProfile struct {
	profiles.Common `mapstructure:",squash"`
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

func applyProfile(pf *config.Config, name, key string) (err error) {
	ct := geneos.ParseComponent("san")

	var sans []SanProfile
	if err2 := pf.UnmarshalKey(pf.Join("profiles", name, key), &sans, config.NoExpand()); err2 != nil {
		log.Error("failed to unmarshal SANs", slog.Any("err", err2))
		return err2
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
		name := config.Expand[string](pf, san.Name, config.LookupTable(lookup))

		vals := profiles.ApplyCommonParams(san.Common)
		vals.Gateways = san.Gateways
		vals.Types = san.Types
		vals.Attributes = san.Attributes
		vals.Variables = san.Variables

		// now add managed-entities, prefixing each with the "name/"
		for _, me := range san.ManagedEntities {
			// start with clean values for each managed entity
			if extraVals, err := processSANManagedEntity(me); err == nil {
				// merge into top-level values
				vals.Attributes = append(vals.Attributes, extraVals.Attributes...)
				vals.Types = append(vals.Types, extraVals.Types...)
				vals.Variables = append(vals.Variables, extraVals.Variables...)
			}
		}

		if err := profiles.ApplyInstance(ct, san.Common, name, deleteKeys, vals); err != nil {
			return err
		}
	}
	return
}

func processSANManagedEntity(entity ManagedEntity) (vals values.Values, err error) {
	entities := []ManagedEntity{entity}
	names := entity.Match

	ct := geneos.ParseComponent(entity.ForEach)

	if entity.ForEach != "" {
		log.Debug("processing SAN managed entity", slog.String("component", ct.String()))
		// default is match all
		match := []string{"*"}
		if len(entity.Match) > 0 {
			match = entity.Match
		}

		names, err = instance.Match(geneos.LOCAL, ct, false, true, match...)
		if err != nil {
			log.Debug("no matching instances, skipping")
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
			// TODO: get this passed down to the function - global lookup table
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
					Value: config.Expand[string](cf, v.Value,
						config.LookupTable(lookup),
						config.Prefix("select", profiles.SelectPrefix),
						config.Prefix("replace", profiles.ReplacePrefix),
					),
				}
			}

			name := config.Expand[string](cf, entity.Name,
				config.LookupTable(lookup),
				config.Prefix("select", profiles.SelectPrefix),
				config.Prefix("replace", profiles.ReplacePrefix),
			)

			e := ManagedEntity{
				Name: name,
				Types: cf.ExpandStringSlice(entity.Types,
					config.LookupTable(lookup),
					config.Prefix("select", profiles.SelectPrefix),
					config.Prefix("replace", profiles.ReplacePrefix),
				),
				Attributes: cf.ExpandStringSlice(entity.Attributes,
					config.LookupTable(lookup),
					config.Prefix("select", profiles.SelectPrefix),
					config.Prefix("replace", profiles.ReplacePrefix),
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
