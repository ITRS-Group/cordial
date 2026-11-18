package adbcmd

import (
	"slices"
	"testing"
)

func TestUnmarshalLineToStruct(t *testing.T) {
	const severity = "com.itrsgroup.swing.activedashboardmanager.modifiers.DataItemSeverityNode~7.0~$0$~Severity~dataitem(/geneos)~/ Demo~2~f1~f2~$0$~false~192~192~192~188~240~188~255~207~80~255~116~116~"

	t.Run("severity colours above 127", func(t *testing.T) {
		var n dataItemSeverityNodeFields
		if err := unmarshalLineToStruct(severity, &n); err != nil {
			t.Fatal(err)
		}
		if n.WarningColourR != 255 || n.CriticalColourR != 255 || n.OKColourG != 240 {
			t.Errorf("colours = %+v", n)
		}
		if want := []string{"f1", "f2"}; !slices.Equal(n.Filters, want) {
			t.Errorf("filters = %v, want %v", n.Filters, want)
		}
	})

	t.Run("empty fields keep their position", func(t *testing.T) {
		var n dataItemExprNodeFields
		line := "com.x.DataItemExprNode~7.0~$0$~Value~dataitem(/geneos)~~1~/filter~?~false~"
		if err := unmarshalLineToStruct(line, &n); err != nil {
			t.Fatal(err)
		}
		if n.URLAlias != "" || n.FilterCount != 1 || !slices.Equal(n.Filters, []string{"/filter"}) || n.DefaultValue != "?" {
			t.Errorf("fields = %+v", n)
		}
	})

	for _, test := range []struct {
		name     string
		line     string
		severity bool
	}{
		{"short line", "com.x.DataItemExprNode~7.0~$0$", false},
		{"too few fields for the negative indexes", "a~7.0~b~c~d~e~0~", false},
		{"filter count larger than the line", "com.x.DataItemExprNode~7.0~$0$~Value~dataitem(/geneos)~alias~9~/filter~?~false~", false},
		{"colour out of range", "com.x.DataItemSeverityNode~7.0~$0$~Severity~t~a~0~$0$~false~300~192~192~188~240~188~255~207~80~255~116~116~", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var err error
			if test.severity {
				var n dataItemSeverityNodeFields
				err = unmarshalLineToStruct(test.line, &n)
			} else {
				var n dataItemExprNodeFields
				err = unmarshalLineToStruct(test.line, &n)
			}
			if err == nil {
				t.Error("no error for an invalid line")
			}
		})
	}
}
