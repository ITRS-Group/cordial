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

/*
Package to handle Geneos Gateway specific XPaths

These are a subset of the W3C XPath 1.0 standard and this package is
only interested in absolute paths and no matching is done. Geneos XPaths
are of a particular hierarchy and the ones we are interesting in here
are those used to communicate with the Gateway REST command API.

The two types of path handled are for headline or table cells, which
have the form:

/geneos/gateway/directory/probe/managedEntity/sampler/dataview/ ...

	... headlines/cell or ... rows/row/cell

Each component except "geneos", "directory", "headlines" and "rows" can
have a name and other predicates. The path can terminate at any level
that can carry a name. Apart from names, only attributes in
managedEntities are currently handled in anyway.
*/
package xpath

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

var ErrInvalidPath = errors.New("invalid Geneos XPath")
var ErrRelativePath = errors.New("unsupported relative Geneos XPath")

// XPath represents a Geneos Gateway XPath
//
// Each field (except Rows)  is a pointer, which if nil means the Xpath
// terminates at that point, not that the value is unset. The "Rows"
// boolean indicates in lower level components are rows or headlines
type XPath struct {
	Gateway  *Gateway  `json:"gateway,omitempty"`
	Probe    *Probe    `json:"probe,omitempty"`
	Entity   *Entity   `json:"entity,omitempty"`
	Sampler  *Sampler  `json:"sampler,omitempty"`
	Dataview *Dataview `json:"dataview,omitempty"`
	Rows     bool      `json:"-"`
	Headline *Headline `json:"headline,omitempty"`
	Row      *Row      `json:"row,omitempty"`
	Column   *Column   `json:"column,omitempty"`
}

type Gateway struct {
	Name string `json:"name,omitempty"`
}

type Probe struct {
	Name string `json:"name,omitempty"`
}

