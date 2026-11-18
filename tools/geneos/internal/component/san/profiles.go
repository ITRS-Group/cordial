package san

import (
	"errors"
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
		vals.Gateways = append(vals.Gateways, profile.ExpandCommaList(pf, san.Gateways)...)

		vals.Types = pf.ExpandStringSlice(san.Types)

		log.Debug("attributes", slog.Any("attrs", san.Attributes))
		vals.Attributes = append(vals.Attributes, splitList(config.Expand[string](pf, san.Attributes.String()))...)

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
			entityVals, err := processSANManagedEntity(pf, h, me)
			if err != nil {
				if errors.Is(err, geneos.ErrNotExist) {
					log.Debug("no instances found for managed entity", slog.String("for-each", me.ForEach), slog.String("name", me.Name))
				} else {
					log.Warn("skipping managed entity", slog.String("for-each", me.ForEach), slog.String("name", me.Name), slog.Any("error", err))
				}
				continue
			}
			vals.Attributes = append(vals.Attributes, entityVals.Attributes...)
			vals.Types = append(vals.Types, entityVals.Types...)
			vals.Variables = append(vals.Variables, entityVals.Variables...)
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

func processSANManagedEntity(pf *config.Config, h *geneos.Host, entity ManagedEntity) (vals values.Values, err error) {
	// without a for-each the entity is used as defined, but still
	// expanded using the profile configuration
	entities := []ManagedEntity{expandStaticEntity(pf, entity)}

	if entity.ForEach != "" {
		ct := geneos.ParseComponent(entity.ForEach)
		if ct == nil {
			return vals, fmt.Errorf("%w: unknown for-each component type %q", geneos.ErrInvalidArgs, entity.ForEach)
		}

		// default is match all
		match := []string{"*"}
		if len(entity.Match) > 0 {
			match = entity.Match
		}

		names, err := instance.Match(h, ct, false, true, match...)
		if err != nil {
			return vals, geneos.ErrNotExist
		}

		// now ignore instances that match the ignore patterns
		names = filterIgnored(names, entity.Ignore)
		if len(names) == 0 {
			// there is nothing left to match. an empty list of names
			// would match all instances, so stop here
			return vals, nil
		}

		// expand the entities slice based on the for-each logic
		instances := instance.Instances(h, ct, instance.MatchNames(names...))

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

			e := ManagedEntity{
				Name:       config.Expand[string](cf, entity.Name, expandOptions...),
				Types:      cf.ExpandStringSlice(entity.Types, expandOptions...),
				Attributes: splitList(config.Expand[string](cf, entity.Attributes.String(), expandOptions...)),
				Variables:  variables,
			}
			entities[idx] = e
		}
	}

	// append managed entity information to the view, prefixed with the entity name
	for _, entity := range entities {
		for _, t := range entity.Types {
			vals.Types = append(vals.Types, entity.Name+"/"+t)
		}
		for _, a := range entity.Attributes {
			vals.Attributes = append(vals.Attributes, entity.Name+"/"+a)
		}
		for _, v := range entity.Variables {
			vals.Variables = append(vals.Variables, values.Variable{
				Type:  v.Type,
				Name:  entity.Name + "/" + v.Name,
				Value: v.Value,
			})
		}
	}

	return vals, nil
}

// expandStaticEntity expands the name, types, attributes and variables
// of an entity that is not generated from instances, using the profile
// configuration.
func expandStaticEntity(pf *config.Config, entity ManagedEntity) ManagedEntity {
	variables := make([]values.Variable, len(entity.Variables))
	for idx, v := range entity.Variables {
		variables[idx] = values.Variable{
			Type:  v.Type,
			Name:  v.Name,
			Value: config.Expand[string](pf, v.Value),
		}
	}

	entity.Name = config.Expand[string](pf, entity.Name)
	entity.Types = pf.ExpandStringSlice(entity.Types)
	entity.Attributes = splitList(config.Expand[string](pf, entity.Attributes.String()))
	entity.Variables = variables
	return entity
}

// splitList splits an expanded comma-separated list, ignoring empty
// items.
func splitList(s string) (list []string) {
	for item := range strings.SplitSeq(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			list = append(list, item)
		}
	}
	return
}

// filterIgnored removes the instance names, as returned by
// [instance.Match] and so with an `@HOST` suffix, that match any of the
// ignore patterns. A pattern is matched against the instance name
// without the host and, to allow host specific patterns, the full name.
func filterIgnored(names []string, ignore []string) []string {
	if len(ignore) == 0 {
		return names
	}
	return slices.DeleteFunc(slices.Clone(names), func(name string) bool {
		bare, _, _ := strings.Cut(name, "@")
		return slices.ContainsFunc(ignore, func(pattern string) bool {
			if matched, _ := path.Match(pattern, bare); matched {
				return true
			}
			matched, _ := path.Match(pattern, name)
			return matched
		})
	})
}
