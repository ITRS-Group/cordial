package san

import (
	"fmt"
	"log/slog"
	"path"
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

// SanProfile TODO: Describe its purpose and intended use.
type SanProfile struct {
	profile.Common        `mapstructure:",squash"`
	Gateways              []string            `yaml:"gateways,omitempty"`
	Types                 []string            `yaml:"types,omitempty"`
	Attributes            values.NameValues   `yaml:"attributes,omitempty"`
	Variables             []values.Variable   `yaml:"variables,omitempty"`
	ManagedEntitiesCommon ManagedEntityCommon `yaml:"managed-entities-common,omitempty"`
	ManagedEntities       []ManagedEntity     `yaml:"managed-entities,omitempty"`
}

// ManagedEntityCommon contains values shared by all managed entities in a SAN.
type ManagedEntityCommon struct {
	Match      []string          `yaml:"match,omitempty"`
	Ignore     []string          `yaml:"ignore,omitempty"`
	Types      []string          `yaml:"types,omitempty"`
	Attributes values.NameValues `yaml:"attributes,omitempty"`
	Variables  []values.Variable `yaml:"variables,omitempty"`
}

// ManagedEntity TODO: Describe its purpose and intended use.
type ManagedEntity struct {
	Name       string            `yaml:"name"`
	ForEach    string            `yaml:"for-each,omitempty"`
	Match      []string          `yaml:"match,omitempty"`  // filter only those instances that match one of the global patterns - Match is applied before Ignore
	Ignore     []string          `yaml:"ignore,omitempty"` // filter out those instances that match one of the global patterns - Ignore is applied after Match
	Types      []string          `yaml:"types,omitempty"`
	Attributes values.NameValues `yaml:"attributes,omitempty"`
	Variables  []values.Variable `yaml:"variables,omitempty"`
}

func applyProfile(h *geneos.Host, ct *geneos.Component, pf *config.Config, name, key string, forceRebuild bool) (err error) {
	var sans []SanProfile

	err = pf.UnmarshalKey(pf.Join("profiles", name, key), &sans, config.NoExpand())
	if err != nil {
		return fmt.Errorf("failed to unmarshal SANs: %w", err)
	}
	for idx := range sans {
		sans[idx].ManagedEntities = applyManagedEntityCommon(sans[idx].ManagedEntitiesCommon, sans[idx].ManagedEntities)
	}

	lookup := profile.CommonLookups(h)

	// build param key delete list
	clearKeys := []string{
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
		if instanceName == "" {
			return fmt.Errorf("profile %q contains an empty SAN instance name", name)
		}

		vals := profile.ApplyCommonParams(pf, name, &san.Common, lookup)

		// expand all elements of gateways, join with commas and split again to catch already expanded values
		for _, gw := range profile.ExpandCommaList(pf, san.Gateways) {
			// san.Gateways expands to a map of host:port
			vals.Gateways = append(vals.Gateways, gw)
		}

		vals.Types = pf.ExpandStringSlice(san.Types)

		log.Debug("attributes", slog.Any("attrs", san.Attributes))
		for attr := range strings.SplitSeq(config.Expand[string](pf, san.Attributes.String()), ",") {
			log.Debug("expanded attribute", slog.String("attr", attr))
			vals.Attributes = append(vals.Attributes, attr)
		}

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
			if entityVals, err := processSANManagedEntity(pf, me); err == nil {
				vals.Attributes = append(vals.Attributes, entityVals.Attributes...)
				vals.Types = append(vals.Types, entityVals.Types...)
				vals.Variables = append(vals.Variables, entityVals.Variables...)
			}
		}

		if err := profile.ApplyInstance(h, pf, ct, instanceName, san.Common, vals, clearKeys, forceRebuild); err != nil {
			return err
		}
	}
	return
}

func applyManagedEntityCommon(common ManagedEntityCommon, entities []ManagedEntity) []ManagedEntity {
	result := make([]ManagedEntity, 0, len(entities))
	for _, entity := range entities {
		entity.Match = mergeStrings(common.Match, entity.Match)
		entity.Ignore = mergeStrings(common.Ignore, entity.Ignore)
		entity.Types = mergeStrings(common.Types, entity.Types)
		entity.Attributes = mergeNameValues(common.Attributes, entity.Attributes)
		entity.Variables = mergeVariables(common.Variables, entity.Variables)
		result = append(result, entity)
	}
	return result
}

func mergeStrings(common, specific []string) []string {
	result := make([]string, 0, len(common)+len(specific))
	seen := make(map[string]struct{}, len(common)+len(specific))
	for _, group := range [][]string{common, specific} {
		for _, value := range group {
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}

func mergeNameValues(common, specific values.NameValues) values.NameValues {
	result := make(values.NameValues, 0, len(common)+len(specific))
	for _, group := range []values.NameValues{common, specific} {
		for _, value := range group {
			key, _, found := strings.Cut(value, "=")
			if !found {
				key = value
			}
			idx := slices.IndexFunc(result, func(existing string) bool {
				existingKey, _, found := strings.Cut(existing, "=")
				if !found {
					existingKey = existing
				}
				return existingKey == key
			})
			if idx >= 0 {
				result[idx] = value
			} else {
				result = append(result, value)
			}
		}
	}
	return result
}

func mergeVariables(common, specific []values.Variable) []values.Variable {
	result := make([]values.Variable, 0, len(common)+len(specific))
	for _, group := range [][]values.Variable{common, specific} {
		for _, variable := range group {
			idx := slices.IndexFunc(result, func(existing values.Variable) bool {
				return existing.Name == variable.Name
			})
			if idx >= 0 {
				result[idx] = variable
			} else {
				result = append(result, variable)
			}
		}
	}
	return result
}

func processSANManagedEntity(pf *config.Config, entity ManagedEntity) (vals values.Values, err error) {
	entities := []ManagedEntity{entity}
	// names := entity.Match

	ct := geneos.ParseComponent(entity.ForEach)

	if entity.ForEach != "" {
		// default is match all
		match := []string{"*"}
		if len(entity.Match) > 0 {
			match = entity.Match
		}

		names, err := instance.Match(geneos.LOCAL, ct, false, true, match...)
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

			// set default expand options for the instance configuration
			// based on the profile
			cf.SetDefaultExpandOptions(pf.DefaultExpandOptions()...)

			// per ME lookup table for expansion, add items as required from metadata
			lookup := map[string]string{
				"type": ct.String(),
			}
			expandOptions := []config.ExpandOption{
				config.LookupTable(lookup),
				config.Prefix("joinpath", profile.JoinPathPrefix, lookup),
			}

			// build entity using lookup table for expansion, with custom functions
			variables := make(values.Variables, len(entity.Variables))
			for idx, v := range entity.Variables {
				variables[idx] = values.Variable{
					Type:  v.Type,
					Name:  v.Name,
					Value: config.Expand[string](cf, v.Value, expandOptions...),
				}
			}

			var attrs values.NameValues
			for attr := range strings.SplitSeq(config.Expand[string](cf, entity.Attributes.String(), expandOptions...), ",") {
				attrs = append(attrs, attr)
			}

			e := ManagedEntity{
				Name:       config.Expand[string](cf, entity.Name, expandOptions...),
				Types:      cf.ExpandStringSlice(entity.Types, expandOptions...),
				Attributes: attrs,
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
