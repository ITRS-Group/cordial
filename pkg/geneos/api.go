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

// API Plugin

type APIPlugin struct {
	Parameters  []Parameter       `xml:"api>parameters>parameter"`
	SummaryView *SingleLineString `xml:"api>showSummaryView>always>viewName,omitempty"`
}

func (f *APIPlugin) String() string {
	return "api"
}

type Parameter struct {
	Name  string            `xml:"name"`
	Value *SingleLineString `xml:"value"`
}

// API Streams Plugin

type APIStreamsPlugin struct {
	XMLName    xml.Name `xml:"api-streams"`
	Streams    *Streams `xml:"api-streams>streams"`
	CreateView *Value   `xml:"api-streams>createView,omitempty"`
}

func (f *APIStreamsPlugin) String() string {
	return "api-streams"
}

type Streams struct {
	XMLName xml.Name            `xml:"streams" json:"-" yaml:"-"`
	Stream  []*SingleLineString `xml:"stream"`
}
