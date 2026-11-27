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
that can carry a name. Names, managed-entity attributes, sampler types,
and cell columns are interpreted structurally. Other predicates are
retained verbatim for transmission to and from a Gateway.
*/
package xpath

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// ErrInvalidPath TODO: Describe its purpose and intended use.
var ErrInvalidPath = errors.New("invalid Geneos XPath")

// ErrRelativePath TODO: Describe its purpose and intended use.
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

	// headlines is set by Parse for a path that ends at ".../headlines",
	// so that the suffix is kept when the path is formatted again
	headlines bool

	// predicates retains predicates not represented by the structured
	// component fields so they can be passed back to the Gateway.
	predicates map[string][]string
}

// Elements TODO: Describe its purpose and intended use.
type Elements interface {
	Gateway | Probe | Entity | Sampler | Dataview | Headline | Row | Column
	xpathElement()
}

type element interface {
	xpathElement()
}

func (Gateway) xpathElement()  {}
func (Probe) xpathElement()    {}
func (Entity) xpathElement()   {}
func (Sampler) xpathElement()  {}
func (Dataview) xpathElement() {}
func (Headline) xpathElement() {}
func (Row) xpathElement()      {}
func (Column) xpathElement()   {}

// Gateway TODO: Describe its purpose and intended use.
type Gateway struct {
	Name string `json:"name,omitempty"`
}

// Probe TODO: Describe its purpose and intended use.
type Probe struct {
	Name string `json:"name,omitempty"`
}

