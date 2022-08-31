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

package xmlrpc

import (
	"fmt"
	"time"
)

type Sampler struct {
	Client
	entityName  string
	samplerName string
}

func (s *Sampler) Exists() bool {
	res, err := s.samplerExists(s.entityName, s.samplerName)
	if err != nil {
		return false
	}
	return res
}

// String returns the Sampler name as a string
func (s Sampler) String() string {
	return s.samplerName
}

// Parameter - Get a parameter from the Geneos sampler config as a string
// It would not be difficult to add numeric and other type getters
func (s Sampler) Parameter(name string) (string, error) {
	if !s.Exists() {
		err := fmt.Errorf("Parameter(): sampler doesn't exist")
		return "", err
	}
	return s.getParameter(s.entityName, s.samplerName, name)
}

// SignOn to the sampler with the interval given
func (s *Sampler) SignOn(heartbeat time.Duration) error {
	return s.signOn(s.entityName, s.samplerName, int(heartbeat.Seconds()))
}

// SignOff and cancel the heartbeat requirement for the sampler
func (s *Sampler) SignOff() error {
	return s.signOff(s.entityName, s.samplerName)
}

// Heartbeat sends a heartbeat to reset the watchdog timer activated by
// SignOn
func (s Sampler) Heartbeat() error {
	return s.heartbeat(s.entityName, s.samplerName)
}

// Dataview returns a Dataview on the current Sampler.
func (s Sampler) Dataview(groupName string, viewName string) (d *Dataview) {
	if viewName == "" || groupName == "" || viewName == groupName {
		return
	}
	d = &Dataview{Sampler: s, viewName: viewName, groupName: groupName}
	return
}

/*
NewDataview - Create a new Dataview on the Sampler with an optional initial table of data.

If supplied the data is in the form of rows, each of which is a slice of
strings containing cell data. The first row must be the column names and the first
string in each row must be the rowname (including the first row of column names).

The underlying API appears to accept incomplete data so you can just send a row
of column names followed by each row only contains the first N columns each.
*/
func (s Sampler) NewDataview(groupName string, viewName string, args ...[]string) (d *Dataview, err error) {
	if !s.Exists() {
		err = fmt.Errorf("sampler %q does not exist", s)
		return
	}

	// try to remove it - failure shouldn't matter
	s.removeView(s.entityName, s.samplerName, viewName, groupName)

	if d = s.Dataview(groupName, viewName); d == nil {
		err = fmt.Errorf("dataview \"%s-%s\" not valid", groupName, viewName)
		return
	}
	err = d.createView(s.entityName, s.samplerName, viewName, groupName)
	if err != nil {
		return
	}

	if len(args) > 0 {
		d.UpdateTable(args[0], args[1:]...)
	}
	return
}
