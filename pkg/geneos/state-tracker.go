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

// StateTrackerPlugin TODO: Describe its purpose and intended use.
type StateTrackerPlugin struct {
	XMLName         xml.Name                     `xml:"stateTracker" json:"-" yaml:"-"`
	Columns         StateTrackerColumns          `xml:"columns"`
	TemplateOptions *StateTrackerTemplateOptions `xml:"templateOptions,omitempty"`
	Performance     *StateTrackerPerformance     `xml:"performance,omitempty"`
	TrackerGroup    []StateTrackerGroup          `xml:"trackerGroup,omitempty"`
}

// String TODO: Document its behavior, inputs, and effects.
func (p *StateTrackerPlugin) String() string {
	return p.XMLName.Local
}

// StateTrackerColumns TODO: Describe its purpose and intended use.
type StateTrackerColumns struct {
	RenameTrackerColumn *SingleLineStringVar   `xml:"renameTrackerColumn"`
	Columns             []*SingleLineStringVar `xml:"column"`
}

// StateTrackerTemplateOptions TODO: Describe its purpose and intended use.
type StateTrackerTemplateOptions struct {
	HideTrackerIDs *Value `xml:"hideTrackerIDs"`
}

// StateTrackerPerformance TODO: Describe its purpose and intended use.
type StateTrackerPerformance struct {
	SkipMidsampleUpdates   *Value `xml:"skipMidSampleUpdates,omitempty"`
	MaxMessagesPerSample   *Value `xml:"maxMessagesPerSample,omitempty"`
	MinimumReservedRowpace *Value `xml:"minimumReservedRowspace,omitempty"`
}

// StateTrackerGroup TODO: Describe its purpose and intended use.
type StateTrackerGroup struct {
	Name     string                 `xml:"name,attr"`
	Trackers []*StateTrackerTracker `xml:"trackers>tracker,omitempty"`
}

// StateTrackerTracker TODO: Describe its purpose and intended use.
type StateTrackerTracker struct {
	// XMLName        xml.Name             `xml:"tracker" json:"-" yaml:"-"`
	DeprecatedName string               `xml:"name,attr"`
	Name           *SingleLineStringVar `xml:"name"`
	Filename       *SingleLineStringVar `xml:"filename"`
	Stream         *SingleLineStringVar `xml:"stream"`
	Rewind         *Value               `xml:"rewind"`
	Template       *Value               `xml:"template"`
}
