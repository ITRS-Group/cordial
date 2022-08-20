package geneos

import (
	"encoding/xml"
	"strings"
)

// FKM

type FKMPlugin struct {
	Display *FKMDisplay `xml:"fkm>display,omitempty"`
	Files   []FKMFile   `xml:"fkm>files>file,omitempty"`
}

type FKMDisplay struct {
	TriggerMode string `xml:"triggerMode,omitempty"`
}

type FKMFile struct {
	XMLName             xml.Name          `xml:"file"`
	Filename            *SingleLineString `xml:"source>filename,omitempty"`
	Stream              *SingleLineString `xml:"source>stream,omitempty"`
	Tables              []FKMTable        `xml:"tables>table,omitempty"`
	ClearTime           *Value            `xml:"clearTime"`
	DefaultKeyClearTime *Value            `xml:"defaultKeyClearTime"`
	Rewind              *Value            `xml:"rewind"`
	Alias               *Value            `xml:"alias"`
}

type FKMTable struct {
	XMLName  xml.Name `xml:"table"`
	Severity string   `xml:"severity"`
	KeyTable *Value   `xml:"keyTable"`
}

type FKMKeyTable struct {
	XMLName xml.Name `xml:"fkmTable"`
	Name    string   `xml:"ref,attr"`
}

type FKMStaticKeyTable struct {
	XMLName xml.Name `xml:"fkmTable"`
	Name    string   `xml:"name,attr"`
	Keys    FKMKeys
}

type FKMKeyData struct {
	XMLName xml.Name `xml:"data"`
	Keys    FKMKeys
}

type FKMKeys struct {
	XMLName xml.Name      `xml:"keys"`
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
//
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
	XMLName xml.Name `xml:"ignoreKey"`
	Match   FKMMatch `xml:"match"`
	// ActiveTime
}

type FKMKey struct {
	XMLName  xml.Name  `xml:"key"`
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
