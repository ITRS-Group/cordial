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

package geneos

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/strftime"
)

// utility interfaces and functions

// KeyedObject is an interface to structures that have a GetKey() method that returns a string
type KeyedObject interface {
	GetKey() string
}

// RemoveDuplicates iterates over any slice that satisfies KeyedObject
// and returns a slice of the same type but with duplicates removed. The
// first item is kept in all cases.
func RemoveDuplicates[T KeyedObject](kvs []T) []T {
	result := []T{}
	a := make(map[string]bool)
	for _, v := range kvs {
		if _, ok := a[v.GetKey()]; !ok {
			result = append(result, v)
			a[v.GetKey()] = true
		}
	}
	return result
}

// UnrollProbes takes the top level geneos.Probes struct and returns a
// map of probe name to geneos.Probe objects while setting the Type
// field to the type of probe
func UnrollProbes(in *Probes) (probes map[string]Probe) {
	probes = make(map[string]Probe)

	if in == nil {
		return
	}

	for _, p := range in.Probes {
		if p.Disabled {
			continue
		}
		probes[p.Name] = p
	}
	for _, f := range in.FloatingProbes {
		if f.Disabled {
			continue
		}
		probes[f.Name] = Probe{
			Name:      f.Name,
			Type:      ProbeTypeFloating,
			Disabled:  f.Disabled,
			ProbeInfo: ProbeInfo{ProbeInfoWithoutPort: f.ProbeInfoWithoutPort},
		}
	}
	for _, v := range in.VirtualProbes {
		if v.Disabled {
			continue
		}
		probes[v.Name] = Probe{
			Name:     v.Name,
			Type:     ProbeTypeVirtual,
			Disabled: v.Disabled,
		}
	}

	// top level has no group defaults so just loop over sub-groups and
	// append non-disabled probes
	for _, g := range in.ProbeGroups {
		p, f, v := unrollProbeGroup(&g)
		for _, p := range p {
			if p.Disabled {
				continue
			}
			probes[p.Name] = p
		}
		for _, f := range f {
			if f.Disabled {
				continue
			}
			probes[f.Name] = Probe{
				Name:      f.Name,
				Type:      ProbeTypeFloating,
				Disabled:  f.Disabled,
				ProbeInfo: ProbeInfo{ProbeInfoWithoutPort: f.ProbeInfoWithoutPort},
			}
		}
		for _, v := range v {
			if v.Disabled {
				continue
			}
			probes[v.Name] = Probe{
				Name:     v.Name,
				Type:     ProbeTypeVirtual,
				Disabled: v.Disabled,
			}
		}
	}
	return
}

// unrollProbeGroup works through a geneos.ProbeGroup and applies
// defaults to all it's contents and returns three slices of probe types
func unrollProbeGroup(in *ProbeGroup) (probes []Probe, floatingProbes []FloatingProbe, virtualProbes []VirtualProbe) {
	if in.Disabled {
		return
	}

	for _, p := range in.Probes {
		if p.Disabled {
			continue
		}
		setDefaults(in.ProbeInfo, &p.ProbeInfo)
		probes = append(probes, p)

	}
	for _, f := range in.FloatingProbes {
		if f.Disabled {
			continue
		}
		setDefaults(in.ProbeInfoWithoutPort, &f.ProbeInfoWithoutPort)
		floatingProbes = append(floatingProbes, f)
	}
	for _, v := range in.VirtualProbes {
		if v.Disabled {
			continue
		}
		// virtual probes have no other settings, defaults do not matter
		virtualProbes = append(virtualProbes, v)
	}

	for _, g := range in.ProbeGroups {
		setDefaults(in.ProbeInfo, &g.ProbeInfo)
		p, f, v := unrollProbeGroup(&g)

		for _, p := range p {
			if p.Disabled {
				continue
			}
			probes = append(probes, p)

		}
		for _, f := range f {
			if f.Disabled {
				continue
			}
			// updateStruct(&f.ProbeInfoWithoutPort, in.ProbeInfoWithoutPort)
			floatingProbes = append(floatingProbes, f)
		}
		for _, v := range v {
			if v.Disabled {
				continue
			}
			virtualProbes = append(virtualProbes, v)
		}
	}

	return
}

