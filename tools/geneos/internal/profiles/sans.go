package profiles

import (
	"log/slog"
	"os"
	"path"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

type San struct {
	Common          `mapstructure:",squash"`
	Gateways        map[string]string `yaml:"gateways,omitempty"`
	Types           []string          `yaml:"types,omitempty"`
	Attributes      values.NameValues `yaml:"attributes,omitempty"`
	Variables       []values.Variable `yaml:"variables,omitempty"`
	ManagedEntities []ManagedEntity   `yaml:"managed-entities,omitempty"`
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

		vals := values.Values{
			Gateways:   san.Gateways,
			Types:      san.Types,
			Attributes: san.Attributes,
			Variables:  san.Variables,
		}

		log.Debug("processing SAN", slog.String("name", name), slog.Any("values", vals))

		// now add managed-entities, prefixing each with the "name/"
		for _, me := range san.ManagedEntities {
			// start with clean values for each managed entity
			if extraVals, err := processSANManagedEntity(me); err == nil {
				// merge into top-level values
				log.Debug("merging ME attributes", slog.String("managedEntity", me.Name), slog.Any("attributes", extraVals.Attributes))
				vals.Attributes = append(vals.Attributes, extraVals.Attributes...)
				log.Debug("merging ME types", slog.String("managedEntity", me.Name), slog.Any("types", extraVals.Types))
				vals.Types = append(vals.Types, extraVals.Types...)
				log.Debug("merging ME variables", slog.String("managedEntity", me.Name), slog.Any("variables", extraVals.Variables))
				vals.Variables = append(vals.Variables, extraVals.Variables...)
			}
		}

		log.Debug("final SAN values", slog.String("name", name), slog.Any("values", vals))

		// apply common params
		vals, err := applyCommonParams(san.Common, vals)
		if err != nil {
			log.Error("failed to apply common params", slog.String("name", name), slog.Any("error", err))
			return
		}

		instances := instance.Instances(geneos.LOCAL, ct, instance.MatchNames(name))
		log.Debug("retrieved SAN instances", slog.String("name", name), slog.Int("count", len(instances)))
		if len(instances) == 0 {
			// create here
			log.Debug("creating SAN instance", slog.String("name", name))
			instance.Add(geneos.LOCAL, ct, name, 0, vals,
				instance.CertBundle(san.Common.CertBundle),
				instance.CertBundlePassword(san.Common.CertBundlePassword),
			)
			instance.Do(geneos.LOCAL, ct, []string{name}, func(i geneos.Instance, a ...any) (resp *responses.General) {
				resp = responses.New[responses.General](i)
				resp.Err = instance.Start(i)
				return
			}, vals).Report(os.Stdout, responses.IgnoreErr(geneos.ErrRunning))
		} else {
			// update the first / only instance
			i := instances[0]
			log.Debug("updating SAN instance", slog.String("name", i.Name()))
			cf := i.Config()
			keyfile := config.Get[config.KeyFile](cf, "keyfile")

			// Reset existing SAN configuration for this instance before applying new values
			config.Delete(cf, values.GATEWAYS)
			config.Delete(cf, values.ATTRIBUTES)
			config.Delete(cf, values.TYPES)
			config.Delete(cf, values.VARIABLES)
			config.Delete(cf, values.ENVIRONMENT)
			config.Delete(cf, values.MANAGED_ENTITIES)

			if ncf, err := values.Set(i, vals, keyfile); err == nil {
				i.SetConfig(ncf)
			}
			if resp := instance.Write(i); resp.Err != nil {
				i.Log().Error("write failed", slog.Any("error", resp.Err))
				return
			}
		}
	}
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
						config.Prefix("select", selectPrefix),
						config.Prefix("replace", replacePrefix),
					),
				}
			}

			name := config.Expand[string](cf, entity.Name,
				config.LookupTable(lookup),
				config.Prefix("select", selectPrefix),
				config.Prefix("replace", replacePrefix),
			)

			e := ManagedEntity{
				Name: name,
				Types: cf.ExpandStringSlice(entity.Types,
					config.LookupTable(lookup),
					config.Prefix("select", selectPrefix),
					config.Prefix("replace", replacePrefix),
				),
				Attributes: cf.ExpandStringSlice(entity.Attributes,
					config.LookupTable(lookup),
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
