/*
Copyright © 2024 ITRS Group

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
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos"
	"github.com/itrs-group/cordial/pkg/geneos/netprobe"
)

// NetprobeConfig returns a netprobe.Netprobe struct for output via XML
// marshalling. The returned `finalComponentType` is for logging in the
// caller for those cases where the type changes from the requested
// one..
//
// If hostname is not found in hosts then return the unknown component
// settings and the fallback gateway(s)
//
// If no gateways are available then do the same as above for hardware
// probes - not sure about apps at this stage
func (cs *ConfigServer) NetprobeConfig(hostname string, componentOverride string) (np *netprobe.Netprobe, finalComponentType string) {
	cs.RLock()
	conf := cs.conf
	hostmap := cs.hosts
	cs.RUnlock()

	// build `unknown` mappings table as default, use inventory.mappings
	mappings := conf.GetStringMapString("inventory.mappings")
	mappings["hostname"] = hostname
	mappings["hosttype"] = "unknown"

	_, hostmapOK := hostmap[hostname]

	if componentOverride != "" {
		if conf.IsSet("components." + componentOverride) {
			if hostmapOK {
				mappings = maps.Clone(hostmap[hostname])
			}
			mappings["hosttype"] = componentOverride
		}
	} else if hostmapOK {
		mappings = maps.Clone(hostmap[hostname])
	}

	initialComponentType := mappings["hosttype"]
	component := conf.Sub(config.Join("components", mappings["hosttype"]))
	// support aliases much like symlinks, limiting the number of levels
	var i int
	for i = 0; i < 10; i++ {
		if !component.IsSet("alias") {
			break
		}
		alias := component.GetString("alias")
		mappings["hosttype"] = alias
		component = conf.Sub("components." + mappings["hosttype"])
	}
	if i == 10 {
		log.Warn().Msgf("component alias loop for %s, skipping", initialComponentType)
		return
	}
	finalComponentType = mappings["hosttype"]

	// a SAN can have multiple Entities, so first build some default
	// sets of attributes, types and variables to use later on
	globalAttrs := map[string]string{}
	for _, a := range conf.GetSliceStringMapString("components.defaults.attributes", config.LookupTable(mappings)) {
		globalAttrs[a["name"]] = a["value"]
	}
	globalTypes := conf.GetStringSlice("components.defaults.types", config.LookupTable(mappings))
	globalVars := getVars(conf, "components.defaults.variables", config.LookupTable(mappings))

	np = &netprobe.Netprobe{
		Compatibility: 1,
		XMLNs:         conf.GetString("geneos.sans.xmlns"),
		XSI:           conf.GetString("geneos.sans.xsi"),
		SelfAnnounce: &netprobe.SelfAnnounce{
			Enabled:                  true,
			RetryInterval:            int(conf.GetDuration("geneos.sans.retry-interval").Seconds()),
			RequireReverseConnection: conf.GetBool("geneos.sans.reverse-connection"),
			ProbeName:                component.GetString("probe-name", config.LookupTable(mappings)),
			Gateways:                 cs.Gateways(hostname),
		},
	}

	entities := component.Get("entities")
	if entities == nil {
		log.Error().Msgf("skipping %s: no entities defined for component type %s", hostname, mappings["hosttype"])
		return
	}
	t := reflect.TypeOf(entities)
	if t.Kind() != reflect.Slice {
		log.Fatal().Msgf("entities is not a slice: %T -> %v", entities, entities)
	}

	// extract defaults for this component, if they exist
	defAttrs := maps.Clone(globalAttrs)
	for _, a := range component.GetSliceStringMapString("attributes", config.LookupTable(mappings)) {
		defAttrs[a["name"]] = a["value"]
	}
	defTypes := append(globalTypes, component.GetStringSlice("types", config.LookupTable(mappings))...)
	defVars := append(globalVars, getVars(component, "variables", config.LookupTable(mappings))...)

	// iterate over Entities, filling in defaults as defined
	for i, em := range entities.([]any) {
		_, ok := em.(map[string]any)
		if !ok {
			continue
		}

		entity := component.Sub(config.Join("entities", strconv.Itoa(i)))
		ent := netprobe.ManagedEntity{
			Name: entity.GetString("name", config.LookupTable(mappings)),
		}

		attrs := entity.GetSliceStringMapString("attributes",
			config.LookupTable(mappings),
			config.Prefix("uuid", func(c *config.Config, s string, b bool) (string, error) {
				s = strings.TrimPrefix(s, "uuid:")
				uuidSource := c.GetString(s, config.TrimPrefix(), config.LookupTable(mappings))
				if b {
					uuidSource = strings.TrimSpace(uuidSource)
				}
				return uuid.NewSHA1(uuidNS, []byte(uuidSource)).String(), nil
			}))
		attributes := maps.Clone(defAttrs)
		for _, a := range attrs {
			attributes[a["name"]] = a["value"]
		}

		a := slices.Sorted(maps.Keys(attributes))
		if len(a) > 0 {
			ent.Attributes = &netprobe.Attributes{}
			for _, attr := range a {
				ent.Attributes.Attributes = append(ent.Attributes.Attributes, geneos.Attribute{
					Name:  attr,
					Value: attributes[attr],
				})
			}
		}

		types := append(defTypes, entity.GetStringSlice("types", config.LookupTable(mappings))...)

		if len(types) > 0 {
			ent.Types = &netprobe.Types{}
			for _, t := range types {
				ent.Types.Types = append(ent.Types.Types, t)
			}
		}

		vars := append(defVars, getVars(entity, "variables", config.LookupTable(mappings))...)
		if len(vars) > 0 {
			ent.Vars = &netprobe.Vars{
				Vars: vars,
			}
		}

		np.SelfAnnounce.ManagedEntities = append(np.SelfAnnounce.ManagedEntities, ent)
	}

	return
}

// Gateways returns a slice of all the gateways that this SAN should try
// to connect to.
//
// When a named gateway is configured with both a primary and standby
// host then this counts as one gateway but results in two slice
// elements. The number of named gateways is limited by the
// `geneos.sans.gateways` value which defaults to 1.
func (cs *ConfigServer) Gateways(hostname string) (NPgateways []netprobe.Gateway) {
	gateways := []GatewaySet{}

	cs.RLock()
	conf := cs.conf
	allGateways := cs.gateways
	cs.RUnlock()

	allGatewayDetails := conf.GetSliceStringMapString("geneos.gateways")

	netprobeID := netprobeID(conf, hostname)
	gatewayNames := OrderGateways(netprobeID, allGateways)
	if len(gatewayNames) > 0 {
		for _, g := range gatewayNames {
			gateways = append(gateways, GatewayDetails(g, allGatewayDetails))
		}
	} else {
		gatewayNames = []string{conf.GetString("geneos.fallback-gateway.name", config.Default(conf.GetString("geneos.fallback-gateway.primary")))}
		if len(gatewayNames) == 0 {
			log.Error().Msg("fallback gateway not configured correctly")
			return
		}

		gateways = append(gateways, GatewayDetails(gatewayNames[0], []map[string]string{conf.GetStringMapString("geneos.fallback-gateway")}))
	}

	maxGateways := cf.GetInt("geneos.sans.gateways", config.Default(1))
	log.Debug().Msgf("selecting up to %d gateways for host %s with prefix %s", maxGateways, hostname, netprobeID)

	i := 0
	for _, gateway := range gateways {
		// if maxGateways is not 0 and we've been around the loop at
		// least that many times, quit
		if maxGateways > 0 && i >= maxGateways {
			break
		}

		NPgateways = append(NPgateways,
			netprobe.Gateway{Hostname: gateway.Primary, Port: gateway.PrimaryPort, Secure: gateway.Secure},
		)

		if gateway.Standby != "" {
			NPgateways = append(NPgateways,
				netprobe.Gateway{Hostname: gateway.Standby, Port: gateway.StandbyPort, Secure: gateway.Secure},
			)
		}

		i++
	}

	return
}

type varConf struct {
	Name  string
	Type  string
	Value any
}

func getVars(conf *config.Config, key string, options ...config.ExpandOptions) (vars []geneos.Vars) {
	var vs []varConf
	if err := conf.UnmarshalKey(key, &vs); err != nil {
		log.Error().Err(err).Msgf("skipping %s", key)
		return
	}

	for _, v := range vs {
		vr := geneos.Vars{
			Name: v.Name,
		}
		switch strings.ToLower(v.Type) {
		case "bool", "boolean":
			// Boolean       *bool          `xml:"boolean,omitempty" json:",omitempty" yaml:",omitempty"`
			switch val := v.Value.(type) {
			case bool:
				vr.Boolean = &val
			case string:
				valbool, _ := strconv.ParseBool(val)
				vr.Boolean = &valbool
			}
		case "float", "double":
			// Double        *float64       `xml:"double,omitempty" json:",omitempty" yaml:",omitempty"`
			switch val := v.Value.(type) {
			case float64:
				vr.Double = &val
			case string:
				var val64 float64
				val64, _ = strconv.ParseFloat(val, 64)
				vr.Double = &val64
			}
		case "int", "integer":
			// Integer       *int64         `xml:"integer,omitempty" json:",omitempty" yaml:",omitempty"`
			switch val := v.Value.(type) {
			case int:
				var val64 int64
				val64 = int64(val)
				vr.Integer = &val64
			case string:
				var val64 int64
				val64, _ = strconv.ParseInt(val, 10, 64)
				vr.Integer = &val64
			}
		case "string":
			// String        string         `xml:"string,omitempty" json:",omitempty" yaml:",omitempty"`
			if val, ok := v.Value.(string); ok {
				vr.String = config.ExpandString(val, options...)
			}
		case "stringlist":
			// StringList    *StringList    `xml:"stringList,omitempty" json:",omitempty" yaml:",omitempty"`
			vr.StringList = &geneos.StringList{}

			switch val := v.Value.(type) {
			case []any:
				for _, j := range val {
					if s, ok := j.(string); ok {
						vr.StringList.Strings = append(vr.StringList.Strings, config.ExpandString(s, options...))
					}
				}
			}
		case "regex", "regexp":
			// Regex         *Regex         `xml:"regex,omitempty" json:",omitempty" yaml:",omitempty"`
			vr.Regex = &geneos.Regex{}
			if val, ok := v.Value.(string); ok {
				// '/regex/flags' or plain string
				if strings.HasPrefix(val, "/") {
					r := strings.SplitN(val, "/", 3)
					vr.Regex.Regex = r[1]
					if len(r) == 3 {
						vr.Regex.Flags = &geneos.RegexFlags{}
						if strings.Contains(r[2], "i") {
							i := true
							vr.Regex.Flags.CaseInsensitive = &i
						}
						if strings.Contains(r[2], "s") {
							s := true
							vr.Regex.Flags.DotMatchesAll = &s
						}
					}
				} else {
					vr.Regex.Regex = val
				}
			}
		default:
			log.Warn().Msgf("variable type %s not supported, skipping", v.Type)
		}
		vars = append(vars, vr)
	}
	return
}

func netprobeID(conf *config.Config, hostname string) (id string) {
	// extract the part of the hostname used for netprobeID, default to hostname
	id = hostname
	if g := conf.GetString("geneos.sans.grouping"); g != "" {
		if r, err := regexp.Compile(g); err != nil {
			log.Error().Err(err).Msg("ignoring grouping")
		} else {
			if m := r.FindStringSubmatch(hostname); len(m) > 0 {
				id = m[1]
			} else {
				log.Warn().Msgf("grouping for %s did not match, using full hostname", hostname)
			}
		}
	}
	return
}
