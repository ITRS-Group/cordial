package fkm

import (
	"encoding/xml"
	"strings"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName                      xml.Name                    `xml:"fkm" json:"-" yaml:"-"`
	Display                      *Display                    `xml:"display,omitempty" json:",omitempty" yaml:",omitempty"`
	IsPCREMode                   *common.Value               `xml:"isPCREMode,omitempty"`
	MaxSamplesLagging            *common.Value               `xml:"maxSamplesLagging,omitempty"`
	MaxLinesProcessed            *common.Value               `xml:"maxLinesProcessed,omitempty"`
	ExcessiveUpdatesTimeout      *common.Value               `xml:"excessiveUpdatesTimeout,omitempty"`
	MaxConditionsPerSample       *common.Value               `xml:"maxConditionsPerSample,omitempty"`
	MaxConditionsPerFile         *common.Value               `xml:"maxConditionsPerFile,omitempty"`
	MaxConditionsPerKey          *common.Value               `xml:"maxConditionsPerKey,omitempty"`
	MaxConditionID               *common.Value               `xml:"maxConditionID,omitempty"`
	CheckIgnoreFiles             *common.Value               `xml:"checkIgnoreFiles,omitempty"`
	DefaultIgnoreFilePath        *common.SingleLineStringVar `xml:"defaultIgnoreFilePath,omitempty"`
	DefaultIgnoreFileName        *common.SingleLineStringVar `xml:"defaultIgnoreFileName,omitempty"`
	ContinueTriggerScan          *common.Value               `xml:"continueTriggerScan,omitempty"`
	SingleTriggerCleartimes      *common.Value               `xml:"singleTriggerCleartimes,omitempty"`
	WildcardMonitorAllMatches    *common.Value               `xml:"wildcardMonitorAllMatches,omitempty"`
	WildcardMatchTime            *string                     `xml:"wildcardMatchTime,omitempty"`
	Tags                         *Tags                       `xml:"tags,omitempty"`
	ParseTriggerDetails          *common.Value               `xml:"parseTriggerDetails,omitempty"`
	WrapViewFileNearTriggerLines *common.Value               `xml:"wrapViewFileNearTriggerLines,omitempty"`
	ReorderKeywordTables         *common.Value               `xml:"reorderKeywordTables,omitempty"`
	CheckInactiveClearKeys       *common.Value               `xml:"checkInactiveClearKeys,omitempty"`
	DefaultClearMode             *string                     `xml:"defaultClearMode,omitempty"`
	ExpandDatesInKeywords        *common.Value               `xml:"expandDatesInKeywords,omitempty"`
	MonitoredDays                *common.Weekdays            `xml:"monitoredDays,omitempty"`
	InternalTextEncoding         *common.SingleLineStringVar `xml:"internalTextEncoding,omitempty"`
	IgnoreIfNotInActiveTime      *common.Value               `xml:"ignoreIfNotInActiveTime,omitempty"`
	DefaultFileActiveTime        *common.Value               `xml:"defaultFileActiveTime,omitempty"`
	Files                        *Files                      `xml:"files,omitempty" mapstructure:"files"`
	ExtendedNTEventLogOutput     *common.Value               `xml:"extendedNTEventLogOutput,omitempty"`
	PersistentData               *PersistentData             `xml:"persistentData,omitempty"`
}

func (*Plugin) String() string { return "fkm" }

type Display struct {
	TriggerMode *string `xml:"triggerMode,omitempty" json:",omitempty" yaml:",omitempty"`
}

type Tags struct {
	StartTag *common.SingleLineStringVar `xml:"startTag,omitempty"`
	EndTag   *common.SingleLineStringVar `xml:"endTag,omitempty"`
}

type PersistentData struct {
	Enabled             *common.Value               `xml:"enabled,omitempty"`
	ExpireTimeInMinutes *common.Value               `xml:"expireTimeInMinutes,omitempty"`
	SaveLocation        *common.SingleLineStringVar `xml:"saveLocation,omitempty"`
}

type Files struct {
	XMLName xml.Name `xml:"files" json:"-" yaml:"-"`
	Files   []File   `xml:"file,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"file"`
}

type File struct {
	XMLName             xml.Name      `xml:"file" json:"-" yaml:"-"`
	Source              *FileSource   `xml:"source"`
	Tables              []Table       `xml:"tables>table,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"tables"`
	ClearTime           *common.Value `xml:"clearTime,omitempty" json:",omitempty" yaml:",omitempty"`
	DefaultKeyClearTime *common.Value `xml:"defaultKeyClearTime,omitempty" json:",omitempty" yaml:",omitempty"`
	Rewind              *common.Value `xml:"rewind,omitempty" json:",omitempty" yaml:",omitempty"`
	Alias               *common.Value `xml:"alias,omitempty" json:",omitempty" yaml:",omitempty"`
}

