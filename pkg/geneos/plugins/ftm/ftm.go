package ftm

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName                    xml.Name          `xml:"ftm" json:"-" yaml:"-"`
	Files                      []File            `xml:"files>file"`
	MonitoringPeriod           *MonitoringPeriod `xml:"monitoringPeriod,omitempty"`
	HolidayDates               *Holidays         `xml:"holidays,omitempty" geneos:"deprecated"`
	Holidays                   *HolidaysVars     `xml:"holidaysVar,omitempty"`
	MonitoredDays              *common.Weekdays  `xml:"monitoredDays,omitempty"`
	ConsistentDateStamps       *common.Value     `xml:"consistentDateStamps,omitempty"`
	DisplayTimeInISO8601Format *common.Value     `xml:"displayTimeInIso8601Format,omitempty"`
	ShowPreviousDay            *common.Value     `xml:"showPreviousDay,omitempty"`
	PreviousDayFile            *common.Value     `xml:"previousDayFile,omitempty"`
	ShowActualFilename         *common.Value     `xml:"showActualFilename,omitempty"`
	DelayUnit                  *string           `xml:"delayUnit,omitempty"`
	SizeUnit                   *string           `xml:"sizeUnit,omitempty"`
	UseChangedTime             *common.Value     `xml:"useChangedTime,omitempty"`
	Debug                      *common.Value     `xml:"debug,omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }

type File struct {
	XMLName         xml.Name                    `xml:"file" json:"-" yaml:"-"`
	Path            *common.SingleLineStringVar `xml:"path"`
	AdditionalPaths *AdditionalPaths            `xml:"additionalPaths,omitempty"`
	ExpectedArrival *common.Value               `xml:"expectedArrival,omitempty"`
	ExpectedPeriod  *struct {
		Period string `xml:",innerxml"`
	} `xml:"expectedPeriod,omitempty"`
	TZOffset         *common.Value               `xml:"tzOffset,omitempty"`
	MonitoringPeriod *MonitoringPeriod           `xml:"monitoringPeriod,omitempty"`
	Alias            *common.SingleLineStringVar `xml:"alias"`
}

type MonitoringPeriodAlias struct {
	Alias string `xml:"periodAlias"`
}

type MonitoringPeriodStart struct {
	PeriodStart *common.Value `xml:"periodStartTime,omitempty"`
}

type MonitoringPeriod struct {
	PeriodAlias *string       `xml:"periodAlias,omitempty"`
	PeriodStart *common.Value `xml:"periodStartTime,omitempty"`
}

type Holidays struct {
	Holidays []string `xml:"holiday,omitempty"`
}

type AdditionalPaths struct {
	Paths []*common.SingleLineStringVar `xml:"additionalPath"`
}

type HolidaysVars struct {
	XMLName xml.Name                      `xml:"holidaysVar" json:"-" yaml:"-"`
	Holiday []*common.SingleLineStringVar `xml:"holiday,omitempty"`
	Day     *common.Weekdays              `xml:"day,omitempty"`
}