// Entity TODO: Describe its purpose and intended use.
type Entity struct {
	Name       string            `json:"name,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Sampler TODO: Describe its purpose and intended use.
type Sampler struct {
	Name string  `json:"name,omitempty"`
	Type *string `json:"type,omitempty"`
}

// Dataview TODO: Describe its purpose and intended use.
type Dataview struct {
	Name string `json:"name,omitempty"`
}

// Headline TODO: Describe its purpose and intended use.
type Headline struct {
	Name string `json:"name,omitempty"`
}

// Row TODO: Describe its purpose and intended use.
type Row struct {
	Name string `json:"name,omitempty"`
}

// Column TODO: Describe its purpose and intended use.
type Column struct {
	Name string `json:"name,omitempty"`
}

// New returns an XPath to the level of the element passed, which can be
// populated with fields.
func New[E Elements](element *E) *XPath {
	x := &XPath{}
	if r := x.ResolveTo(*element); r != nil {
		return r
	}
	return x
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

// Clone returns a deep copy of x, sharing no elements, attribute maps
// or sampler types with the original. A nil XPath returns nil.
func (x *XPath) Clone() *XPath {
	if x == nil {
		return nil
	}
	nx := *x
	if x.Gateway != nil {
		v := *x.Gateway
		nx.Gateway = &v
	}
	if x.Probe != nil {
		v := *x.Probe
		nx.Probe = &v
	}
	if x.Entity != nil {
		v := *x.Entity
		v.Attributes = maps.Clone(x.Entity.Attributes)
		nx.Entity = &v
	}
	if x.Sampler != nil {
		v := *x.Sampler
		if x.Sampler.Type != nil {
			t := *x.Sampler.Type
			v.Type = &t
		}
		nx.Sampler = &v
	}
	if x.Dataview != nil {
		v := *x.Dataview
		nx.Dataview = &v
	}
	if x.Headline != nil {
		v := *x.Headline
		nx.Headline = &v
	}
	if x.Row != nil {
		v := *x.Row
		nx.Row = &v
	}
	if x.Column != nil {
		v := *x.Column
		nx.Column = &v
	}
	nx.predicates = clonePredicates(x.predicates)
	return &nx
}

func clonePredicates(predicates map[string][]string) map[string][]string {
	if predicates == nil {
		return nil
	}
	clone := make(map[string][]string, len(predicates))
	for component, values := range predicates {
		clone[component] = slices.Clone(values)
	}
	return clone
}

func selectPredicates(predicates map[string][]string, components ...string) map[string][]string {
	var out map[string][]string
	for _, component := range components {
		if values, ok := predicates[component]; ok {
			if out == nil {
				out = make(map[string][]string, len(components))
			}
			out[component] = slices.Clone(values)
		}
	}
	return out
}

// cloneAttributes returns a copy of attrs, never nil
func cloneAttributes(attrs map[string]string) map[string]string {
	if attrs == nil {
		return make(map[string]string)
	}
	return maps.Clone(attrs)
}

// ResolveTo will, given an element type, return a new XPath to that
// element, which shares nothing with x or the element passed, removing lower level elements or adding empty elements to
// the level required. If the XPath does not contain an element of the
// type given then use the argument (which can include populated
// fields), but if empty then any existing element will be left as-is
// and not cleaned.
//
// e.g.
//
//	x := x.ResolveTo(&Dataview{})
//	y := xpath.ResolveTo(&Headline{Name: "headlineName"})
func (x *XPath) ResolveTo(element element) *XPath {
	switch e := element.(type) {
	case Gateway, Probe, Entity, Sampler, Dataview, Headline, Row, Column:
		// already a value
	case *Gateway:
		if e == nil {
			return nil
		}
		element = *e
	case *Probe:
		if e == nil {
			return nil
		}
		element = *e
	case *Entity:
		if e == nil {
			return nil
		}
		element = *e
	case *Sampler:
		if e == nil {
			return nil
		}
		element = *e
	case *Dataview:
		if e == nil {
			return nil
		}
		element = *e
	case *Headline:
		if e == nil {
			return nil
		}
		element = *e
	case *Row:
		if e == nil {
			return nil
		}
		element = *e
	case *Column:
		if e == nil {
			return nil
		}
		element = *e
	default:
		return nil
	}

	// work on a deep copy so that the returned XPath never shares
	// elements with the receiver or the element argument
	c := x.Clone()
	if c == nil {
		c = &XPath{}
	}
	var nx XPath

	// set the element, remove others
	switch e := element.(type) {
	case Gateway:
		nx = XPath{
			Gateway: c.Gateway,
		}
		if nx.Gateway == nil {
			nx.Gateway = &e
		}
		nx.predicates = selectPredicates(c.predicates, "gateway")
	case Probe:
		nx = XPath{
			Gateway: c.Gateway,
			Probe:   c.Probe,
		}
		if nx.Gateway == nil {
			nx.Gateway = &Gateway{}
		}
		if nx.Probe == nil {
			nx.Probe = &e
		}
		nx.predicates = selectPredicates(c.predicates, "gateway", "probe")
	case Entity:
		nx = XPath{
			Gateway: c.Gateway,
			Probe:   c.Probe,
			Entity:  c.Entity,
		}
		if nx.Gateway == nil {
			nx.Gateway = &Gateway{}
		}
		if nx.Probe == nil {
			nx.Probe = &Probe{}
		}
		if nx.Entity == nil {
			e.Attributes = cloneAttributes(e.Attributes)
			nx.Entity = &e
		}
		nx.predicates = selectPredicates(c.predicates, "gateway", "probe", "entity")
	case Sampler:
		nx = XPath{
			Gateway: c.Gateway,
			Probe:   c.Probe,
			Entity:  c.Entity,
			Sampler: c.Sampler,
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
			if e.Type != nil {
				t := *e.Type
				e.Type = &t
			}
			nx.Sampler = &e
		}
		nx.predicates = selectPredicates(c.predicates, "gateway", "probe", "entity", "sampler")
	case Dataview:
		nx = XPath{
			Gateway:  c.Gateway,
			Probe:    c.Probe,
			Entity:   c.Entity,
			Sampler:  c.Sampler,
			Dataview: c.Dataview,
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
		nx.predicates = selectPredicates(c.predicates, "gateway", "probe", "entity", "sampler", "dataview")
	case Headline:
		nx = XPath{
			Gateway:  c.Gateway,
			Probe:    c.Probe,
			Entity:   c.Entity,
			Sampler:  c.Sampler,
			Dataview: c.Dataview,
			Rows:     false,
			Headline: c.Headline,
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
		nx.predicates = selectPredicates(c.predicates, "gateway", "probe", "entity", "sampler", "dataview", "headline")
	case Row:
		nx = XPath{
			Gateway:  c.Gateway,
			Probe:    c.Probe,
			Entity:   c.Entity,
			Sampler:  c.Sampler,
			Dataview: c.Dataview,
			Rows:     true,
			Row:      c.Row,
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
		nx.predicates = selectPredicates(c.predicates, "gateway", "probe", "entity", "sampler", "dataview", "row")
	case Column:
		nx = XPath{
			Gateway:  c.Gateway,
			Probe:    c.Probe,
			Entity:   c.Entity,
			Sampler:  c.Sampler,
			Dataview: c.Dataview,
			Rows:     true,
			Row:      c.Row,
			Column:   c.Column,
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
		nx.predicates = selectPredicates(c.predicates, "gateway", "probe", "entity", "sampler", "dataview", "row", "column")
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

// IsHeadline TODO: Document its behavior, inputs, and effects.
func (x *XPath) IsHeadline() bool {
	return x.Headline != nil && x.Row == nil
}

// IsDataview TODO: Document its behavior, inputs, and effects.
func (x *XPath) IsDataview() bool {
	return x.Dataview != nil && x.Headline == nil && x.Row == nil
}

// IsSampler TODO: Document its behavior, inputs, and effects.
func (x *XPath) IsSampler() bool {
	return x.Sampler != nil && x.Dataview == nil
}

// IsEntity TODO: Document its behavior, inputs, and effects.
func (x *XPath) IsEntity() bool {
	return x.Entity != nil && x.Sampler == nil
}

// IsProbe TODO: Document its behavior, inputs, and effects.
func (x *XPath) IsProbe() bool {
	return x.Probe != nil && x.Entity == nil
}

// IsGateway TODO: Document its behavior, inputs, and effects.
func (x *XPath) IsGateway() bool {
	return x.Gateway != nil && x.Probe == nil
}

// writeComponent appends a path component with an optional name
// predicate
func writeComponent(b *strings.Builder, component, attr, value string, predicates ...string) {
	b.WriteString(component)
	if value != "" {
		fmt.Fprintf(b, "[(@%s=%q)]", attr, value)
	}
	for _, predicate := range predicates {
		b.WriteString(predicate)
	}
}

// String returns the XPath formatted as a Geneos Gateway XPath string,
// down to the level that x terminates at. Names are quoted so that the
// result can be passed to [Parse] to get an equivalent XPath, whatever
// characters, including '/' and '"', the names contain.
func (x *XPath) String() string {
	if x == nil || x.Gateway == nil {
		return ""
	}
	var p strings.Builder
	writeComponent(&p, "/geneos/gateway", "name", x.Gateway.Name, x.predicates["gateway"]...)
	p.WriteString("/directory")

	if x.Probe == nil {
		return p.String()
	}
	writeComponent(&p, "/probe", "name", x.Probe.Name, x.predicates["probe"]...)

	if x.Entity == nil {
		return p.String()
	}
	writeComponent(&p, "/managedEntity", "name", x.Entity.Name)
	for _, name := range slices.Sorted(maps.Keys(x.Entity.Attributes)) {
		fmt.Fprintf(&p, "[(attr(%q)=%q)]", name, x.Entity.Attributes[name])
	}
	for _, predicate := range x.predicates["entity"] {
		p.WriteString(predicate)
	}

	if x.Sampler == nil {
		return p.String()
	}
	writeComponent(&p, "/sampler", "name", x.Sampler.Name)
	if x.Sampler.Type != nil {
		fmt.Fprintf(&p, "[(@type=%q)]", *x.Sampler.Type)
	}
	for _, predicate := range x.predicates["sampler"] {
		p.WriteString(predicate)
	}

	if x.Dataview == nil {
		return p.String()
	}
	writeComponent(&p, "/dataview", "name", x.Dataview.Name, x.predicates["dataview"]...)

	if x.Rows {
		p.WriteString("/rows")
		if x.Row == nil {
			return p.String()
		}
		writeComponent(&p, "/row", "name", x.Row.Name, x.predicates["row"]...)
		if x.Column == nil {
			return p.String()
		}
		writeComponent(&p, "/cell", "column", x.Column.Name, x.predicates["column"]...)
	} else {
		if x.Headline == nil {
			if x.headlines {
				p.WriteString("/headlines")
			}
			return p.String()
		}
		p.WriteString("/headlines")
		writeComponent(&p, "/cell", "name", x.Headline.Name, x.predicates["headline"]...)
	}
	return p.String()
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
// Common predicates are represented by structured component fields:
// all components understand "name", managedEntity supports multiple
// "attr" predicates, sampler supports "type", and table cells support
// "column". Other predicates are retained verbatim and included when the
// XPath is formatted or marshalled, so callers can pass expressions
// supported by the Gateway without requiring this package to interpret
// them.
func Parse(s string) (xpath *XPath, err error) {
	xpath = &XPath{}

	parts := splitPath(s)

	// if the path is relative, handle it differently
	if !strings.HasPrefix(s, "/") {
		err = ErrRelativePath
		return
	}

	// if the path is a wildcard, find the first level and prefix the parts[] with
	// the higher levels
	if strings.HasPrefix(s, "//") {
		var first string
		for field := range strings.FieldsFuncSeq(s, func(r rune) bool { return !unicode.IsLetter(r) }) {
			first = field
			break
		}
		if first == "" {
			err = ErrInvalidPath
			return
		}
		parts = parts[2:]
		switch first {
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
			parts = append([]string{"gateway", "directory"}, parts...)
			fallthrough
		case "gateway":
			// simple case
			parts = append([]string{"", "geneos"}, parts...)

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
		xpath.setPredicates("column", parts[p-1])
		p--
		fallthrough
	case 10:
		if strings.HasPrefix(parts[p-1], "row") {
			xpath.Rows = true
			row, _ := getAttr(parts[p-1], "name")
			xpath.Row = &Row{Name: row}
			xpath.setPredicates("row", parts[p-1])
		} else if strings.HasPrefix(parts[p-1], "cell") {
			headline, _ := getAttr(parts[p-1], "name")
			xpath.Headline = &Headline{Name: headline}
			xpath.setPredicates("headline", parts[p-1])
		} else {
			err = ErrInvalidPath
			return
		}
		p--
		fallthrough
	case 9:
		// a path that ends at rows or headlines is kept as such
		if strings.HasPrefix(parts[p-1], "rows") {
			xpath.Rows = true
		} else if strings.HasPrefix(parts[p-1], "headlines") {
			xpath.headlines = xpath.Headline == nil
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
		xpath.setPredicates("dataview", parts[p-1])
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
		xpath.setPredicates("sampler", parts[p-1])
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
		xpath.setPredicates("entity", parts[p-1])
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
		xpath.setPredicates("probe", parts[p-1])
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
		xpath.setPredicates("gateway", parts[p-1])
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

func (x *XPath) setPredicates(component, pathComponent string) {
	for _, raw := range splitPredicates(pathComponent) {
		_, predicates := parsePredicates(raw)
		if len(predicates) != 1 || !supportedPredicate(component, predicates[0]) {
			if x.predicates == nil {
				x.predicates = make(map[string][]string)
			}
			x.predicates[component] = append(x.predicates[component], raw)
		}
	}
}

func supportedPredicate(component string, p predicate) bool {
	switch component {
	case "entity":
		return (!p.attr && p.name == "name") || p.attr
	case "sampler":
		return !p.attr && (p.name == "name" || p.name == "type")
	case "column":
		return !p.attr && p.name == "column"
	default:
		return !p.attr && p.name == "name"
	}
}

func splitPredicates(component string) (predicates []string) {
	for i := 0; i < len(component); {
		if component[i] != '[' {
			i++
			continue
		}
		start := i
		depth := 0
		var quote byte
		for ; i < len(component); i++ {
			switch component[i] {
			case '\\':
				i++
			case '"', '\'':
				if quote == 0 {
					quote = component[i]
				} else if quote == component[i] {
					quote = 0
				}
			case '[':
				if quote == 0 {
					depth++
				}
			case ']':
				if quote == 0 {
					depth--
					if depth == 0 {
						i++
						predicates = append(predicates, component[start:i])
						goto next
					}
				}
			}
		}
		predicates = append(predicates, component[start:])
	next:
	}
	return
}

// matchName reports whether name matches the glob pattern. Unlike
// [path.Match], a '*' or '?' also matches '/', as names such as the
// rows of a DISK dataview ("/dev/sda1") commonly contain it. A pattern
// that is not valid never matches.
func matchName(pattern, name string) bool {
	const sub = "\x00"
	match, err := path.Match(strings.ReplaceAll(pattern, "/", sub), strings.ReplaceAll(name, "/", sub))
	return err == nil && match
}

// Match checks if the given XPath x matches the filter XPath. It
// applies glob patterns for names, as described for [path.Match] except
// that wildcards also match '/' in names, and treats nil elements and
// empty names in the filter as wildcards. A filter for an entity can be
// only attributes and a filter for a sampler can be only a type. A
// sampler without a type is matched as if it had an empty type.
func (x *XPath) Match(filter *XPath) bool {
	if x == nil || filter == nil {
		return false
	}

	// for each element of x, check if the same element in filter would
	// match. empty (nil) elements in filter mean wildcard. empty elements
	// in x must match exactly. attributes and types have to be treated
	// specially

	if filter.Gateway == nil {
		return true
	}
	if filter.Gateway.Name != "" {
		// if there's a filter, the element must exist and match
		if x.Gateway == nil || !matchName(filter.Gateway.Name, x.Gateway.Name) {
			return false
		}
	}

	if filter.Probe == nil {
		return true
	}
	if filter.Probe.Name != "" {
		if x.Probe == nil || !matchName(filter.Probe.Name, x.Probe.Name) {
			return false
		}
	}

	if filter.Entity == nil {
		return true
	}
	if filter.Entity.Name != "" || len(filter.Entity.Attributes) > 0 {
		if x.Entity == nil {
			return false
		}
		if filter.Entity.Name != "" && !matchName(filter.Entity.Name, x.Entity.Name) {
			return false
		}
		for k, v := range filter.Entity.Attributes {
			value, ok := x.Entity.Attributes[k]
			if !ok || !matchName(v, value) {
				return false
			}
		}
	}

	if filter.Sampler == nil {
		return true
	}
	var filterType string
	if filter.Sampler.Type != nil {
		filterType = *filter.Sampler.Type
	}
	if filter.Sampler.Name != "" || filterType != "" {
		if x.Sampler == nil {
			return false
		}
		if filter.Sampler.Name != "" && !matchName(filter.Sampler.Name, x.Sampler.Name) {
			return false
		}
		if filterType != "" {
			var samplerType string
			if x.Sampler.Type != nil {
				samplerType = *x.Sampler.Type
			}
			if !matchName(filterType, samplerType) {
				return false
			}
		}
	}

	if filter.Dataview == nil {
		return true
	}
	if filter.Dataview.Name != "" {
		if x.Dataview == nil || !matchName(filter.Dataview.Name, x.Dataview.Name) {
			return false
		}
	}

	// filter down to one of these, then check row / headline etc.
	if filter.Headline == nil && filter.Column == nil && filter.Row == nil {
		return true
	}

	if filter.Rows != x.Rows {
		return false
	}

	if filter.Headline != nil && filter.Headline.Name != "" {
		if x.Headline == nil || !matchName(filter.Headline.Name, x.Headline.Name) {
			return false
		}
	}

	if filter.Column != nil && filter.Column.Name != "" {
		if x.Column == nil || !matchName(filter.Column.Name, x.Column.Name) {
			return false
		}
	}

	if filter.Row != nil && filter.Row.Name != "" {
		if x.Row == nil || !matchName(filter.Row.Name, x.Row.Name) {
			return false
		}
	}

	// if we get this far, they match
	return true
}

// MarshalJSON returns the XPath as a JSON string.
func (x XPath) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

// UnmarshalJSON sets x from a JSON string containing an XPath. A JSON
// null leaves x unchanged and an empty string results in an empty
// XPath, which is what MarshalJSON produces for one, so that all valid
// values round-trip. If the string cannot be parsed then x is not
// changed.
func (x *XPath) UnmarshalJSON(b []byte) (err error) {
	if string(b) == "null" {
		return nil
	}
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	if s == "" {
		*x = XPath{}
		return nil
	}
	nx, err := Parse(s)
	if err != nil {
		return
	}
	*x = *nx
	return
}

// splitPath splits an XPath on the '/' separating its components,
// leaving out any that are inside a double-quoted string or are escaped
// with a backslash, such as the one in a name like "/dev/sda1". The
// components returned are unchanged so that quoting and escapes can be
// processed by the predicate parser.
func splitPath(s string) (parts []string) {
	var start int
	var quote byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++ // skip the escaped character
		case '"', '\'':
			if quote == 0 {
				quote = s[i]
			} else if quote == s[i] {
				quote = 0
			}
		case '/':
			if quote == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}

// a predicate is one [(@name="value")] or [(attr("name")="value")]
// following the name of a path component
type predicate struct {
	attr  bool // true for attr("name"), false for @name
	name  string
	value string
}

// parseQuoted parses the double-quoted string starting at s[i] and
// returns its unquoted value and the index after the closing quote.
// Strings as written by [XPath.String] use the same escapes as Go but
// as other tools may be less strict, any other backslash sequence
// stands for the escaped character.
func parseQuoted(s string, i int) (value string, next int, ok bool) {
	if i >= len(s) || s[i] != '"' {
		return
	}
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case '"':
			raw := s[i : j+1]
			if v, err := strconv.Unquote(raw); err == nil {
				return v, j + 1, true
			}
			var b strings.Builder
			for k := i + 1; k < j; k++ {
				if s[k] == '\\' && k+1 < j {
					k++
				}
				b.WriteByte(s[k])
			}
			return b.String(), j + 1, true
		}
	}
	return
}

// parsePredicates splits a path component into its element name and
// structurally supported predicates. Parsing stops at the first
// predicate that is not in an expected form; callers retain that and
// subsequent predicates verbatim for Gateway round trips.
func parsePredicates(s string) (element string, preds []predicate) {
	// find the end of the element name, which cannot be quoted
	i := strings.IndexByte(s, '[')
	if i < 0 {
		return s, nil
	}
	element = s[:i]

	for strings.HasPrefix(s[i:], "[(") {
		i += 2
		var p predicate
		switch {
		case strings.HasPrefix(s[i:], "@"):
			i++
			j := i
			for j < len(s) && s[j] != '=' {
				j++
			}
			p.name = s[i:j]
			i = j
		case strings.HasPrefix(s[i:], `attr(`):
			name, next, ok := parseQuoted(s, i+len("attr("))
			if !ok || !strings.HasPrefix(s[next:], ")") {
				return
			}
			p.attr, p.name = true, name
			i = next + 1
		default:
			return
		}
		if !strings.HasPrefix(s[i:], "=") {
			return
		}
		value, next, ok := parseQuoted(s, i+1)
		if !ok || !strings.HasPrefix(s[next:], ")]") {
			return
		}
		p.value = value
		preds = append(preds, p)
		i = next + 2
	}
	return
}

// extract the value of the first attribute from the xpath component in
// the form [(@attr="value")]
func getAttr(s string, attr string) (v string, ok bool) {
	for _, raw := range splitPredicates(s) {
		_, preds := parsePredicates(raw)
		for _, p := range preds {
			if !p.attr && p.name == attr {
				return p.value, true
			}
		}
	}
	return
}

// extract the attributes from the xpath managedEntity component in the form
// [(attr("KEY")="value")]
func getAttributes(s string) (attrs map[string]string) {
	attrs = map[string]string{}
	for _, raw := range splitPredicates(s) {
		_, preds := parsePredicates(raw)
		for _, p := range preds {
			if p.attr {
				attrs[p.name] = p.value
			}
		}
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