type FileSource struct {
	Filename          *common.SingleLineStringVar `xml:"filename,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"filename"`
	Stream            *common.SingleLineStringVar `xml:"stream,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"stream"`
	NTEventLog        string                      `xml:"ntEventLog,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"ntEventLog"`
	ProcessDescriptor *common.Reference           `xml:"processDescriptor,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"processDescriptor"`
}

type Table struct {
	XMLName  xml.Name      `xml:"table" json:"-" yaml:"-"`
	Severity string        `xml:"severity"`
	KeyTable *common.Value `xml:"keyTable"`
}

type KeyTable struct {
	XMLName xml.Name `xml:"fkmTable" json:"-" yaml:"-"`
	Name    string   `xml:"ref,attr"`
}

type StaticKeyTable struct {
	XMLName xml.Name `xml:"fkmTable" json:"-" yaml:"-"`
	Name    string   `xml:"name,attr"`
	Keys    Keys
}

type KeyData struct {
	XMLName xml.Name `xml:"data" json:"-" yaml:"-"`
	Keys    Keys
}

type Keys struct {
	XMLName xml.Name `xml:"keys" json:"-" yaml:"-"`
	Keys    []any
}

func NewKeys(keys ...string) (out Keys) {
	for _, key := range keys {
		out = out.Append(key)
	}
	return
}

func (in Keys) Append(key string) (out Keys) {
	out = in
	rule := Basic
	switch {
	case strings.HasPrefix(key, "!="), strings.HasPrefix(key, "=!"):
		key = key[2:]
		out.Keys = append(out.Keys, IgnoreKey{Match: Match{SearchString: common.NewSingleLineString(key), Rules: rule}})
	case strings.HasPrefix(key, "="):
		key = key[1:]
		out.Keys = append(out.Keys, Key{SetKey: SetKey{Match: &Match{SearchString: common.NewSingleLineString(key), Rules: rule}}})
	case strings.HasPrefix(key, "/"):
		key = key[1:]
		rule = Regexp
		if strings.HasSuffix(key, "/") {
			key = key[:len(key)-1]
		} else if strings.HasSuffix(key, "/i") {
			key = key[:len(key)-2]
			rule = RegexpIgnoreCase
		}
		out.Keys = append(out.Keys, Key{SetKey: SetKey{Match: &Match{SearchString: common.NewSingleLineString(key), Rules: rule}}})
	case strings.HasPrefix(key, "!"):
		key = key[1:]
		if strings.ContainsAny(key, `?+|*^${}()[]\`) {
			rule = Regexp
			if strings.HasSuffix(key, "/i") {
				key = key[:len(key)-2]
				rule = RegexpIgnoreCase
			}
		}
		out.Keys = append(out.Keys, IgnoreKey{Match: Match{SearchString: common.NewSingleLineString(key), Rules: rule}})
	case strings.ContainsAny(key, `?+|*^${}()[]\`):
		rule = Regexp
		if strings.HasSuffix(key, "/i") {
			key = key[:len(key)-2]
			rule = RegexpIgnoreCase
		}
		out.Keys = append(out.Keys, Key{SetKey: SetKey{Match: &Match{SearchString: common.NewSingleLineString(key), Rules: rule}}})
	case strings.HasSuffix(key, "/i"):
		key = key[:len(key)-2]
		rule = RegexpIgnoreCase
		out.Keys = append(out.Keys, Key{SetKey: SetKey{Match: &Match{SearchString: common.NewSingleLineString(key), Rules: rule}}})
	default:
		out.Keys = append(out.Keys, Key{SetKey: SetKey{Match: &Match{SearchString: common.NewSingleLineString(key), Rules: rule}}})
	}
	return
}

type IgnoreKey struct {
	XMLName xml.Name `xml:"ignoreKey" json:"-" yaml:"-"`
	Match   Match    `xml:"match"`
}

type Key struct {
	XMLName  xml.Name      `xml:"key" json:"-" yaml:"-"`
	SetKey   SetKey        `xml:"setKey"`
	ClearKey *Match        `xml:"clearKey,omitempty"`
	Message  *common.Value `xml:"message,omitempty"`
	Severity string        `xml:"severity,omitempty"`
}

type SetKey struct {
	Match        *Match        `xml:"match,omitempty"`
	NotUpdatedIn *common.Value `xml:"notUpdatedIn>timePeriodInSeconds,omitempty"`
	Updated      string        `xml:"updated,omitempty"`
}

type MatchRules string

const (
	Basic            MatchRules = "BASIC"
	Regexp           MatchRules = "REGEXP"
	RegexpIgnoreCase MatchRules = "REGEXP_IGNORE_CASE"
)

type Match struct {
	SearchString *common.SingleLineStringVar `xml:"searchString"`
	Rules        MatchRules                  `xml:"rules,omitempty"`
}