// UnrollEntities func
func UnrollEntities(in *ManagedEntities, types map[string]Type) (entities map[string]ManagedEntity) {
	entities = make(map[string]ManagedEntity)

	if in == nil {
		return
	}

	for _, entity := range in.Entities {
		if entity.Disabled {
			continue
		}

		resolveEntitySamplers(nil, &entity, types)
		entities[entity.Name] = entity
	}

	for _, group := range in.ManagedEntityGroups {
		resolveGroup(&group, types)
		unrolledEntities := unrollEntityGroup(&group, types)
		for _, entity := range unrolledEntities {
			if entity.Disabled {
				continue
			}
			entities[entity.Name] = entity
		}
	}

	return
}

// unrollEntityGroup type
func unrollEntityGroup(group *ManagedEntityGroup, types map[string]Type) (entities map[string]ManagedEntity) {
	entities = make(map[string]ManagedEntity)

	if group.Disabled {
		return
	}

	// fix-up group here - resolve samplers etc.

	for _, entity := range group.Entities {
		if entity.Disabled {
			continue
		}

		setDefaults(group.ManagedEntityInfo, &entity.ManagedEntityInfo)

		// remove dups from merged slices in setDefaults
		entity.ManagedEntityInfo.Attributes = RemoveDuplicates(entity.ManagedEntityInfo.Attributes)
		entity.ManagedEntityInfo.Vars = RemoveDuplicates(entity.ManagedEntityInfo.Vars)

		resolveEntitySamplers(group, &entity, types)
		entities[entity.Name] = entity
	}

	group.ManagedEntityInfo.Attributes = RemoveDuplicates(group.ManagedEntityInfo.Attributes)
	group.ManagedEntityInfo.Vars = RemoveDuplicates(group.ManagedEntityInfo.Vars)

	for _, childGroup := range group.ManagedEntityGroups {
		setDefaults(group.ManagedEntityInfo, &childGroup.ManagedEntityInfo)
		// remove dups from merged slices in setDefaults

		resolveSamplersFromGroup(group.ManagedEntityInfo, &childGroup.ManagedEntityInfo, types)

		unrolledEntities := unrollEntityGroup(&childGroup, types)
		for _, entity := range unrolledEntities {
			if entity.Disabled {
				continue
			}
			entities[entity.Name] = entity
		}
	}

	return
}

func resolveGroup(group *ManagedEntityGroup, types map[string]Type) {
	if group.ResolvedSamplers == nil {
		group.ResolvedSamplers = map[string]bool{}
	}

	if group.AddTypes != nil {
		for _, at := range group.AddTypes.Types {
			for _, sampler := range types[at.Type].Samplers {
				if !sampler.Disabled {
					group.ResolvedSamplers[at.Type+":"+sampler.Name] = true
				}
			}
		}
	}

}

// resolveSamplersFromGroup processes RemoveTypes, RemoveSamplers and AddTypes in
// "from" and applies them to the ResolvedSamplers map in "to".
func resolveSamplersFromGroup(from ManagedEntityInfo, to *ManagedEntityInfo, types map[string]Type) {
	if to.ResolvedSamplers == nil {
		to.ResolvedSamplers = map[string]bool{}
		if from.ResolvedSamplers != nil {
			for k, v := range from.ResolvedSamplers {
				to.ResolvedSamplers[k] = v
			}
		}
	}

	if from.RemoveTypes != nil {
		for _, tr := range from.RemoveTypes.Types {
			if t, ok := types[tr.Type]; ok {
				for _, s := range t.Samplers {
					delete(to.ResolvedSamplers, tr.Type+":"+s.Name)
				}
			}
		}
	}

	if from.RemoveSamplers != nil {
		for _, sr := range from.RemoveSamplers.Samplers {
			delete(to.ResolvedSamplers, sr.Type.Type+":"+sr.Sampler)
		}
	}

	if to.AddTypes != nil {
		for _, at := range to.AddTypes.Types {
			for _, sampler := range types[at.Type].Samplers {
				if !sampler.Disabled {
					to.ResolvedSamplers[at.Type+":"+sampler.Name] = true
				}
			}
		}
	}
}

