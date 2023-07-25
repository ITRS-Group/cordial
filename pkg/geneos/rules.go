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

// Rules

type Rules struct {
	XMLName    xml.Name    `xml:"rules" json:"-" yaml:"-"`
	RuleGroups []RuleGroup `xml:"ruleGroup,omitempty" json:"rulegroup,omitempty"`
	Rules      []Rule      `xml:"rule,omitempty" json:"rule,omitempty"`
}

type RuleGroup struct {
	XMLName    xml.Name      `xml:"ruleGroup" json:"-" yaml:"-"`
	Name       string        `xml:"name,attr"`
	Defaults   []RuleDefault `xml:"default,omitempty"`
	Rules      []Rule        `xml:"rule,omitempty" json:"rule,omitempty"`
	RuleGroups []RuleGroup   `xml:"ruleGroup,omitempty" json:"rulegroup,omitempty"`
}

type RuleDefault struct {
	XMLName       xml.Name       `xml:"default,omitempty" json:"default,omitempty"`
	Name          string         `xml:"name,attr"`
	Contexts      []string       `xml:"rule>contexts>context,omitempty"`
	PriorityGroup *int           `xml:"priorityGroup,omitempty"`
	ActiveTime    *ActiveTimeRef `xml:"activeTime"`
}

type Rule struct {
	XMLName      xml.Name `xml:"rule" json:"-" yaml:"-"`
	Name         string   `xml:"name,attr"`
	Targets      []string `xml:"targets>target"`
	Priority     int      `xml:"priority"`
	Ifs          []interface{}
	Transactions []interface{}
}
