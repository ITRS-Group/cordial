package common

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"
)

// isEmptyXMLValue reports whether encoding/xml would omit the value
// for a field with the omitempty option, also treating a pointer to an
// empty slice or map as empty
func isEmptyXMLValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface:
		return v.IsNil()
	case reflect.Pointer:
		if v.IsNil() {
			return true
		}
		switch v.Elem().Kind() {
		case reflect.Array, reflect.Map, reflect.Slice:
			return v.Elem().Len() == 0
		}
	}
	return false
}

// MarshalOmitEmptyParents encodes the struct v, or the struct it points
// to, as encoding/xml would but without the empty parent elements that
// encoding/xml writes for a field tagged `xml:"parent>child,omitempty"`
// when the field is empty and so omitted. These are not wanted, and may
// not be valid, in Geneos configuration. A field that is a struct, not
// a pointer to one, with the omitempty option is omitted if it is zero.
//
// It is for use in the MarshalXML method of a struct that has such
// fields, as it encodes a copy of the struct that has no methods and so
// does not call that method again, but it must not be used for a struct
// that is embedded in another as the method would then be promoted.
func MarshalOmitEmptyParents(e *xml.Encoder, start xml.StartElement, v any) (err error) {
	rv := reflect.Indirect(reflect.ValueOf(v))
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("cannot encode %T, not a struct", v)
	}
	rt := rv.Type()

	// being a Marshaler the element name is not taken from the XMLName
	// field, as encoding/xml otherwise does, so do that here
	if f, ok := rt.FieldByName("XMLName"); ok && f.Type == reflect.TypeFor[xml.Name]() {
		if name, _, _ := strings.Cut(f.Tag.Get("xml"), ","); name != "" {
			if space, local, found := strings.Cut(name, " "); found {
				start.Name = xml.Name{Space: space, Local: local}
			} else {
				start.Name = xml.Name{Local: name}
			}
		} else if n := rv.FieldByIndex(f.Index).Interface().(xml.Name); n.Local != "" {
			start.Name = n
		}
	}

	// build an equivalent struct type, with the tags of empty wrapped
	// fields changed to exclude them
	fields := make([]reflect.StructField, 0, rt.NumField())
	index := make([]int, 0, rt.NumField())
	for i := range rt.NumField() {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		name, opts, _ := strings.Cut(f.Tag.Get("xml"), ",")
		if strings.Contains(","+opts+",", ",omitempty,") {
			// encoding/xml does not omit an empty struct, which is not
			// wanted for the elements that are only there if they have content
			if strings.Contains(name, ">") && isEmptyXMLValue(rv.Field(i)) || f.Type.Kind() == reflect.Struct && f.Type != reflect.TypeFor[xml.Name]() && f.Name != "XMLName" && rv.Field(i).IsZero() {
				f.Tag = `xml:"-"`
			}
		}
		fields = append(fields, f)
		index = append(index, i)
	}

	defer func() {
		// StructOf panics for embedded fields it cannot handle
		if r := recover(); r != nil {
			err = fmt.Errorf("cannot encode %s: %v", rt, r)
		}
	}()
	nv := reflect.New(reflect.StructOf(fields)).Elem()
	for n, i := range index {
		nv.Field(n).Set(rv.Field(i))
	}
	return e.EncodeElement(nv.Interface(), start)
}
