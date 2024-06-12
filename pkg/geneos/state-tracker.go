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
