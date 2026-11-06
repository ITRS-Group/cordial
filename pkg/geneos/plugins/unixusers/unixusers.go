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
}

func (p *Plugin) String() string { return p.XMLName.Local }
