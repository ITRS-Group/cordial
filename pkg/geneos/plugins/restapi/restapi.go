package restapi

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName                 xml.Name      `xml:"rest-api" json:"-" yaml:"-"`
	BufferSize              *common.Value `xml:"bufferSize,omitempty"`
	RowLimit                *common.Value `xml:"rowLimit,omitempty"`
	VerifyClientCertificate *common.Value `xml:"verifyClientCertificate,omitempty"`
	ClientCACertificate     *common.Value `xml:"clientCACertificate,omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }
