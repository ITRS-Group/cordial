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

import (
	"encoding/xml"
	"strings"
)

// FKM

type FKMPlugin struct {
	XMLName xml.Name    `xml:"fkm" json:"-" yaml:"-"`
	Display *FKMDisplay `xml:"display,omitempty" json:",omitempty" yaml:",omitempty"`
	Files   FKMFiles    `xml:"files,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"files"`
}

func (_ *FKMPlugin) String() string {
	return "fkm"
}

type FKMDisplay struct {
	TriggerMode string `xml:"triggerMode,omitempty" json:",omitempty" yaml:",omitempty"`
}

type FKMFiles struct {
	XMLName xml.Name  `xml:"files" json:"-" yaml:"-"`
	Files   []FKMFile `xml:"file,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"file"`
}

type FKMFile struct {
	XMLName             xml.Name       `xml:"file" json:"-" yaml:"-"`
	Source              *FKMFileSource `xml:"source"`
	Tables              []FKMTable     `xml:"tables>table,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"tables"`
	ClearTime           *Value         `xml:"clearTime,omitempty" json:",omitempty" yaml:",omitempty"`
	DefaultKeyClearTime *Value         `xml:"defaultKeyClearTime,omitempty" json:",omitempty" yaml:",omitempty"`
	Rewind              *Value         `xml:"rewind,omitempty" json:",omitempty" yaml:",omitempty"`
	Alias               *Value         `xml:"alias,omitempty" json:",omitempty" yaml:",omitempty"`
}

type FKMFileSource struct {
	Filename          *SingleLineStringVar `xml:"filename,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"filename"`
	Stream            *SingleLineStringVar `xml:"stream,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"stream"`
	NTEventLog        string               `xml:"ntEventLog,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"ntEventLog"`
	ProcessDescriptor *Reference           `xml:"processDescriptor,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"processDescriptor"`
}

type FKMTable struct {
	XMLName  xml.Name `xml:"table" json:"-" yaml:"-"`
	Severity string   `xml:"severity"`
	KeyTable *Value   `xml:"keyTable"`
}

type FKMKeyTable struct {
	XMLName xml.Name `xml:"fkmTable" json:"-" yaml:"-"`
	Name    string   `xml:"ref,attr"`
}

type FKMStaticKeyTable struct {
	XMLName xml.Name `xml:"fkmTable" json:"-" yaml:"-"`
	Name    string   `xml:"name,attr"`
	Keys    FKMKeys
}

type FKMKeyData struct {
	XMLName xml.Name `xml:"data" json:"-" yaml:"-"`
	Keys    FKMKeys
}

type FKMKeys struct {
	XMLName xml.Name      `xml:"keys" json:"-" yaml:"-"`
	Keys    []interface{} // should be FKMIgnoreKey or FKMKey
}

// Return an FKMKey struct with keys built from the parameters. The keys
// are interpreted as follows:
//
// * "=" prefixed string - force basic match (can contain embedded Geneos $(var))
// * "!" prefixed string - ignore key
// * "!=" or "=!" prefixed string - force basic ignore key
// * "/text[/]" - text will be treated as a regexp, trailing '/' optional
// * any occurrence of non-alpha character (ignoring '.') - treat as regexp, "!" means ignore key
// * "/i" - as a suffix of a regexp will force case insensitive matches
// * plain string (see below) - Basic match
func NewFKMKeys(keys ...string) (out FKMKeys) {
	for _, k := range keys {
		out = out.Append(k)
	}
	return
}

func (in FKMKeys) Append(key string) (out FKMKeys) {
	out = in
	rule := Basic

	switch {
	case strings.HasPrefix(key, "!="), strings.HasPrefix(key, "=!"):
		key = key[2:]
		k := FKMIgnoreKey{
			Match: FKMMatch{
				SearchString: NewSingleLineString(key),
				Rules:        rule,
			},
		}
		out.Keys = append(out.Keys, k)

	case strings.HasPrefix(key, "="):
		key = key[1:]
		k := FKMKey{
			SetKey: FKMSetKey{
				Match: &FKMMatch{
					SearchString: NewSingleLineString(key),
					Rules:        rule,
				},
			},
		}
		out.Keys = append(out.Keys, k)

	case strings.HasPrefix(key, "/"):
		key = key[1:]
		rule = Regexp
		if strings.HasSuffix(key, "/") {
			key = key[:len(key)-1]
		} else if strings.HasSuffix(key, "/i") {
			key = key[:len(key)-2]
			rule = RegexpIgnoreCase
		}
		k := FKMKey{
			SetKey: FKMSetKey{
				Match: &FKMMatch{
					SearchString: NewSingleLineString(key),
					Rules:        rule,
				},
			},
		}
		out.Keys = append(out.Keys, k)

	case strings.HasPrefix(key, "!"):
		key = key[1:]
		if strings.ContainsAny(key, `?+|*^${}()[]\`) {
			rule = Regexp
			if strings.HasSuffix(key, "/i") {
				key = key[:len(key)-2]
				rule = RegexpIgnoreCase
			}
		}
		k := FKMIgnoreKey{
			Match: FKMMatch{
				SearchString: NewSingleLineString(key),
				Rules:        rule,
			},
		}
		out.Keys = append(out.Keys, k)

	case strings.ContainsAny(key, `?+|*^${}()[]\`):
		rule = Regexp
		if strings.HasSuffix(key, "/i") {
			key = key[:len(key)-2]
			rule = RegexpIgnoreCase
		}
		k := FKMKey{
			SetKey: FKMSetKey{
				Match: &FKMMatch{
					SearchString: NewSingleLineString(key),
					Rules:        rule,
				},
			},
		}
		out.Keys = append(out.Keys, k)

	case strings.HasSuffix(key, "/i"):
		key = key[:len(key)-2]
		rule = RegexpIgnoreCase
		k := FKMKey{
			SetKey: FKMSetKey{
				Match: &FKMMatch{
					SearchString: NewSingleLineString(key),
					Rules:        rule,
				},
			},
		}
		out.Keys = append(out.Keys, k)

	default:
		k := FKMKey{
			SetKey: FKMSetKey{
				Match: &FKMMatch{
					SearchString: NewSingleLineString(key),
					Rules:        rule,
				},
			},
		}
		out.Keys = append(out.Keys, k)
	}

	return
}

type FKMIgnoreKey struct {
	XMLName xml.Name `xml:"ignoreKey" json:"-" yaml:"-"`
	Match   FKMMatch `xml:"match"`
	// ActiveTime
}

type FKMKey struct {
	XMLName  xml.Name  `xml:"key" json:"-" yaml:"-"`
	SetKey   FKMSetKey `xml:"setKey"`
	ClearKey *FKMMatch `xml:"clearKey,omitempty"`
	Message  *Value    `xml:"message,omitempty"`
	Severity string    `xml:"severity,omitempty"`
}

type FKMSetKey struct {
	Match        *FKMMatch `xml:"match,omitempty"`
	NotUpdatedIn *Value    `xml:"notUpdatedIn>timePeriodInSeconds,omitempty"`
	Updated      string    `xml:"updated,omitempty"`
}

type FKMMatchRules string

const (
	Basic            FKMMatchRules = "BASIC"
	Regexp           FKMMatchRules = "REGEXP"
	RegexpIgnoreCase FKMMatchRules = "REGEXP_IGNORE_CASE"
)

type FKMMatch struct {
	SearchString *SingleLineStringVar `xml:"searchString"`
	Rules        FKMMatchRules        `xml:"rules,omitempty"`
}