func resolveEntitySamplers(group *ManagedEntityGroup, entity *ManagedEntity, types map[string]Type) {
	if entity.ResolvedSamplers == nil {
		entity.ResolvedSamplers = map[string]bool{}
		if group != nil && group.ResolvedSamplers != nil {
			for k, v := range group.ResolvedSamplers {
				entity.ResolvedSamplers[k] = v
			}
		}
	}

	if group != nil {
		if group.RemoveTypes != nil {
			for _, tr := range group.RemoveTypes.Types {
				if t, ok := types[tr.Type]; ok {
					for _, s := range t.Samplers {
						delete(entity.ResolvedSamplers, tr.Type+":"+s.Name)
					}
				}
			}
		}

		if group.RemoveSamplers != nil {
			for _, sr := range group.RemoveSamplers.Samplers {
				delete(entity.ResolvedSamplers, sr.Type.Type+":"+sr.Sampler)
			}
		}
	}

	for _, s := range entity.Samplers {
		if !s.Disabled {
			entity.ResolvedSamplers[":"+s.Name] = true
		}
	}

	if entity.AddTypes != nil {
		for _, at := range entity.AddTypes.Types {
			for _, sampler := range types[at.Type].Samplers {
				if !sampler.Disabled {
					entity.ResolvedSamplers[at.Type+":"+sampler.Name] = true
				}
			}
		}
	}
}

func UnrollTypes(in *Types) (types map[string]Type) {
	types = make(map[string]Type)

	if in == nil {
		return
	}

	for _, t := range in.Types {
		if t.Disabled {
			continue
		}
		types[t.Name] = t
	}

	for _, g := range in.TypeGroups {
		t := unrollTypeGroup(&g)
		for _, t := range t {
			if t.Disabled {
				continue
			}
			types[t.Name] = t
		}
	}
	return
}

func unrollTypeGroup(in *TypeGroup) (types map[string]Type) {
	types = make(map[string]Type)
	for _, t := range in.Types {
		if t.Disabled {
			continue
		}
		types[t.Name] = t
	}
	for _, g := range in.TypeGroups {
		t := unrollTypeGroup(&g)
		for _, t := range t {
			if t.Disabled {
				continue
			}
			types[t.Name] = t
		}
	}
	return
}

func UnrollSamplers(in *Samplers) (samplers map[string]Sampler) {
	samplers = make(map[string]Sampler)

	if in == nil {
		return
	}

	for _, s := range in.Samplers {
		if s.Disabled {
			continue
		}
		samplers[s.Name] = s
	}
	for _, s := range in.Samplers {
		samplers[s.Name] = s
	}
	for _, g := range in.SamplerGroups {
		s := unrollSamplerGroup(&g)
		for _, s := range s {
			if s.Disabled {
				continue
			}
			samplers[s.Name] = s
		}
	}
	return
}

func unrollSamplerGroup(in *SamplerGroup) (samplers map[string]Sampler) {
	samplers = make(map[string]Sampler)
	for _, s := range in.Samplers {
		if s.Disabled {
			continue
		}
		samplers[s.Name] = s
	}
	for _, g := range in.SamplerGroups {
		s := unrollSamplerGroup(&g)
		for _, s := range s {
			if s.Disabled {
				continue
			}
			samplers[s.Name] = s
		}
	}
	return
}

func UnrollProcessDescriptors(in *ProcessDescriptors) (processDescriptors map[string]ProcessDescriptor) {
	processDescriptors = make(map[string]ProcessDescriptor)

	if in == nil {
		return
	}

	for _, p := range in.ProcessDescriptors {
		if p.Disabled {
			continue
		}
		processDescriptors[p.Name] = p
	}

	for _, g := range in.ProcessDescriptorGroups {
		p := unrollProcessDescriptorGroups(&g)
		for _, p := range p {
			if p.Disabled {
				continue
			}
			processDescriptors[p.Name] = p
		}
	}
	return
}

func unrollProcessDescriptorGroups(in *ProcessDescriptorGroup) (processDescriptors map[string]ProcessDescriptor) {
	processDescriptors = make(map[string]ProcessDescriptor)
	for _, p := range in.ProcessDescriptors {
		if p.Disabled {
			continue
		}
		processDescriptors[p.Name] = p
	}
	for _, g := range in.ProcessDescriptorGroups {
		p := unrollProcessDescriptorGroups(&g)
		for _, p := range p {
			if p.Disabled {
				continue
			}
			processDescriptors[p.Name] = p
		}
	}
	return
}

// UnrollRules resolves all the individual rules in the input by
// descending RuleGroups and returns a map of Rule with a key of rule
// path (group name and rule name joined by ` > ` as per Geneos
// convention) and each component followed by a priority in parenthesis.
// If a group is disabled it is skipped and the contents ignored. If a
// rule is disabled is is skipped.
func UnrollRules(in *Rules) (rules map[string]Rule) {
	rules = make(map[string]Rule)

	if in == nil {
		return
	}

	for _, p := range in.Rules {
		if p.Disabled {
			continue
		}
		rules[p.Name] = p
	}

	for _, g := range in.RuleGroups {
		p, grouppath := unrollRuleGroups(&g, "")
		for _, p := range p {
			if p.Disabled {
				continue
			}
			rules[grouppath+" > "+p.Name] = p
		}
	}
	return
}

