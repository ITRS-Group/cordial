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
	"cmp"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

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
	mappings := maps.Clone(config.Get[map[string]string](conf, "inventory.mappings"))
	if mappings == nil {
		mappings = map[string]string{}
	}
	mappings["hostname"] = hostname
	mappings["hosttype"] = "unknown"

	// inventory keys are lower-cased by the configuration package, so
	// look hosts up the same way
	hostmapping, hostmapOK := hostmap[strings.ToLower(hostname)]

	if componentOverride != "" {
		if conf.IsSet(conf.Join("components", componentOverride)) {
			if hostmapOK {
				mappings = maps.Clone(hostmapping)
			}
			mappings["hosttype"] = componentOverride
		}
	} else if hostmapOK {
		mappings = maps.Clone(hostmapping)
	}

	initialComponentType := mappings["hosttype"]
	component := conf.Sub(config.Join("components", mappings["hosttype"]))
	// support aliases much like symlinks, limiting the number of levels
	var i int
	for i = 0; i < 10; i++ {
		if !component.IsSet("alias") {
			break
		}
		alias := config.Get[string](component, "alias")
		mappings["hosttype"] = alias
		component = conf.Sub("components." + mappings["hosttype"])
	}
	if i == 10 {
		log.Warn("component alias loop, skipping", slog.String("component", initialComponentType))
		return nil, ""
	}
	finalComponentType = mappings["hosttype"]

	// a SAN can have multiple Entities, so first build some default
	// sets of attributes, types and variables to use later on
	globalAttrs := map[string]string{}
	for _, a := range config.Get[[]map[string]string](conf, "components.defaults.attributes", config.LookupTable(mappings)) {
		globalAttrs[a["name"]] = a["value"]
	}
	globalTypes := config.Get[[]string](conf, "components.defaults.types", config.LookupTable(mappings))
	globalVars := getVars(conf, "components.defaults.variables", config.LookupTable(mappings))

	np = &netprobe.Netprobe{
		Compatibility: 1,
		XMLNs:         config.Get[string](conf, "geneos.sans.xmlns"),
		XSI:           config.Get[string](conf, "geneos.sans.xsi"),
		SelfAnnounce: &netprobe.SelfAnnounce{
			Enabled:                  true,
			RetryInterval:            int(config.Get[time.Duration](conf, "geneos.sans.retry-interval").Seconds()),
			RequireReverseConnection: reverseConnections(conf),
			ProbeName:                config.Get[string](component, "probe-name", config.LookupTable(mappings)),
			Gateways:                 cs.Gateways(hostname),
		},
	}

	entities := config.Get[any](component, "entities")
	if entities == nil {
		log.Error("no entities defined for component, skipping", slog.String("hostname", hostname), slog.String("component", mappings["hosttype"]))
		return
	}
	entityList, ok := entities.([]any)
	if !ok {
		log.Error("entities is not a list", slog.String("hostname", hostname), slog.String("component", mappings["hosttype"]), slog.Any("entities", entities))
		return nil, ""
	}

	// extract defaults for this component, if they exist
	defAttrs := maps.Clone(globalAttrs)
	for _, a := range config.Get[[]map[string]string](component, "attributes", config.LookupTable(mappings)) {
		defAttrs[a["name"]] = a["value"]
	}
	defTypes := slices.Concat(globalTypes, config.Get[[]string](component, "types", config.LookupTable(mappings)))
	defVars := slices.Concat(globalVars, getVars(component, "variables", config.LookupTable(mappings)))

	// iterate over Entities, filling in defaults as defined
	for i, em := range entityList {
		_, ok := em.(map[string]any)
		if !ok {
			continue
		}

		entity := component.Sub(config.Join("entities", strconv.Itoa(i)))
		ent := netprobe.ManagedEntity{
			Name: config.Get[string](entity, "name", config.LookupTable(mappings)),
		}

		attrs := config.Get[[]map[string]string](entity, "attributes",
			config.LookupTable(mappings),
			config.Prefix("uuid", func(ci map[string]any, s string, b bool, _ ...any) (string, error) {
				s = strings.TrimPrefix(s, "uuid:")
				uuidSource, ok := ci[s].(string) // config.Get[string](c, s, config.TrimPrefix(), config.LookupTable(mappings))
				if !ok {
					return "", nil
				}
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

		types := slices.Concat(defTypes, config.Get[[]string](entity, "types", config.LookupTable(mappings)))

		if len(types) > 0 {
			ent.Types = &netprobe.Types{Types: types}
		}

		vars := slices.Concat(defVars, getVars(entity, "variables", config.LookupTable(mappings)))
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

	allGatewayDetails := config.Get[[]map[string]string](conf, "geneos.gateways")

	netprobeID := netprobeID(conf, hostname)
	gatewayNames := OrderGateways(netprobeID, allGateways)
	if len(gatewayNames) > 0 {
		for _, g := range gatewayNames {
			gateways = append(gateways, GatewayDetails(g, allGatewayDetails))
		}
	} else {
		fallback := config.Get[map[string]string](conf, "geneos.fallback-gateway")
		name := cmp.Or(fallback["name"], fallback["primary"])
		if name == "" {
			log.Error("fallback gateway not configured correctly")
			return
		}

		gateways = append(gateways, GatewayDetails(name, []map[string]string{fallback}))
	}

	maxGateways := config.Get[int](conf, "geneos.sans.gateways", config.DefaultValue(1))
	log.Debug("selecting gateways for host with prefix", slog.Int("maxGateways", maxGateways), slog.String("hostname", hostname), slog.String("netprobeID", netprobeID))

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

// reverseConnections returns the `geneos.sans.reverse-connections`
// setting. The singular `reverse-connection`, which is what the code
// used to read, is still accepted if it is set.
//
// TODO(maintainer): the documented default for this setting was `true`
// but the original code never applied it and always generated
// `requireReverseConnection` as `false`, which is the default kept here
// (in san-config.defaults.yaml) to avoid changing the generated XML of
// existing setups. The documented and the effective defaults differ and
// a decision is needed on which one is wanted.
func reverseConnections(conf *config.Config) bool {
	if conf.IsSet(conf.Join("geneos", "sans", "reverse-connection")) {
		return config.Get[bool](conf, "geneos.sans.reverse-connection")
	}
	return config.Get[bool](conf, "geneos.sans.reverse-connections")
}

type varConf struct {
	Name  string
	Type  string
	Value any
}

// toInt64 converts a numeric configuration value to an int64, if it is
// an integer value
func toInt64(v any) (i int64, ok bool) {
	switch v := v.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), uint64(v) <= math.MaxInt64
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), uint64(v) <= math.MaxInt64
	case float32:
		return toInt64(float64(v))
	case float64:
		return int64(v), v == math.Trunc(v) && v >= math.MinInt64 && v <= math.MaxInt64
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return i, err == nil
	}
	return
}

// toFloat64 converts a numeric configuration value to a float64
func toFloat64(v any) (f float64, ok bool) {
	switch v := v.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f, err == nil
	}
	if i, ok := toInt64(v); ok {
		return float64(i), true
	}
	return
}

