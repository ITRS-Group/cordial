/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import "github.com/itrs-group/cordial/pkg/xpath"

// SnoozeManual runs the internal command of the same name
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

// Unsnooze runs the internal command of the same name
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

// SnoozeInfo runs the internal command of the same name
func (c *Connection) SnoozeInfo(target *xpath.XPath) (crs []CommandResponse, err error) {
	crs, err = c.RunCommandAll("/SNOOZE:info", target)
	return
}
