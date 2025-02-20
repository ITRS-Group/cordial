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
		err := fmt.Errorf("sampler %q doesn't exist", s)
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
