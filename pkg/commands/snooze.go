package commands

import "github.com/itrs-group/cordial/pkg/xpath"

// test commands to work out kinks in args and returns

func (c *Connection) SnoozeManual(target *xpath.XPath, info string) (err error) {
	if target.IsGateway() || target.IsProbe() || target.IsEntity() {
		_, err = c.RunCommandAll("/SNOOZE:manual", target, Arg(1, info))
		return
	}
	if target.IsSampler() || target.IsHeadline() || target.IsTableCell() || target.IsDataview() {
		_, err = c.RunCommandAll("/SNOOZE:manualAllMe", target, Arg(1, info), Arg(5, "this"))
	}
	return
}

func (c *Connection) Unsnooze(target *xpath.XPath, info string) (err error) {
	if target.IsGateway() || target.IsProbe() || target.IsEntity() {
		_, err = c.RunCommandAll("/SNOOZE:unsnooze", target, Arg(1, info))
		return
	}
	if target.Rows || target.Headline != nil || target.Sampler != nil {
		_, err = c.RunCommandAll("/SNOOZE:unsnoozeAllMe", target, Arg(1, "this"), Arg(2, info))
	}
	return
}

func (c *Connection) SnoozeInfo(target *xpath.XPath) (crs []CommandsResponse, err error) {
	crs, err = c.RunCommandAll("/SNOOZE:info", target)
	return
}
