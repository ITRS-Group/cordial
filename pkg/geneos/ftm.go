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

import "encoding/xml"

type FTMPlugin struct {
	XMLName                    xml.Name         `xml:"ftm" json:"-" yaml:"-"`
	Files                      []FTMFile        `xml:"files>file"`
	Holidays                   *FTMHolidaysVars `xml:"holidaysVar,omitempty"`
	MonitoredDays              *FTMWeekdays     `xml:"monitoredDays,omitempty"`
	ConsistentDateStamps       *Value           `xml:"consistentDateStamps,omitempty"`
	DisplayTimeInISO8601Format *Value           `xml:"displayTimeInIso8601Format,omitempty"`
	ShowActualFilename         *Value           `xml:"showActualFilename,omitempty"`
	DelayUnit                  string           `xml:"delayUnit"`
	SizeUnit                   string           `xml:"sizeUnit"`
}

func (p *FTMPlugin) String() string {
	return p.XMLName.Local
}

type FTMFile struct {
	XMLName         xml.Name             `xml:"file" json:"-" yaml:"-"`
	Path            *SingleLineStringVar `xml:"path"`
	AdditionalPaths *FTMAdditionalPaths  `xml:"additionalPaths,omitempty"`
	ExpectedArrival *Value               `xml:"expectedArrival,omitempty"`
	ExpectedPeriod  *struct {
		Period string `xml:",innerxml"`
	} `xml:"expectedPeriod,omitempty"`
	TZOffset         *Value               `xml:"tzOffset,omitempty"`
	MonitoringPeriod interface{}          `xml:"monitoringPeriod"`
	Alias            *SingleLineStringVar `xml:"alias"`
}

type MonitoringPeriodAlias struct {
	Alias string `xml:"periodAlias"`
}

type MonitoringPeriodStart struct {
	PeriodStart *Value `xml:"periodStartTime,omitempty"`
}

type FTMAdditionalPaths struct {
	Paths []*SingleLineStringVar `xml:"additionalPath"`
}

type FTMHolidaysVars struct {
	XMLName xml.Name               `xml:"holidaysVar" json:"-" yaml:"-"`
	Holiday []*SingleLineStringVar `xml:"holiday,omitempty"`
	Day     *FTMWeekdays           `xml:"day,omitempty"`
}

type FTMWeekdays struct {
	Monday    bool `xml:"monday"`
	Tuesday   bool `xml:"tuesday"`
	Wednesday bool `xml:"wednesday"`
	Thursday  bool `xml:"thursday"`
	Friday    bool `xml:"friday"`
	Saturday  bool `xml:"saturday"`
	Sunday    bool `xml:"sunday"`
}
