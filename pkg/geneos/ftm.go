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