// getVars returns the Geneos variables defined in the configuration
// list at key. Variables with an unsupported type or a value that cannot
// be converted to the type are logged and skipped.
func getVars(conf *config.Config, key string, options ...config.ExpandOption) (vars []geneos.Vars) {
	var vs []varConf
	if err := conf.UnmarshalKey(key, &vs, config.NoExpand()); err != nil {
		log.Error("skipping key", slog.Any("error", err), slog.String("key", key))
		return
	}

	for _, v := range vs {
		vr := geneos.Vars{
			Name: v.Name,
		}
		valid := true
		switch strings.ToLower(v.Type) {
		case "bool", "boolean":
			// Boolean       *bool          `xml:"boolean,omitempty" json:",omitempty" yaml:",omitempty"`
			switch val := v.Value.(type) {
			case bool:
				vr.Boolean = &val
			case string:
				valbool, err := strconv.ParseBool(strings.TrimSpace(val))
				vr.Boolean = &valbool
				valid = err == nil
			default:
				valid = false
			}
		case "float", "double":
			// Double        *float64       `xml:"double,omitempty" json:",omitempty" yaml:",omitempty"`
			var val float64
			val, valid = toFloat64(v.Value)
			vr.Double = &val
		case "int", "integer":
			// Integer       *int64         `xml:"integer,omitempty" json:",omitempty" yaml:",omitempty"`
			var val int64
			val, valid = toInt64(v.Value)
			vr.Integer = &val
		case "string":
			// String        string         `xml:"string,omitempty" json:",omitempty" yaml:",omitempty"`
			switch val := v.Value.(type) {
			case string:
				vr.String = config.Expand[string](conf, val, options...)
			case bool, int, int64, uint64, float64:
				vr.String = fmt.Sprint(val)
			default:
				valid = false
			}
		case "stringlist":
			// StringList    *StringList    `xml:"stringList,omitempty" json:",omitempty" yaml:",omitempty"`
			vr.StringList = &geneos.StringList{}

			switch val := v.Value.(type) {
			case []any:
				for _, j := range val {
					if s, ok := j.(string); ok {
						vr.StringList.Strings = append(vr.StringList.Strings, config.Expand[string](conf, s, options...))
					}
				}
			case []string:
				for _, s := range val {
					vr.StringList.Strings = append(vr.StringList.Strings, config.Expand[string](conf, s, options...))
				}
			default:
				valid = false
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
							vr.Regex.Flags.CaseInsensitive = new(true)
						}
						if strings.Contains(r[2], "s") {
							vr.Regex.Flags.DotMatchesAll = new(true)
						}
					}
				} else {
					vr.Regex.Regex = val
				}
			} else {
				valid = false
			}
		default:
			log.Warn("variable type not supported, skipping", slog.String("name", v.Name), slog.String("type", v.Type))
			continue
		}
		if !valid {
			log.Warn("variable value is not valid for the type, skipping", slog.String("name", v.Name), slog.String("type", v.Type), slog.Any("value", v.Value))
			continue
		}
		vars = append(vars, vr)
	}
	return
}