type Entity struct {
	Name       string            `json:"name,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type Sampler struct {
	Name string  `json:"name,omitempty"`
	Type *string `json:"type,omitempty"`
}

type Dataview struct {
	Name string `json:"name,omitempty"`
}

type Headline struct {
	Name string `json:"name,omitempty"`
}

type Row struct {
	Name string `json:"name,omitempty"`
}

type Column struct {
	Name string `json:"name,omitempty"`
}

// New returns an XPath to the level of the element passed, which can be
// populated with fields.
func New(element interface{}) *XPath {
	x := &XPath{}
	return x.ResolveTo(element)
}

// NewDataviewPath returns an xpath populated to the dataview, with name
// dv if no name is passed, create a wildcard dataview path
func NewDataviewPath(name string) (x *XPath) {
	x = New(&Dataview{Name: name})
	return
}

// NewTableCellPath returns an xpath populated to the table cell
// identifies by row and column
func NewTableCellPath(row, column string) (x *XPath) {
	x = New(&Column{Name: column})
	x.Rows = true
	x.Row = &Row{Name: row}
	return
}

// NewHeadlinePath returns an xpath populated to the headline cell,
// identified by headline
func NewHeadlinePath(name string) (x *XPath) {
	x = New(&Headline{Name: name})
	return
}

// ResolveTo will, given an element type, return a new XPath to that
// element, removing lower level elements or adding empty elements to
// the level required. If the XPath does not contain an element of the
// type given then use the argument (which can include populated
// fields), but if empty then any existing element will be left as-is
// and not cleaned.
//
// e.g.
//
//	x := x.ResolveTo(&Dataview{})
//	y := xpath.ResolveTo(&Headline{Name: "headlineName"})
func (x *XPath) ResolveTo(element interface{}) *XPath {
	if element == nil {
		return nil
	}

	// copy the xpath
	var nx XPath

	// skip through any pointers
	for reflect.ValueOf(element).Kind() == reflect.Ptr {
		element = reflect.Indirect(reflect.ValueOf(element)).Interface()
	}

	// set the element, remove others
	switch e := element.(type) {
	case Gateway:
		nx = XPath{
			Gateway: x.Gateway,
		}
		if nx.Gateway == nil {
			nx.Gateway = &e
		}
	case Probe:
		nx = XPath{
			Gateway: x.Gateway,
			Probe:   x.Probe,
		}
		if nx.Gateway == nil {
			nx.Gateway = &Gateway{}
		}
		if nx.Probe == nil {
			nx.Probe = &e
		}
	case Entity:
		nx = XPath{
			Gateway: x.Gateway,
			Probe:   x.Probe,
			Entity:  x.Entity,
		}
		if nx.Gateway == nil {
			nx.Gateway = &Gateway{}
		}
		if nx.Probe == nil {
			nx.Probe = &Probe{}
		}
		if nx.Entity == nil {
			if e.Attributes == nil {
				e.Attributes = make(map[string]string)
			}
			nx.Entity = &e
		}
	case Sampler:
		nx = XPath{
			Gateway: x.Gateway,
			Probe:   x.Probe,
			Entity:  x.Entity,
			Sampler: x.Sampler,
		}
		if nx.Gateway == nil {
			nx.Gateway = &Gateway{}
		}
		if nx.Probe == nil {
			nx.Probe = &Probe{}
		}
		if nx.Entity == nil {
			nx.Entity = &Entity{
				Attributes: make(map[string]string),
			}
		}
		if nx.Sampler == nil {
			nx.Sampler = &e
		}
	case Dataview:
		nx = XPath{
			Gateway:  x.Gateway,
			Probe:    x.Probe,
			Entity:   x.Entity,
			Sampler:  x.Sampler,
			Dataview: x.Dataview,
		}
		if nx.Gateway == nil {
			nx.Gateway = &Gateway{}
		}
		if nx.Probe == nil {
			nx.Probe = &Probe{}
		}
		if nx.Entity == nil {
			nx.Entity = &Entity{
				Attributes: make(map[string]string),
			}
		}
		if nx.Sampler == nil {
			nx.Sampler = &Sampler{}
		}
		if nx.Dataview == nil {
			nx.Dataview = &e
		}
	case Headline:
		nx = XPath{
			Gateway:  x.Gateway,
			Probe:    x.Probe,
			Entity:   x.Entity,
			Sampler:  x.Sampler,
			Dataview: x.Dataview,
			Rows:     false,
			Headline: x.Headline,
		}
		if nx.Gateway == nil {
			nx.Gateway = &Gateway{}
		}
		if nx.Probe == nil {
			nx.Probe = &Probe{}
		}
		if nx.Entity == nil {
			nx.Entity = &Entity{
				Attributes: make(map[string]string),
			}
		}
		if nx.Sampler == nil {
			nx.Sampler = &Sampler{}
		}
		if nx.Dataview == nil {
			nx.Dataview = &Dataview{}
		}
		if nx.Headline == nil {
			nx.Headline = &e
		}
	case Row:
		nx = XPath{
			Gateway:  x.Gateway,
			Probe:    x.Probe,
			Entity:   x.Entity,
			Sampler:  x.Sampler,
			Dataview: x.Dataview,
			Rows:     true,
			Row:      x.Row,
		}
		if nx.Gateway == nil {
			nx.Gateway = &Gateway{}
		}
		if nx.Probe == nil {
			nx.Probe = &Probe{}
		}
		if nx.Entity == nil {
			nx.Entity = &Entity{
				Attributes: make(map[string]string),
			}
		}
		if nx.Sampler == nil {
			nx.Sampler = &Sampler{}
		}
		if nx.Dataview == nil {
			nx.Dataview = &Dataview{}
		}
		if nx.Row == nil {
			nx.Row = &e
		}
	case Column:
		nx = XPath{
			Gateway:  x.Gateway,
			Probe:    x.Probe,
			Entity:   x.Entity,
			Sampler:  x.Sampler,
			Dataview: x.Dataview,
			Rows:     true,
			Row:      x.Row,
			Column:   x.Column,
		}
		if nx.Gateway == nil {
			nx.Gateway = &Gateway{}
		}
		if nx.Probe == nil {
			nx.Probe = &Probe{}
		}
		if nx.Entity == nil {
			nx.Entity = &Entity{
				Attributes: make(map[string]string),
			}
		}
		if nx.Sampler == nil {
			nx.Sampler = &Sampler{}
		}
		if nx.Dataview == nil {
			nx.Dataview = &Dataview{}
		}
		if nx.Row == nil {
			nx.Row = &Row{}
		}
		if nx.Column == nil {
			nx.Column = &e
		}
	}

	return &nx
}

// IsEmpty returns true is the XPath appears to be empty
func (x *XPath) IsEmpty() bool {
	return x.Gateway == nil
}

// SetGatewayName sets the Gateway name
//
// ?do we need setters? validation?
func (x *XPath) SetGatewayName(gateway string) {
	x.Gateway = &Gateway{Name: gateway}
}

// IsTableCell returns true if x represents a table cell
func (x *XPath) IsTableCell() bool {
	return x.Rows && x.Row != nil && x.Column != nil
}

func (x *XPath) IsHeadline() bool {
	return x.Headline != nil && x.Row == nil
}

func (x *XPath) IsDataview() bool {
	return x.Dataview != nil && x.Headline == nil && x.Row == nil
}

func (x *XPath) IsSampler() bool {
	return x.Sampler != nil && x.Dataview == nil
}

func (x *XPath) IsEntity() bool {
	return x.Entity != nil && x.Sampler == nil
}

func (x *XPath) IsProbe() bool {
	return x.Probe != nil && x.Entity == nil
}

func (x *XPath) IsGateway() bool {
	return x.Gateway != nil && x.Probe == nil
}

// return a string representation of an XPath
func (x *XPath) String() (p string) {
	if x.Gateway == nil {
		return
	}
	p += "/geneos/gateway"
	if x.Gateway.Name != "" {
		p += fmt.Sprintf("[(@name=%q)]", x.Gateway.Name)
	}
	p += "/directory"

	if x.Probe == nil {
		return
	}
	p += "/probe"
	if x.Probe.Name != "" {
		p += fmt.Sprintf("[(@name=%q)]", x.Probe.Name)
	}

	if x.Entity == nil {
		return
	}
	p += "/managedEntity"
	if x.Entity.Name != "" {
		p += fmt.Sprintf("[(@name=%q)]", x.Entity.Name)
	}
	for k, v := range x.Entity.Attributes {
		p += fmt.Sprintf("[(attr(%q)=%q)]", k, v)
	}

	if x.Sampler == nil {
		return
	}
	p += "/sampler"
	if x.Sampler.Name != "" {
		p += fmt.Sprintf("[(@name=%q)]", x.Sampler.Name)
	}
	if x.Sampler.Type != nil {
		p += fmt.Sprintf("[(@type=%q)]", *x.Sampler.Type)
	}

	if x.Dataview == nil {
		return
	}
	p += "/dataview"
	if x.Dataview.Name != "" {
		p += fmt.Sprintf("[(@name=%q)]", x.Dataview.Name)
	}

	if x.Rows {
		p += "/rows"
		if x.Row == nil {
			return
		}
		p += "/row"
		if x.Row.Name != "" {
			p += fmt.Sprintf("[(@name=%q)]", x.Row.Name)
		}
		if x.Column == nil {
			return
		}
		p += "/cell"
		if x.Column.Name != "" {
			p += fmt.Sprintf("[(@column=%q)]", x.Column.Name)
		}
	} else {
		if x.Headline == nil {
			return
		}
		p += "/headlines"
		if x.Headline.Name != "" {
			p += fmt.Sprintf("/cell[(@name=%q)]", x.Headline.Name)
		}
	}
	return
}

// Parse takes an absolute Geneos XPath and returns an XPath structure.
//
// A leading double slash, e.g. //probe[(@name="myprobe")], results in
// preceding levels being filled-in and further processing continuing
// from there. Because of the general purpose nature of the function
// only levels down to //rows and //headlines are supported. If you need
// a general path to a cell then you must use either //rows/row/cell or
// //headlines/cell to ensure the returned path uses the correct
// structure.
//
// Full wildcards, e.g. `//*`, are not supported as it is not possible
// to determine the terminating level.
//
// Support for predicates is limited. Currently all components
// understand "name", e.g. //probe[(@name="probeName")], while
// "managedEntity" supports multiple "attribute" predicates and table
// cells (under "rows/row/cell") support "column" (which is used instead
// of "name").
func Parse(s string) (xpath *XPath, err error) {
	xpath = &XPath{}

	parts := splitWithEscaping(s, '/', '\\')

	// if the path is relative, handle it differently
	if !strings.HasPrefix(s, "/") {
		err = ErrRelativePath
		return
	}

	// if the path is a wildcard, find the first level and prefix the parts[] with
	// the higher levels
	if strings.HasPrefix(s, "//") {
		f := strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) })
		if len(f) == 0 {
			err = ErrInvalidPath
			return
		}
		parts = parts[2:]
		switch f[0] {
		case "gateway":
			// simple case
			parts = []string{"", "geneos", "gateway"}
		case "rows", "headlines":
			parts = append([]string{"dataview"}, parts...)
			fallthrough
		case "dataview":
			parts = append([]string{"sampler"}, parts...)
			fallthrough
		case "sampler":
			parts = append([]string{"managedEntity"}, parts...)
			fallthrough
		case "managedEntity":
			parts = append([]string{"probe"}, parts...)
			fallthrough
		case "probe":
			parts = append([]string{"", "geneos", "gateway", "directory"}, parts...)

		}
	}

	// walk through path backwards, dropping through each case
	switch p := len(parts); p {
	case 11:
		// table cell, check @column
		if !strings.HasPrefix(parts[p-1], "cell") {
			err = ErrInvalidPath
			return
		}
		column, _ := getAttr(parts[p-1], "column")
		xpath.Column = &Column{Name: column}
		p--
		fallthrough
	case 10:
		if strings.HasPrefix(parts[p-1], "row") {
			xpath.Rows = true
			row, _ := getAttr(parts[p-1], "name")
			xpath.Row = &Row{Name: row}
		} else if strings.HasPrefix(parts[p-1], "cell") {
			headline, _ := getAttr(parts[p-1], "name")
			xpath.Headline = &Headline{Name: headline}
		} else {
			err = ErrInvalidPath
			return
		}
		p--
		fallthrough
	case 9:
		if strings.HasPrefix(parts[p-1], "rows") {
			// x.Dataview.Column = column
		} else if strings.HasPrefix(parts[p-1], "headlines") {
			//
		} else {
			err = ErrInvalidPath
			return
		}
		p--
		fallthrough
	case 8:
		// dataview
		if !strings.HasPrefix(parts[p-1], "dataview") {
			err = ErrInvalidPath
			return
		}
		dataview, _ := getAttr(parts[p-1], "name")
		xpath.Dataview = &Dataview{Name: dataview}
		p--
		fallthrough
	case 7:
		// sampler
		if !strings.HasPrefix(parts[p-1], "sampler") {
			err = ErrInvalidPath
			return
		}
		var tp *string
		if t, ok := getAttr(parts[p-1], "type"); ok {
			tp = &t
		}
		name, _ := getAttr(parts[p-1], "name")
		xpath.Sampler = &Sampler{
			Name: name,
			Type: tp,
		}
		p--
		fallthrough
	case 6:
		// entity
		if !strings.HasPrefix(parts[p-1], "managedEntity") {
			err = ErrInvalidPath
			return
		}
		entity, _ := getAttr(parts[p-1], "name")
		xpath.Entity = &Entity{
			Name:       entity,
			Attributes: getAttributes(parts[p-1]),
		}
		p--
		fallthrough
	case 5:
		// probe
		if !strings.HasPrefix(parts[p-1], "probe") {
			err = ErrInvalidPath
			return
		}
		probe, _ := getAttr(parts[p-1], "name")
		xpath.Probe = &Probe{Name: probe}
		p--
		fallthrough
	case 4:
		// directory
		if parts[p-1] != "directory" {
			err = ErrInvalidPath
			return
		}
		p--
		fallthrough
	case 3:
		// gateway
		if !strings.HasPrefix(parts[p-1], "gateway") {
			err = ErrInvalidPath
			return
		}
		gateway, _ := getAttr(parts[p-1], "name")
		xpath.Gateway = &Gateway{Name: gateway}
		p--
		fallthrough
	case 2:
		// geneos
		if parts[p-1] != "geneos" {
			err = ErrInvalidPath
			return
		}
		p--
		fallthrough
	case 1:
		// leading '/' stripped, must be empty
		if parts[p-1] != "" {
			err = ErrInvalidPath
			return
		}
		return
	default:
		err = ErrInvalidPath
		return
	}
}