func unrollRuleGroups(in *RuleGroup, parentpath string) (rules map[string]Rule, grouppath string) {
	rules = make(map[string]Rule)
	if parentpath != "" {
		grouppath = parentpath + " > " + in.Name
	} else {
		grouppath = in.Name
	}

	for _, p := range in.Rules {
		if p.Disabled {
			continue
		}
		rules[grouppath+" > "+p.Name] = p
	}
	for _, g := range in.RuleGroups {
		p, lowerpath := unrollRuleGroups(&g, grouppath)
		for _, p := range p {
			if p.Disabled {
				continue
			}
			rules[lowerpath+" > "+p.Name] = p
		}
	}
	return
}

// walk through a struct and set any defaults for fields that are
// empty/nil
//
// loop over fields and if the default is not a nil pointer or a nil
// values and the field is not set then assign. uses field name not
// position for defaults.
func setDefaults(source any, dest any) {
	if reflect.TypeOf(dest).Kind() != reflect.Pointer {
		return
	}
	sv := reflect.ValueOf(dest).Elem()

	for i := 0; i < sv.NumField(); i++ {
		fv := sv.Field(i)
		fn := sv.Type().Field(i).Name
		dv := reflect.ValueOf(source).FieldByName(fn)

		switch {
		case fv.Type() == reflect.PointerTo(reflect.TypeOf((bool)(false))):
			if fv.IsNil() && !dv.IsNil() && fv.CanSet() {
				fv.Set(dv)
			}
		case fv.Kind() == reflect.Int, fv.Kind() == reflect.String:
			if fv.Int() == 0 && dv.Int() != 0 && fv.CanSet() {
				fv.Set(dv)
			}
		case fv.Kind() == reflect.Slice:
			if dv.Len() != 0 && fv.CanSet() {
				fv.Set(reflect.AppendSlice(fv, dv)) // append defaults - then later checks can delete duplicates (e.g. attributes) easier
			}
		case fv.Kind() == reflect.Struct && fv.CanSet():
			setDefaults(dv.Interface(), fv.Addr().Interface())
		default:
			// ignore RemoveTypes and RemoveSamplers in managed entities and
			// groups - they are not "inherited"
		}
	}
}

// GetPlugin searches plugin for the first non-nil/non-empty field and
// returns it, using reflection
func GetPlugin(plugin *Plugin) interface{} {
	if plugin == nil {
		return nil
	}
	v := reflect.ValueOf(plugin).Elem()
	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i)
		if fv.Kind() == reflect.String {
			return fv.String()
		}
		if !fv.IsNil() {
			return fv.Interface()
		}
	}
	return nil
}

var dateRE = regexp.MustCompile(`<today\s*([+-]\d+)?([^>]*)?>`)

// ExpandFileDates substitutes Geneos formatted dates in the input for
// the time t and returns the result.
//
// The dates in the input are in the form "<today>" etc. as per the FKM
// path Date generation:
// <https://docs.itrsgroup.com/docs/geneos/current/collection/fkm-config/index.html#date-generation>
//
// Only <today...> is supported as there is no support (yet) for
// monitored days. The full format is <today[-N |+N ]FORMAT> where
// FORMAT is strftime-style patterns and spaces around the offsets are
// ignored but at least one is required between the offset and FORMAT if
// both are given. The strftime patterns are those from the defaults in
// the Go package <https://github.com/lestrrat-go/strftime>
func ExpandFileDates(in string, t time.Time) (out string, err error) {
	if !strings.Contains(in, "<") {
		out = in
		return
	}

	out = dateRE.ReplaceAllStringFunc(in, func(s string) (r string) {
		m := dateRE.FindStringSubmatch(s)
		if len(m) != 3 {
			return
		}
		// parsing error means zero offset
		offset, _ := strconv.Atoi(strings.TrimSpace(m[1]))
		t2 := t.AddDate(0, 0, offset)
		format := strings.TrimSpace(m[2])
		if format == "" {
			format = "%Y%m%d"
		}
		r, err = strftime.Format(format, t2)
		if err != nil {
			return ""
		}
		return
	})

	return
}
