package commands

import (
	"github.com/itrs-group/cordial/pkg/xpath"
)

func (c *Connection) SampleNow(target *xpath.XPath) (err error) {
	_, err = c.RunCommandAll("/PLUGIN:sampleNow", target)
	return
}

func (c *Connection) LastSampleInfo(target *xpath.XPath) (crs []CommandsResponse, err error) {
	return c.RunCommandAll("/PLUGIN:lastSampleInfo", target)
}