// return Xpath as a string
func (x XPath) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

// return an xpath parsed from a string
func (x *XPath) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	nx, err := Parse(s)
	*x = *nx
	return
}

// split a string on separator (byte) except when separator is escaped
// by the escape byte given. typical usage is for when an xpath element
// has an escaped '/'
func splitWithEscaping(s string, separator, escape byte) []string {
	var token []byte
	var tokens []string
	for i := 0; i < len(s); i++ {
		if s[i] == separator {
			tokens = append(tokens, string(token))
			token = token[:0]
		} else if s[i] == escape && i+1 < len(s) {
			i++
			token = append(token, s[i])
		} else {
			token = append(token, s[i])
		}
	}
	tokens = append(tokens, string(token))
	return tokens
}

// extract the value of an attribute from the xpath component in the form
// [(@attr="value")] - this just uses a regexp and applies to validation
func getAttr(s string, attr string) (v string, ok bool) {
	attrRE := regexp.MustCompile(fmt.Sprintf(`\[\(@%s="(.*?)\\{0}?"\)\]`, attr))
	m := attrRE.FindStringSubmatch(s)
	if m == nil {
		return
	}
	if len(m) > 1 {
		v = m[1]
		ok = true
	}
	return
}

var attributeRE = regexp.MustCompile(`\[\(attr\("(.*?)\\{0}?"\)="(.*?)\\{0}?"\)\]`)