// groupingRegexp is a compiled `geneos.sans.grouping` expression, or
// the error from compiling it
type groupingRegexp struct {
	re  *regexp.Regexp
	err error
}

// groupingRegexps caches compiled grouping expressions, by pattern, so
// they are only compiled once and not for every request
var groupingRegexps sync.Map

func compileGrouping(pattern string) groupingRegexp {
	if g, ok := groupingRegexps.Load(pattern); ok {
		return g.(groupingRegexp)
	}
	g := groupingRegexp{}
	g.re, g.err = regexp.Compile(pattern)
	if g.err == nil && g.re.NumSubexp() < 1 {
		g.re, g.err = nil, fmt.Errorf("grouping expression %q has no capture group", pattern)
	}
	groupingRegexps.Store(pattern, g)
	return g
}

func netprobeID(conf *config.Config, hostname string) (id string) {
	// extract the part of the hostname used for netprobeID, default to hostname
	id = hostname
	if g := config.Get[string](conf, "geneos.sans.grouping"); g != "" {
		grouping := compileGrouping(g)
		if grouping.err != nil {
			log.Error("ignoring grouping", slog.Any("error", grouping.err))
			return
		}
		if m := grouping.re.FindStringSubmatch(hostname); len(m) > 1 {
			id = m[1]
		} else {
			log.Warn("grouping did not match, using full hostname", slog.String("hostname", hostname))
		}
	}
	return
}
