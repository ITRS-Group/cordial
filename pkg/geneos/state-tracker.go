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

type StateTrackerPlugin struct {
	XMLName         xml.Name                     `xml:"stateTracker" json:"-" yaml:"-"`
	Columns         StateTrackerColumns          `xml:"columns"`
	TemplateOptions *StateTrackerTemplateOptions `xml:"templateOptions,omitempty"`
	Performance     *StateTrackerPerformance     `xml:"performance,omitempty"`
	TrackerGroup    []StateTrackerGroup          `xml:"trackerGroup,omitempty"`
}

func (p *StateTrackerPlugin) String() string {
	return p.XMLName.Local
}

type StateTrackerColumns struct {
	RenameTrackerColumn *SingleLineStringVar   `xml:"renameTrackerColumn"`
	Columns             []*SingleLineStringVar `xml:"column"`
}

type StateTrackerTemplateOptions struct {
	HideTrackerIDs *Value `xml:"hideTrackerIDs"`
}

type StateTrackerPerformance struct {
	SkipMidsampleUpdates   *Value `xml:"skipMidSampleUpdates,omitempty"`
	MaxMessagesPerSample   *Value `xml:"maxMessagesPerSample,omitempty"`
	MinimumReservedRowpace *Value `xml:"minimumReservedRowspace,omitempty"`
}

type StateTrackerGroup struct {
	Name     string                 `xml:"name,attr"`
	Trackers []*StateTrackerTracker `xml:"trackers>tracker,omitempty"`
}

type StateTrackerTracker struct {
	// XMLName        xml.Name             `xml:"tracker" json:"-" yaml:"-"`
	DeprecatedName string               `xml:"name,attr"`
	Name           *SingleLineStringVar `xml:"name"`
	Filename       *SingleLineStringVar `xml:"filename"`
	Stream         *SingleLineStringVar `xml:"stream"`
	Rewind         *Value               `xml:"rewind"`
	Template       *Value               `xml:"template"`
}
