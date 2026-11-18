package processes

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName                     xml.Name                    `xml:"processes" json:"-" yaml:"-"`
	VarDisplayMode              *common.Value               `xml:"var-displayMode,omitempty" json:",omitempty" yaml:",omitempty"`
	DisplayMode                 *common.Value               `xml:"displayMode,omitempty" json:",omitempty" yaml:",omitempty" geneos:"deprecated"`
	ProcessParameters           *Parameters                 `xml:"processParameters,omitempty" json:",omitempty" yaml:",omitempty"`
	AdjustForLogicalCPUs        *common.Value               `xml:"adjustForLogicalCPUs,omitempty" json:",omitempty" yaml:",omitempty"`
	AdjustForLogicalCPUsSummary *common.Value               `xml:"adjustForLogicalCPUsSummary,omitempty" json:",omitempty" yaml:",omitempty"`
	ShowProcessDetails          *common.Value               `xml:"showProcessDetails,omitempty" json:",omitempty" yaml:",omitempty"`
	WideProcessSupport          *common.Value               `xml:"wideProcessSupport,omitempty" json:",omitempty" yaml:",omitempty"`
	CacheWideProcessNames       *common.Value               `xml:"cacheWideProcessNames,omitempty" json:",omitempty" yaml:",omitempty"`
	PSInfo64                    *common.SingleLineStringVar `xml:"psinfo64,omitempty" json:",omitempty" yaml:",omitempty"`
	Processes                   []Process                   `xml:"processes>process,omitempty" json:",omitempty" yaml:",omitempty"`
	LookupUsernames             *common.Value               `xml:"lookupUsernames,omitempty" json:",omitempty" yaml:",omitempty"`
	ProcessFilter               *Filter                     `xml:"processFilter,omitempty" json:",omitempty" yaml:",omitempty"`
	MatchCase                   *common.Value               `xml:"matchCase,omitempty" json:",omitempty" yaml:",omitempty"`
	ArgsMaxLength               *common.Value               `xml:"argsMaxLength,omitempty" json:",omitempty" yaml:",omitempty"`
	UserCacheMaxAge             *common.Value               `xml:"userCacheMaxAge,omitempty" json:",omitempty" yaml:",omitempty"`
	Other                       []common.RawXMLElement      `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes             []xml.Attr                  `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the Plugin without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v Plugin) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

func (p *Plugin) String() string { return p.XMLName.Local }

type Parameters struct {
	Var             *common.Reference      `xml:"var,omitempty"`
	Data            *ParametersData        `xml:"data,omitempty"`
	Other           []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

type ParametersData struct {
	Parameters      []Parameter            `xml:"parameter,omitempty"`
	Other           []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

// Parameter is the name of a process parameter, as the text of the
// parameter element.
type Parameter struct {
	Name            string                 `xml:",chardata"`
	Other           []common.RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr             `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

type Filter struct {
	ParentProcess   *common.SingleLineStringVar  `xml:"parentProcess,omitempty"`
	UserNames       []common.SingleLineStringVar `xml:"userNames>userName,omitempty"`
	UserIDs         []common.Value               `xml:"userIDs>userID,omitempty"`
	Other           []common.RawXMLElement       `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes []xml.Attr                   `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the Filter without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v Filter) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

type Process struct {
	XMLName           xml.Name                  `xml:"process" json:"-" yaml:"-"`
	Data              *common.ProcessDescriptor `xml:"data,omitempty" json:",omitempty" yaml:",omitempty"`
	ProcessDescriptor *ProcessDescriptorRef     `xml:"processDescriptor,omitempty" json:",omitempty" yaml:",omitempty"`
	Other             []common.RawXMLElement    `xml:",any" json:",omitempty" yaml:",omitempty"`
	OtherAttributes   []xml.Attr                `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}

type ProcessDescriptorRef struct {
	Name string `xml:"ref,attr" json:"name" yaml:"name"`

	OtherAttributes []xml.Attr `xml:",any,attr" json:",omitempty" yaml:",omitempty"`
}
