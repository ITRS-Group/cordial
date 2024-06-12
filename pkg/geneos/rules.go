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
	Disabled   bool          `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Defaults   []RuleDefault `xml:"default,omitempty"`
	Rules      []Rule        `xml:"rule,omitempty" json:"rule,omitempty"`
	RuleGroups []RuleGroup   `xml:"ruleGroup,omitempty" json:"rulegroup,omitempty"`
}

type RuleDefault struct {
	XMLName       xml.Name       `xml:"default,omitempty" json:"default,omitempty"`
	Name          string         `xml:"name,attr"`
	Contexts      []string       `xml:"rule>contexts>context,omitempty"`
	PriorityGroup int            `xml:"priorityGroup,omitempty" json:",omitempty" yaml:",omitempty"`
	ActiveTime    *ActiveTimeRef `xml:"activeTime" json:",omitempty" yaml:",omitempty"`
}

type Rule struct {
	XMLName       xml.Name      `xml:"rule" json:"-" yaml:"-"`
	Name          string        `xml:"name,attr"`
	Disabled      bool          `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Targets       []string      `xml:"targets>target"`
	PriorityGroup int           `xml:"priorityGroup,omitempty" json:",omitempty" yaml:",omitempty"`
	Priority      int           `xml:"priority"`
	Ifs           []interface{} `xml:"ifs,omitempty" json:",omitempty" yaml:",omitempty"`
	Transactions  []interface{} `xml:"tranactions,omitempty" json:",omitempty" yaml:",omitempty"`
}
