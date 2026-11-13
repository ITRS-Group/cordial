package ftm

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	delayUnit := "SECONDS"
	sizeUnit := "KILOBYTES"
	plugin := Plugin{
		Files: []File{{
			Path: common.NewSingleLineString("/var/log/app.log"),
			MonitoringPeriod: &MonitoringPeriod{
				PeriodStart: common.NewValue("00:00"),
			},
			Alias: common.NewSingleLineString("application log"),
		}},
		MonitoringPeriod: &MonitoringPeriod{PeriodAlias: new("business-hours")},
		HolidayDates:     &Holidays{Holidays: []string{"1-Jan"}},
		Holidays:         &HolidaysVars{Holiday: []*common.SingleLineStringVar{common.NewSingleLineString("$(holiday)")}},
		MonitoredDays: &common.Weekdays{
			Monday: true, Tuesday: true, Wednesday: true, Thursday: true,
			Friday: true, Saturday: false, Sunday: false,
		},
		ShowPreviousDay:            common.NewValue(false),
		PreviousDayFile:            common.NewValue("previous.log"),
		UseChangedTime:             common.NewValue(true),
		Debug:                      common.NewValue(false),
		DelayUnit:                  &delayUnit,
		SizeUnit:                   &sizeUnit,
		ConsistentDateStamps:       common.NewValue(false),
		DisplayTimeInISO8601Format: common.NewValue(true),
		ShowActualFilename:         common.NewValue(false),
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<monitoringPeriod><periodAlias>business-hours</periodAlias></monitoringPeriod>",
		"<holidays><holiday>1-Jan</holiday></holidays>",
		"<showPreviousDay><data>false</data></showPreviousDay>",
		"<previousDayFile><data>previous.log</data></previousDayFile>",
		"<useChangedTime><data>true</data></useChangedTime>",
		"<debug><data>false</data></debug>",
		"<delayUnit>SECONDS</delayUnit>",
		"<sizeUnit>KILOBYTES</sizeUnit>",
		"<monitoredDays><monday>true</monday><tuesday>true</tuesday><wednesday>true</wednesday>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	if got.Files[0].MonitoringPeriod == nil || got.Files[0].MonitoringPeriod.PeriodStart == nil {
		t.Fatalf("file monitoring period did not round-trip: %#v", got.Files[0].MonitoringPeriod)
	}
	testhelpers.AssertStable(t, &got, data)
}

//go:fix inline
func stringPointer(value string) *string { return new(value) }