// extract the attributes from the xpath managedEntity component in the form
// [(attr("KEY")="value")] - this just uses a regexp and applies to validation
func getAttributes(s string) (attrs map[string]string) {
	attrs = map[string]string{}
	m := attributeRE.FindAllStringSubmatch(s, -1)
	for _, n := range m {
		if len(n) != 3 {
			continue
		}
		attrs[n[1]] = n[2]
	}
	return
}

// LookupValues returns a map of the components of x to their values,
// suitable for use as an Expand LookupTable. Only components of x that
// are set are added to the returned map and processing stops as soon as
// an XPath struct field is nil. The returned map is always initialised.
//
// The potential keys set are gateway, probe, entity, sampler, type,
// dataview, row, column, headline. Entity attributes in the XPath are
// ignored.
func (x *XPath) LookupValues() (lookup map[string]string) {
	lookup = map[string]string{}
	if x == nil || x.Gateway == nil {
		return
	}
	lookup["gateway"] = x.Gateway.Name

	if x.Probe == nil {
		return
	}
	lookup["probe"] = x.Probe.Name

	if x.Entity == nil {
		return
	}
	lookup["entity"] = x.Entity.Name

	if x.Sampler == nil {
		return
	}
	lookup["sampler"] = x.Sampler.Name

	if x.Sampler.Type != nil {
		lookup["type"] = *x.Sampler.Type
	}

	if x.Dataview == nil {
		return
	}
	lookup["dataview"] = x.Dataview.Name

	if x.Rows {
		if x.Row == nil {
			return
		}
		lookup["row"] = x.Row.Name

		if x.Column == nil {
			return
		}
		lookup["column"] = x.Column.Name
	} else {
		if x.Headline == nil {
			return
		}
		lookup["headline"] = x.Headline.Name
	}

	return
}
