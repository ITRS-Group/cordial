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

func (s Sampler) String() string {
	return fmt.Sprintf("%s/%s.%s", s.URL(), s.EntityName(), s.SamplerName())
}

func (s *Sampler) IsValid() bool {
	logDebug.Print("called")
	res, err := s.samplerExists(s.EntityName(), s.SamplerName())
	if err != nil {
		logError.Print(err)
		return false
	}
	return res
}

// Getters only
// There are no setters as once created the struct should be immutable as
// otherwise it would not be safe in go routines. The structs get
// copied around a lot

// EntityName returns the Entuty name as a string
func (s Sampler) EntityName() string {
	return s.entityName
}

// SamplerName returns the Sampler name as a string
func (s Sampler) SamplerName() string {
	return s.samplerName
}

// Parameter - Get a parameter from the Geneos sampler config as a string
// It would not be difficult to add numeric and other type getters
func (s Sampler) Parameter(name string) (string, error) {
	logDebug.Print("called")

	if !s.IsValid() {
		err := fmt.Errorf("Parameter(): sampler doesn't exist")
		return "", err
	}
	return s.getParameter(s.EntityName(), s.SamplerName(), name)
}

// SignOn to the sampler with the interval given
func (s *Sampler) SignOn(interval time.Duration) error {
	return s.signOn(s.EntityName(), s.SamplerName(), int(interval.Seconds()))
}

// SignOff and cancel the heartbeat requirement for the sampler
func (s *Sampler) SignOff() error {
	return s.signOff(s.EntityName(), s.SamplerName())
}

// Heartbeat sends a heartbeat to reset the watchdog timer activated by
// SignOn
func (s Sampler) Heartbeat() error {
	return s.heartbeat(s.EntityName(), s.SamplerName())
}

/*
NewDataview - Create a new Dataview on the Sampler with an optional initial table of data.

If supplied the data is in the form of rows, each of which is a slice of
strings containing cell data. The first row must be the column names and the first
string in each row must be the rowname (including the first row of column names).

The underlying API appears to accept incomplete data so you can just send a row
of column names followed by each row only contains the first N columns each.
*/
func (s Sampler) NewDataview(dataviewName string, groupName string, args ...[]string) (d *Dataview, err error) {
	logDebug.Print("called")
	if !s.IsValid() {
		err = fmt.Errorf("NewDataview(): sampler doesn't exist")
		logError.Print(err)
		return
	}

	// try to remove it - failure shouldn't matter
	s.removeView(s.EntityName(), s.SamplerName(), dataviewName, groupName)

	d, err = s.CreateDataview(dataviewName, groupName)
	if err != nil {
		logError.Fatal(err)
		return
	}

	if len(args) > 0 {
		d.UpdateTable(args[0], args[1:]...)
	}
	return
}

// CreateDataview creates a new dataview struct and calls the API to create one on the
// Netprobe. It does NOT check for an existing dataview or remove it if one exists
func (s Sampler) CreateDataview(dataviewName string, groupName string) (d *Dataview, err error) {
	logDebug.Print("called")
	d = &Dataview{s, groupName + "-" + dataviewName}
	err = d.createView(s.EntityName(), s.SamplerName(), dataviewName, groupName)
	return
}
