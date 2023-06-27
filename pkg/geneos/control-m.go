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

package geneos

import "encoding/xml"

type ControlMPlugin struct {
	XMLName   xml.Name           `xml:"control-m" json:"-" yaml:"-"`
	Host      *Host              `xml:"host"`
	Port      *Value             `xml:"port"`
	Username  *Value             `xml:"username"`
	Password  *Value             `xml:"password"`
	Timeout   *Value             `xml:"timeout,omitempty"`
	Dataviews []ControlMDataview `xml:"dataviews>dataview,omitempty"`
}

func (_ *ControlMPlugin) String() string {
	return "control-m"
}

type ControlMDataview struct {
	Name       string              `xml:"name"`
	Parameters []ControlMParameter `xml:"parameters,omitempty"`
	Columns    []string            `xml:"columns>column,omitempty"`
}

type ControlMParameter struct {
	Parameter string `xml:"parameter"`
	Criteria  string `xml:"criteria"`
}
