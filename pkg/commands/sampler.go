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
