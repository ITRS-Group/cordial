package top

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/processes"
)

type Plugin struct {
	XMLName              xml.Name              `xml:"top" json:"-" yaml:"-"`
	ProcessParameters    *processes.Parameters `xml:"processParameters,omitempty"`
	Threshold            *Threshold            `xml:"threshold,omitempty"`
	TailOffPeriod        *int                  `xml:"tailOffPeriod,omitempty"`
	MaxRows              *int                  `xml:"maxRows,omitempty"`
	AdjustForLogicalCPUs *common.Value         `xml:"adjustForLogicalCPUs,omitempty"`
}

type Threshold struct {
	VarThresholdParameter *common.Value `xml:"var-thresholdParameter,omitempty"`
	ThresholdParameter    *string       `xml:"thresholdParameter,omitempty" geneos:"deprecated"`
	GreaterThanOrEqualTo  *float64      `xml:"greaterThanOrEqualTo,omitempty"`
	LessThanOrEqualTo     *float64      `xml:"lessThanOrEqualTo,omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }
