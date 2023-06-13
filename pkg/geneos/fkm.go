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

import (
	"encoding/xml"
	"strings"
)

// FKM

type FKMPlugin struct {
	Display *FKMDisplay `xml:"fkm>display,omitempty" json:",omitempty" yaml:",omitempty"`
	Files   FKMFiles    `xml:"files,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"files"`
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
	Filename *SingleLineString `xml:"filename,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"filename"`
	Stream   *SingleLineString `xml:"stream,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"stream"`
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
	SearchString *SingleLineString `xml:"searchString"`
	Rules        FKMMatchRules     `xml:"rules,omitempty"`
}
