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
func FlattenProbes(in Probes) (probes map[string]Probe) {
	probes = make(map[string]Probe)

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
		p, f, v := flattenProbeGroup(g)
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
func flattenProbeGroup(in ProbeGroup) (probes []Probe, floatingProbes []FloatingProbe, virtualProbes []VirtualProbe) {
	if in.Disabled {
		return
	}

	for _, p := range in.Probes {
		if p.Disabled {
			continue
		}
		setDefaults(&p.ProbeInfo, in.ProbeInfo)
		probes = append(probes, p)

	}
	for _, f := range in.FloatingProbes {
		if f.Disabled {
			continue
		}
		setDefaults(&f.ProbeInfoWithoutPort, in.ProbeInfoWithoutPort)
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
		setDefaults(&g.ProbeInfo, in.ProbeInfo)
		p, f, v := flattenProbeGroup(g)

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
func FlattenEntities(in ManagedEntities) (entities map[string]ManagedEntity) {
	entities = make(map[string]ManagedEntity)
	for _, e := range in.Entities {
		if e.Disabled {
			continue
		}
		entities[e.Name] = e
	}

	for _, g := range in.ManagedEntityGroups {
		e := flattenEntityGroup(g)
		for _, e := range e {
			if e.Disabled {
				continue
			}
			e.ManagedEntityInfo.Attributes = RemoveDuplicates(e.ManagedEntityInfo.Attributes)
			e.ManagedEntityInfo.Vars = RemoveDuplicates(e.ManagedEntityInfo.Vars)
			entities[e.Name] = e
		}
	}
	return
}

// flattenEntityGroup type
func flattenEntityGroup(in ManagedEntityGroup) (entities map[string]ManagedEntity) {
	entities = make(map[string]ManagedEntity)

	if in.Disabled {
		return
	}

	for _, e := range in.Entities {
		if e.Disabled {
			continue
		}
		setDefaults(&e.ManagedEntityInfo, in.ManagedEntityInfo)
		entities[e.Name] = e
	}

	for _, g := range in.ManagedEntityGroups {
		setDefaults(&g.ManagedEntityInfo, in.ManagedEntityInfo)
		e := flattenEntityGroup(g)
		for _, e := range e {
			if e.Disabled {
				continue
			}
			entities[e.Name] = e
		}
	}

	return
}

func FlattenTypes(in Types) (types map[string]Type) {
	types = make(map[string]Type)
	for _, t := range in.Types {
		if t.Disabled {
			continue
		}
		types[t.Name] = t
	}
	for _, g := range in.TypeGroups {
		t := flattenTypeGroup(g)
		for _, t := range t {
			if t.Disabled {
				continue
			}
			types[t.Name] = t
		}
	}
	return
}

func flattenTypeGroup(in TypeGroup) (types map[string]Type) {
	types = make(map[string]Type)
	for _, t := range in.Types {
		if t.Disabled {
			continue
		}
		types[t.Name] = t
	}
	for _, g := range in.TypeGroups {
		t := flattenTypeGroup(g)
		for _, t := range t {
			if t.Disabled {
				continue
			}
			types[t.Name] = t
		}
	}
	return
}

func FlattenSamplers(in Samplers) (samplers map[string]Sampler) {
	samplers = make(map[string]Sampler)
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
		s := flattenSamplerGroup(g)
		for _, s := range s {
			if s.Disabled {
				continue
			}
			samplers[s.Name] = s
		}
	}
	return
}

func flattenSamplerGroup(in SamplerGroup) (samplers map[string]Sampler) {
	samplers = make(map[string]Sampler)
	for _, s := range in.Samplers {
		if s.Disabled {
			continue
		}
		samplers[s.Name] = s
	}
	for _, g := range in.SamplerGroups {
		s := flattenSamplerGroup(g)
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
func setDefaults(s any, defaults any) {
	if reflect.TypeOf(s).Kind() != reflect.Pointer {
		log.Error().Msg("not a pointer")
		return
	}
	sv := reflect.ValueOf(s).Elem()

	for i := 0; i < sv.NumField(); i++ {
		fv := sv.Field(i)
		fn := sv.Type().Field(i).Name
		dv := reflect.ValueOf(defaults).FieldByName(fn)

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
			setDefaults(fv.Addr().Interface(), dv.Interface())
		case fv.Type() == reflect.PointerTo(reflect.TypeOf(AddTypes{})) && fv.CanSet():
			if fv.IsNil() {
				// empty, just copy
				fv.Set(dv)
			} else if !dv.IsNil() {
				// check for something to copy
				tv := fv.Elem().FieldByName("Types")
				tv2 := dv.Elem().FieldByName("Types")
				tv.Set(reflect.AppendSlice(tv2, tv))
			}
		default:
			log.Debug().Msgf("unknown type %v", fv.Type())
		}
	}
}
