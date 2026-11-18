package statetracker

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName         xml.Name               `xml:"stateTracker" json:"-" yaml:"-"`
	Columns         *Columns               `xml:"columns,omitempty"`
	TemplateOptions *TemplateOptions       `xml:"templateOptions,omitempty"`
	Performance     *Performance           `xml:"performance,omitempty"`
	Behavioural     *Behavioural           `xml:"behavioural,omitempty"`
	TrackerGroup    []Group                `xml:"trackerGroup,omitempty"`
	Other           []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }

type Columns struct {
	RenameTrackerColumn *common.SingleLineStringVar   `xml:"renameTrackerColumn"`
	Columns             []*common.SingleLineStringVar `xml:"column"`
	Other               []common.RawXMLElement        `xml:",any" json:",omitempty" yaml:",omitempty"`
}

type TemplateOptions struct {
	HideTrackerIDs *common.Value          `xml:"hideTrackerIDs"`
	Other          []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

type Performance struct {
	SkipMidsampleUpdates   *common.Value          `xml:"skipMidSampleUpdates,omitempty"`
	MaxMessagesPerSample   *common.Value          `xml:"maxMessagesPerSample,omitempty"`
	MinimumReservedRowpace *common.Value          `xml:"minimumReservedRowspace,omitempty"`
	Other                  []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

type Behavioural struct {
	FileTruncateResetsTrackers *common.Value          `xml:"fileTruncateResetsTrackers,omitempty"`
	Other                      []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

type Group struct {
	Name        string                 `xml:"name,attr"`
	Columns     *Columns               `xml:"columns,omitempty"`
	Performance *Performance           `xml:"performance,omitempty"`
	Trackers    []*Tracker             `xml:"trackers>tracker,omitempty"`
	Other       []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the Group without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v Group) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

type Tracker struct {
	DeprecatedName   string                       `xml:"name,attr" geneos:"deprecated"`
	Name             *common.SingleLineStringVar  `xml:"name"`
	Filename         *common.SingleLineStringVar  `xml:"filename"`
	Stream           *common.SingleLineStringVar  `xml:"stream"`
	Rewind           *common.Value                `xml:"rewind"`
	Filters          []common.SingleLineStringVar `xml:"filters>filter,omitempty"`
	MessageTimestamp *common.Value                `xml:"messageTimestamp,omitempty"`
	Template         *common.Value                `xml:"template"`
	Other            []common.RawXMLElement       `xml:",any" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the Tracker without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v Tracker) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}
