package statetracker

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName         xml.Name         `xml:"stateTracker" json:"-" yaml:"-"`
	Columns         *Columns         `xml:"columns,omitempty"`
	TemplateOptions *TemplateOptions `xml:"templateOptions,omitempty"`
	Performance     *Performance     `xml:"performance,omitempty"`
	Behavioural     *Behavioural     `xml:"behavioural,omitempty"`
	TrackerGroup    []Group          `xml:"trackerGroup,omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }

type Columns struct {
	RenameTrackerColumn *common.SingleLineStringVar   `xml:"renameTrackerColumn"`
	Columns             []*common.SingleLineStringVar `xml:"column"`
}

type TemplateOptions struct {
	HideTrackerIDs *common.Value `xml:"hideTrackerIDs"`
}

type Performance struct {
	SkipMidsampleUpdates   *common.Value `xml:"skipMidSampleUpdates,omitempty"`
	MaxMessagesPerSample   *common.Value `xml:"maxMessagesPerSample,omitempty"`
	MinimumReservedRowpace *common.Value `xml:"minimumReservedRowspace,omitempty"`
}

type Behavioural struct {
	FileTruncateResetsTrackers *common.Value `xml:"fileTruncateResetsTrackers,omitempty"`
}

type Group struct {
	Name        string       `xml:"name,attr"`
	Columns     *Columns     `xml:"columns,omitempty"`
	Performance *Performance `xml:"performance,omitempty"`
	Trackers    []*Tracker   `xml:"trackers>tracker,omitempty"`
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
}
