package unixusers

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName          xml.Name                    `xml:"unix-users" json:"-" yaml:"-"`
	UserListCommand  *common.SingleLineStringVar `xml:"userListCommand,omitempty"`
	AuthorisedUsers  []string                    `xml:"authorisedUsers>user,omitempty"`
	AuthorisedGroups []string                    `xml:"authorisedGroups>group,omitempty"`
	Other            []common.RawXMLElement      `xml:",any" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the Plugin without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v Plugin) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

func (p *Plugin) String() string { return p.XMLName.Local }
