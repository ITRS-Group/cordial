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

// Rules TODO: Describe its purpose and intended use.
type Rules struct {
	XMLName                   xml.Name           `xml:"rules" json:"-" yaml:"-"`
	Comment                   string             `xml:",comment"`
	StartupDelay              *RulesStartupDelay `xml:"startupDelay,omitempty"`
	HistoryPeriods            []RawXMLElement    `xml:"historyPeriod,omitempty"`
	ObcervTimeSeriesTemplates []RawXMLElement    `xml:"obcervTimeSeriesTemplate,omitempty"`
	RuleSets                  []RawXMLElement    `xml:"ruleset,omitempty" geneos:"deprecated"`
	RuleGroups                []RuleGroup        `xml:"ruleGroup,omitempty" json:"rulegroup,omitempty"`
	Rules                     []Rule             `xml:"rule,omitempty" json:"rule,omitempty"`
}

// RulesStartupDelay contains the required startup interval.
type RulesStartupDelay struct {
	Interval *int   `xml:"interval"`
	Comment  string `xml:",comment"`
}

// RuleGroup TODO: Describe its purpose and intended use.
type RuleGroup struct {
	XMLName    xml.Name      `xml:"ruleGroup" json:"-" yaml:"-"`
	Comment    string        `xml:",comment"`
	Name       string        `xml:"name,attr"`
	Disabled   *bool         `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Defaults   []RuleDefault `xml:"default,omitempty"`
	Rules      []Rule        `xml:"rule,omitempty" json:"rule,omitempty"`
	RuleGroups []RuleGroup   `xml:"ruleGroup,omitempty" json:"rulegroup,omitempty"`
}

// RuleDefault TODO: Describe its purpose and intended use.
type RuleDefault struct {
	XMLName       xml.Name        `xml:"default,omitempty" json:"default"`
	Comment       string          `xml:",comment"`
	Name          string          `xml:"name,attr"`
	Contexts      []string        `xml:"rule>contexts>context,omitempty"`
	PriorityGroup *int            `xml:"rule>priorityGroup,omitempty" json:",omitempty" yaml:",omitempty"`
	ActiveTime    *ActiveTimeRef  `xml:"rule>activeTime,omitempty" json:",omitempty" yaml:",omitempty"`
	ActiveTimes   []ActiveTimeRef `xml:"rule>activeTimes>activeTime,omitempty" geneos:"deprecated"`
	Transactions  []RawXMLElement `xml:"transaction,omitempty"`
}

// Rule TODO: Describe its purpose and intended use.
type Rule struct {
	XMLName                  xml.Name           `xml:"rule" json:"-" yaml:"-"`
	Comment                  string             `xml:",comment"`
	Name                     string             `xml:"name,attr"`
	Disabled                 *bool              `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Contexts                 []string           `xml:"contexts>context,omitempty"`
	Targets                  []string           `xml:"targets>target"`
	PriorityGroup            *int               `xml:"priorityGroup,omitempty" json:",omitempty" yaml:",omitempty"`
	Priority                 int                `xml:"priority"`
	ActiveTime               *ActiveTimeRef     `xml:"activeTime,omitempty"`
	ActiveTimes              []ActiveTimeRef    `xml:"activeTimes>activeTime,omitempty" geneos:"deprecated"`
	ActiveStateAffectsCell   *bool              `xml:"activeStateAffectsCell,omitempty"`
	StopFurtherEvaluation    *bool              `xml:"stopFurtherEvaluation,omitempty"`
	PathVariables            []RulePathVariable `xml:"pathVariables>pathVariable,omitempty"`
	PathAliases              []RawXMLElement    `xml:"pathAliases>pathAlias,omitempty"`
	EvaluateOnDataviewSample *bool              `xml:"evaluateOnDataviewSample,omitempty"`
	DisableRateWarning       *bool              `xml:"disableRateWarning,omitempty"`
	Block                    *RuleBlock         `xml:"block,omitempty"`
	Ifs                      []any              `xml:"-"`
	Transactions             []any              `xml:"transaction,omitempty" json:",omitempty" yaml:",omitempty"`
}

// RulePathVariable associates a name with an arbitrary Geneos value expression.
type RulePathVariable struct {
	Name    string        `xml:"name,attr"`
	Value   RawXMLElement `xml:"value"`
	Comment string        `xml:",comment"`
}

// RuleBlock preserves ordered statement elements defined by the XSD statement group.
type RuleBlock struct {
	Statements []RawXMLElement `xml:",any"`
	Comment    string          `xml:",comment"`
}
