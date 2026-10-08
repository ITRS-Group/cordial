package adbcmd

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// types based on classes in adb file

type knownClasses interface {
	mainClassFields | dataItemExprNodeFields | dataItemSeverityNodeFields
}

// com.itrsgroup.activeconsole.ui.components.ACActiveDashboard~11.0~1~false~2~activedashboardmanager~ActiveDashboard~false~true~false~2~

const mainClass = "com.itrsgroup.activeconsole.ui.components.ACActiveDashboard"

type mainClassFields struct {
	Class   string  `adb:"0"`
	Version float64 `adb:"1"`
	Name    string  `adb:"6"`
}

// com.itrsgroup.swing.activedashboardmanager.modifiers.DataItemExprNode~7.0~$0$~Value~dataitem(/geneos/gateway[(@name="Demo Gateway")]/directory/probe[(@name="localhost")]/managedEntity[(@name="localhost")]/sampler[(@name="CPU")][(@type="System")]/dataview[(@name="CPU")]/rows/row[(@name="Average_cpu")]/cell[(@column="percentUtilisation")])~/ Demo Gateway / localhost / localhost / CPU(type=System) / CPU / Average_cpu / percentUtilisation~1~/geneos/gateway/directory/probe/managedEntity/sampler/dataview//*~?~false~

const dataItemExprNodeClass = "com.itrsgroup.swing.activedashboardmanager.modifiers.DataItemExprNode"

type dataItemExprNodeFields struct {
	Class                string   `adb:"0"`
	Version              float64  `adb:"1"`
	Name                 string   `adb:"2"`
	Attribute            string   `adb:"3"`
	URLTarget            string   `adb:"4"`
	URLAlias             string   `adb:"5"`
	FilterCount          int      `adb:"6"`
	Filters              []string `adb:"7-"`
	DefaultValue         string   `adb:"-2"`
	UseOppositeBoolValue bool     `adb:"-1"`
}

// com.itrsgroup.swing.activedashboardmanager.modifiers.DataItemSeverityNode~7.0~$0$~Severity~dataitem(/geneos/gateway[(@name="Demo Gateway")]/directory/probe[(@name="localhost")]/managedEntity[(@name="localhost")]/sampler[(@name="CPU")][(@type="System")]/dataview[(@name="CPU")]/rows/row[(@name="Average_cpu")]/cell[(@column="percentUtilisation")])~/ Demo Gateway / localhost / localhost / CPU(type=System) / CPU / Average_cpu / percentUtilisation~0~$0$~false~192~192~192~188~240~188~255~207~80~255~116~116~

const dataItemSeverityNodeClass = "com.itrsgroup.swing.activedashboardmanager.modifiers.DataItemSeverityNode"

type dataItemSeverityNodeFields struct {
	Class                string   `adb:"0"`
	Version              float64  `adb:"1"`
	Name                 string   `adb:"2"`
	Attribute            string   `adb:"3"`
	URLTarget            string   `adb:"4"`
	URLAlias             string   `adb:"5"`
	FilterCount          int      `adb:"6"`
	Filters              []string `adb:"7-"`
	DefaultValue         string   `adb:"-14"`
	UseOppositeBoolValue bool     `adb:"-13"`
	UndefinedColourR     int8     `adb:"-12"`
	UndefinedColourG     int8     `adb:"-11"`
	UndefinedColourB     int8     `adb:"-10"`
	OKColourR            int8     `adb:"-9"`
	OKColourG            int8     `adb:"-8"`
	OKColourB            int8     `adb:"-7"`
	WarningColourR       int8     `adb:"-6"`
	WarningColourG       int8     `adb:"-5"`
	WarningColourB       int8     `adb:"-4"`
	CriticalColourR      int8     `adb:"-3"`
	CriticalColourG      int8     `adb:"-2"`
	CriticalColourB      int8     `adb:"-1"`
}

