/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
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
