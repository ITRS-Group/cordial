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

import (
	"github.com/itrs-group/cordial/pkg/xpath"
)

// SampleNow calls the internal command of the same name on the
// connection c and against the target XPath. An error is returned on
// failure to run the command.
func (c *Connection) SampleNow(target *xpath.XPath) (err error) {
	_, err = c.RunCommandAll("/PLUGIN:sampleNow", target)
	return
}

// LastSampleInfo calls the internal command of the same name and
// returns a slice of CommandResponse types, one for each matching data
// item, on the connection c. While errors from the Gateway are returned
// in the CommandResponse structures, if there is an error connecting or
// running the command then that is returned in err
func (c *Connection) LastSampleInfo(target *xpath.XPath) (crs []CommandResponse, err error) {
	return c.RunCommandAll("/PLUGIN:lastSampleInfo", target)
}