// The field layout includes a line without a class defnition; this is
// an expression/modification indicator for the Modification class,
// e.g.:
//
// breaking an example down and indenting it for readability:
//
//  com.itrsgroup.swing.activedashboardmanager.modifiers.Modifier~3.0~User Assigned Transparency~$0$~2~User Assigned Transparency~false~
//
//     com.itrsgroup.swing.activedashboardmanager.modifiers.Modification~2.0~Fill Transparency~1~
//     true~true~
//         com.itrsgroup.swing.activedashboardmanager.modifiers.LogicExprNode~7.0~$0$~203~2~
//         com.itrsgroup.swing.activedashboardmanager.modifiers.DataItemExprNode~7.0~$0$~UserAssigned~droppedItem()~droppedItem~1~/geneos/gateway/directory//*~$0$~false~
//         com.itrsgroup.swing.activedashboardmanager.modifiers.ValueExprNode~7.0~$0$~true~false~java.lang.String~true~
//         com.itrsgroup.swing.activedashboardmanager.modifiers.ValueExprNode~7.0~$0$~Fill Transparency~false~java.lang.String~1.0~
//
//     com.itrsgroup.swing.activedashboardmanager.modifiers.Modification~2.0~Fill Transparency~1~
//     true~true~
//         com.itrsgroup.swing.activedashboardmanager.modifiers.LogicExprNode~7.0~$0$~203~2~
//         com.itrsgroup.swing.activedashboardmanager.modifiers.DataItemExprNode~7.0~$0$~UserAssigned~droppedItem()~droppedItem~1~/geneos/gateway/directory//*~$0$~false~
//         com.itrsgroup.swing.activedashboardmanager.modifiers.ValueExprNode~7.0~$0$~false~false~java.lang.String~false~
//         com.itrsgroup.swing.activedashboardmanager.modifiers.ValueExprNode~7.0~$0$~Fill Transparency~false~java.lang.String~0.4~
//

// the `true~true~` indicates expression == true and modification is
// true. The `203` on the following line is an `OP_EQUAL` operator,
// guessing this checks if the dropped item equals true.
//
// typical url modifiers are `false~true~`

// unmarshalLineToStruct takes a line of a dashboard file and a pointer
// to a struct of a known class type, and populates the struct fields
// based on the adb tags in the struct definition. It returns an error
// if any field cannot be parsed or if the adb tag is invalid.
//
// In the struct the `adb` tag negative values means "from the end of
// the fields", to allow for data structures that have arbitrary size. A
// tag like `adb:"7-"` indicates a starting point and field specific
// logic needs to be used to derive the actual field indexes
func unmarshalLineToStruct[V knownClasses](line string, v *V) error {
	rv := reflect.ValueOf(v).Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("expected a pointer to a struct, got %T", v)
	}

	rt := rv.Type()

	fields := strings.FieldsFunc(line, func(r rune) bool {
		return r == fieldSeparator
	})

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get("adb")
		if tag == "" {
			continue
		}

		parts := strings.Split(tag, ",")

		fieldSpec := parts[0]
		omitempty := slices.Contains(parts[1:], "omitempty")

		fieldRange := false
		if strings.HasSuffix(fieldSpec, "-") {
			fieldRange = true
			fieldSpec = strings.TrimSuffix(fieldSpec, "-")
		}

		idx, err := strconv.Atoi(fieldSpec)
		if err != nil {
			return fmt.Errorf("invalid adb tag %q on field %s: %v", fieldSpec, field.Name, err)
		}
		if idx >= len(fields) {
			if omitempty {
				continue
			}
			return fmt.Errorf("field index %d out of range for fields length %d", idx, len(fields))
		}

		// negative index indicates counting from the end of the fields slice
		if idx < 0 {
			idx = len(fields) + idx
		}

		if fieldRange {
			// handle field range logic here if needed
			if rt.Field(i).Name == "Filters" {
				if _, ok := rt.FieldByName("FilterCount"); ok {
					filterCount := rv.FieldByName("FilterCount").Int()
					for j := 0; j < int(filterCount); j++ {
						// handle each filter here
						if rv.Field(i).Kind() != reflect.Slice {
							return fmt.Errorf("expected string kind for Filters field, got %s", rv.Field(i).Kind())
						}
						rv.Field(i).Set(reflect.Append(rv.Field(i), reflect.ValueOf(fields[idx+j])))
					}
				}
			}
			continue
		}

		fieldValue := fields[idx]
		if omitempty && fieldValue == "" {
			continue
		}
		switch rv.Field(i).Kind() {
		case reflect.String:
			rv.Field(i).SetString(fieldValue)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal, err := strconv.ParseInt(fieldValue, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse int for field %s: %v", field.Name, err)
			}
			rv.Field(i).SetInt(intVal)
		case reflect.Float32, reflect.Float64:
			floatVal, err := strconv.ParseFloat(fieldValue, 64)
			if err != nil {
				return fmt.Errorf("failed to parse float for field %s: %v", field.Name, err)
			}
			rv.Field(i).SetFloat(floatVal)
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(fieldValue)
			if err != nil {
				return fmt.Errorf("failed to parse bool for field %s: %v", field.Name, err)
			}
			rv.Field(i).SetBool(boolVal)
		default:
			return fmt.Errorf("unsupported field type %s for field %s", rv.Field(i).Kind(), field.Name)
		}
	}
	return nil
}
