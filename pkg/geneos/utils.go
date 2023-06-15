/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package geneos

import (
	"reflect"

	"github.com/rs/zerolog/log"
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

// FlattenProbes takes the top level geneos.Probes struct and returns a
// map of probe name to geneos.Probe objects while setting the Type
// field to the type of probe
func FlattenProbes(in *Probes) (probes map[string]Probe) {
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
		p, f, v := flattenProbeGroup(&g)
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

// flattenProbeGroup works through a geneos.ProbeGroup and applies
// defaults to all it's contents and returns three slices of probe types
func flattenProbeGroup(in *ProbeGroup) (probes []Probe, floatingProbes []FloatingProbe, virtualProbes []VirtualProbe) {
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
		p, f, v := flattenProbeGroup(&g)

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

// FlattenEntities func
func FlattenEntities(in *ManagedEntities, types map[string]Type) (entities map[string]ManagedEntity) {
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

	for _, childGroup := range in.ManagedEntityGroups {
		flattenedEntities := flattenEntityGroup(&childGroup, types)
		for _, entity := range flattenedEntities {
			if entity.Disabled {
				continue
			}

			// remove dups from the flattening process
			// entity.ManagedEntityInfo.Attributes = RemoveDuplicates(entity.ManagedEntityInfo.Attributes)
			// entity.ManagedEntityInfo.Vars = RemoveDuplicates(entity.ManagedEntityInfo.Vars)

			entities[entity.Name] = entity
		}
	}
	return
}

// flattenEntityGroup type
func flattenEntityGroup(group *ManagedEntityGroup, types map[string]Type) (entities map[string]ManagedEntity) {
	entities = make(map[string]ManagedEntity)

	if group.Disabled {
		return
	}

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

	for _, childGroup := range group.ManagedEntityGroups {
		setDefaults(group.ManagedEntityInfo, &childGroup.ManagedEntityInfo)
		// remove dups from merged slices in setDefaults
		group.ManagedEntityInfo.Attributes = RemoveDuplicates(group.ManagedEntityInfo.Attributes)
		group.ManagedEntityInfo.Vars = RemoveDuplicates(group.ManagedEntityInfo.Vars)

		resolveSamplersFromGroup(group.ManagedEntityInfo, &childGroup.ManagedEntityInfo, types)

		flattenedEntities := flattenEntityGroup(&childGroup, types)
		for _, entity := range flattenedEntities {
			if entity.Disabled {
				continue
			}
			entities[entity.Name] = entity
		}
	}

	return
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

func FlattenTypes(in *Types) (types map[string]Type) {
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
		t := flattenTypeGroup(&g)
		for _, t := range t {
			if t.Disabled {
				continue
			}
			types[t.Name] = t
		}
	}
	return
}

func flattenTypeGroup(in *TypeGroup) (types map[string]Type) {
	types = make(map[string]Type)
	for _, t := range in.Types {
		if t.Disabled {
			continue
		}
		types[t.Name] = t
	}
	for _, g := range in.TypeGroups {
		t := flattenTypeGroup(&g)
		for _, t := range t {
			if t.Disabled {
				continue
			}
			types[t.Name] = t
		}
	}
	return
}

func FlattenSamplers(in *Samplers) (samplers map[string]Sampler) {
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
		s := flattenSamplerGroup(&g)
		for _, s := range s {
			if s.Disabled {
				continue
			}
			samplers[s.Name] = s
		}
	}
	return
}

func flattenSamplerGroup(in *SamplerGroup) (samplers map[string]Sampler) {
	samplers = make(map[string]Sampler)
	for _, s := range in.Samplers {
		if s.Disabled {
			continue
		}
		samplers[s.Name] = s
	}
	for _, g := range in.SamplerGroups {
		s := flattenSamplerGroup(&g)
		for _, s := range s {
			if s.Disabled {
				continue
			}
			samplers[s.Name] = s
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
func setDefaults(from any, to any) {
	if reflect.TypeOf(to).Kind() != reflect.Pointer {
		log.Error().Msg("'to' not a pointer")
		return
	}
	sv := reflect.ValueOf(to).Elem()

	for i := 0; i < sv.NumField(); i++ {
		fv := sv.Field(i)
		fn := sv.Type().Field(i).Name
		dv := reflect.ValueOf(from).FieldByName(fn)

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
			// log.Debug().Msgf("unknown type %v", fv.Type())
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
