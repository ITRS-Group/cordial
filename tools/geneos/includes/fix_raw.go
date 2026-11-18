/*
Copyright © 2026 ITRS Group

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

package includes

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html/charset"
)

// The Gateway setup is edited as text, replacing only the content of the
// `location` elements of includes, so that every other byte of the file
// is unchanged. The order of sibling elements in a setup is significant
// and the Go model does not keep comments, unknown elements,
// whitespace or the encoding declaration, so it cannot be used to
// write a setup back.

var xmlDeclEncoding = regexp.MustCompile(`^\s*<\?xml[^>]*?\sencoding\s*=\s*["']([^"']+)["']`)

var cdataContent = regexp.MustCompile(`(?s)^([ \t\r\n]*)<!\[CDATA\[(.*)\]\]>([ \t\r\n]*)$`)

// textMapper converts a setup file to UTF-8 so that it can be decoded
// with its byte offsets, which are then mapped back to the original.
type textMapper struct {
	single bool          // the original is a single byte encoding
	toByte map[rune]byte // the bytes of a single byte encoding for non-ASCII characters
	toRaw  []int         // offset in the original for each offset in the UTF-8 text
	utf8   []byte
}

// newTextMapper prepares raw for decoding. Files declared as UTF-8 or
// ASCII, or with no declaration, are used as they are. Single byte
// encodings, such as the ISO-8859-1 that the Gateway Setup Editor
// declares, are converted one byte at a time. Any other encoding is an
// error.
func newTextMapper(raw []byte) (*textMapper, error) {
	m := &textMapper{utf8: raw}
	match := xmlDeclEncoding.FindSubmatch(raw[:min(len(raw), 512)])
	if match == nil {
		return m, nil
	}
	label := string(match[1])
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "utf-8", "utf8", "us-ascii", "ascii":
		return m, nil
	}

	// the characters of the upper half of a single byte encoding. A few
	// bytes can be undefined, but a multi-byte encoding has many that
	// cannot be decoded alone
	var table [128]string
	m.toByte = make(map[rune]byte)
	undefined := 0
	for b := range table {
		r, err := charset.NewReaderLabel(label, bytes.NewReader([]byte{byte(b + 0x80)}))
		if err != nil {
			return nil, fmt.Errorf("unsupported encoding %q: %w", label, err)
		}
		d, err := io.ReadAll(r)
		if err != nil || !utf8.Valid(d) || utf8.RuneCount(d) != 1 {
			return nil, fmt.Errorf("encoding %q is not supported (not a single byte encoding)", label)
		}
		table[b] = string(d)
		if rn, _ := utf8.DecodeRune(d); rn == utf8.RuneError {
			if undefined++; undefined > 8 {
				return nil, fmt.Errorf("encoding %q is not supported (not a single byte encoding)", label)
			}
			continue
		} else {
			m.toByte[rn] = byte(b + 0x80)
		}
	}
	m.single = true

	out := make([]byte, 0, len(raw)+len(raw)/8)
	m.toRaw = make([]int, 0, len(raw)+len(raw)/8+1)
	for i, b := range raw {
		d := []byte{b}
		if b >= utf8.RuneSelf {
			d = []byte(table[b-0x80])
		}
		out = append(out, d...)
		for range d {
			m.toRaw = append(m.toRaw, i)
		}
	}
	m.toRaw = append(m.toRaw, len(raw))
	m.utf8 = out
	return m, nil
}

// raw returns the offset in the original of an offset in the converted
// text.
func (m *textMapper) raw(offset int) int {
	if m.toRaw == nil {
		return offset
	}
	return m.toRaw[offset]
}

// encode returns s in the original encoding, using character references
// for any characters that it cannot represent if charRefs is true.
func (m *textMapper) encode(s string, charRefs bool) ([]byte, error) {
	if !m.single {
		return []byte(s), nil
	}
	var out []byte
	for _, r := range s {
		switch b, ok := m.toByte[r]; {
		case r < utf8.RuneSelf:
			out = append(out, byte(r))
		case ok:
			out = append(out, b)
		case charRefs:
			out = append(out, fmt.Sprintf("&#x%X;", r)...)
		default:
			return nil, fmt.Errorf("%q cannot be written in the encoding of the setup file", string(r))
		}
	}
	return out, nil
}

// escapeText escapes s for the content of an element
func escapeText(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\r", "&#xD;").Replace(s)
}

type textEdit struct {
	start, end int // offsets in the converted text
	old        string
	text       string
}

// editIncludeLocations returns raw with the content of every include
// `location` replaced where its value is a key of changes, and the
// number of replacements for each key. The locations are those of the
// `include` elements in the top-level `includes` element and in nested
// `includeGroup` elements, as read into the model. Everything else in
// raw, including the surrounding whitespace of the changed text, is
// returned unchanged.
func editIncludeLocations(raw []byte, changes map[string]string) (out []byte, counts map[string]int, err error) {
	m, err := newTextMapper(raw)
	if err != nil {
		return nil, nil, err
	}

	dec := xml.NewDecoder(bytes.NewReader(m.utf8))
	// the text is already UTF-8
	dec.CharsetReader = func(_ string, r io.Reader) (io.Reader, error) { return r, nil }

	type location struct {
		start, end int
		value      string
		complex    bool // comments or elements in the text
	}
	type frame struct {
		name string
		last *location // the last location of an include
	}
	var (
		stack []*frame
		cur   *location
		edits []textEdit
		depth int // depth of the element in cur
	)

	// the path to the location of an include, as the model reads them:
	// gateway > includes > includeGroup* > include > location
	includeParent := func() bool {
		if len(stack) < 3 || stack[0].name != "gateway" || stack[1].name != "includes" || stack[len(stack)-1].name != "include" {
			return false
		}
		for _, f := range stack[2 : len(stack)-1] {
			if f.name != "includeGroup" {
				return false
			}
		}
		return true
	}

	for {
		before := int(dec.InputOffset())
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("decode setup XML: %w", err)
		}
		after := int(dec.InputOffset())

		if cur != nil {
			switch t := tok.(type) {
			case xml.CharData:
				cur.value += string(t)
			case xml.StartElement:
				cur.complex = true
				depth++
			case xml.EndElement:
				if depth > 0 {
					depth--
					break
				}
				cur.end = before
				stack[len(stack)-1].last = cur
				cur = nil
			default:
				cur.complex = true
			}
			continue
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "location" && includeParent() {
				cur = &location{start: after}
				continue
			}
			stack = append(stack, &frame{name: t.Name.Local})
		case xml.EndElement:
			f := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if f.name == "include" && f.last != nil {
				// only the last location is read into the model
				if replacement, ok := changes[f.last.value]; ok {
					if f.last.complex {
						return nil, nil, fmt.Errorf("location %q has comments or elements in its text and cannot be edited exactly", f.last.value)
					}
					edits = append(edits, textEdit{start: f.last.start, end: f.last.end, old: f.last.value, text: replacement})
				}
			}
		}
	}
	if cur != nil || len(stack) != 0 {
		return nil, nil, errors.New("setup XML is incomplete")
	}

	counts = make(map[string]int)
	var b bytes.Buffer
	pos := 0
	for _, e := range edits {
		start, end := m.raw(e.start), m.raw(e.end)
		if start < pos || end < start || end > len(raw) {
			return nil, nil, errors.New("location offsets are inconsistent")
		}
		content := raw[start:end]
		replacement, err := m.replacement(content, e.text)
		if err != nil {
			return nil, nil, fmt.Errorf("location %q: %w", e.old, err)
		}
		b.Write(raw[pos:start])
		b.Write(replacement)
		pos = end
		counts[e.old]++
	}
	b.Write(raw[pos:])
	return b.Bytes(), counts, nil
}

// replacement returns the content to use in place of the existing
// content of a location element, keeping any CDATA section and the
// whitespace around the text.
func (m *textMapper) replacement(content []byte, text string) ([]byte, error) {
	if match := cdataContent.FindSubmatch(content); match != nil {
		if bytes.Contains(match[2], []byte("<![CDATA[")) {
			return nil, errors.New("text with several CDATA sections cannot be edited exactly")
		}
		data, err := m.encode(strings.ReplaceAll(text, "]]>", "]]]]><![CDATA[>"), false)
		if err != nil {
			return nil, err
		}
		return bytes.Join([][]byte{match[1], []byte("<![CDATA["), data, []byte("]]>"), match[3]}, nil), nil
	}
	if bytes.Contains(content, []byte("<![CDATA[")) {
		return nil, errors.New("text mixing CDATA and other text cannot be edited exactly")
	}

	trimmed := bytes.TrimLeft(content, " \t\r\n")
	lead := content[:len(content)-len(trimmed)]
	core := bytes.TrimRight(trimmed, " \t\r\n")
	trail := trimmed[len(core):]
	data, err := m.encode(escapeText(text), true)
	if err != nil {
		return nil, err
	}
	return bytes.Join([][]byte{lead, data, trail}, nil), nil
}
