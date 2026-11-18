package geneos

import (
	"bytes"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html/charset"
)

// The order of sibling elements is significant in a Gateway setup, but
// encoding/xml writes the modelled fields of a struct in field order and
// then the elements that were kept in its Other field. To write a
// Gateway read from a document with the sibling elements in the order
// of that document, ReadGateway records the order of the child elements
// of every element (xmlOrder) and writing restores it with a pass over
// the marshalled bytes (reorderXML). No schema is needed.

// xmlOrder is what is remembered about an element of a document that
// was read: the order of its child elements and, for the elements the
// model does not write as they were read, the original bytes.
//
// Only the local names of the elements are used, so namespace prefixes,
// which encoding/xml may write differently, do not matter. kids is nil if
// no child has anything to remember, and a child that has nothing is a
// nil *xmlOrder. An xmlOrder is never changed after it is built so can
// be shared by copies of a Gateway and used from many goroutines.
type xmlOrder struct {
	names []string
	kids  []*xmlOrder

	// patch is set on an element that the model writes differently from
	// how it was read. If the model still writes the element as it did
	// for the document that was read, which means that it has not been
	// changed, the original is written in its place. sum is the
	// fingerprint of what the model wrote then.
	patch *xmlPatch
	sum   xmlSum

	// attrs are the attributes of the element, used to check that it is
	// the element that was read when it is found by position.
	attrs string

	// patches is, for the root element only, the same elements found by
	// what the model wrote, to find those that have moved.
	patches *xmlPatches

	// comments are the comments directly in the element, that the model
	// does not write as they were, with the fingerprint of what it writes.
	comments    []xmlComment
	commentsSum xmlSum

	// ghost is, for a child of the root element that the model does not
	// have at all, the original element. It is written in its place if the
	// model still has no element for it.
	ghost      []byte
	ghostElems int

	// prolog and epilog are, for the root element only, whatever came
	// before and after it, such as the XML declaration and comments.
	prolog, epilog []byte
}

// emptyOrder is the xmlOrder of an element that has no attributes, text
// or children, and is not in a namespace, such as <headlines/>. The model
// leaves these out if there is nothing in the field for them.
var emptyOrder = &xmlOrder{}

// xmlPatches are the elements of a document that the model writes
// differently from how they were read, because it has no field for
// something in them, or the field is of a different type, or it leaves
// out an empty element. They are found by the name of the element and a
// fingerprint of what the model wrote for the element, so that an
// element that has not been changed, wherever it is by the time the
// document is written, can be replaced by the original.
type xmlPatches struct {
	byName map[string]map[xmlSum][]*xmlPatch
	sizes  map[int]bool // the lengths of what the model wrote
}

type xmlPatch struct {
	raw      []byte // the original element, as UTF-8
	rawElems int    // the number of elements in raw
}

// dropAmbiguous removes the patches that cannot be told from other
// elements by what the model writes, because it writes the same for
// elements that were written differently, such as <double>0</double> for
// both 0 and 0.0. They are still found by their position.
func (p *xmlPatches) dropAmbiguous(m *xmlSpan, model []byte) {
	counts := map[string]map[xmlSum]int{}
	var walk func(*xmlSpan)
	walk = func(s *xmlSpan) {
		if keys, ok := p.byName[s.name]; ok && p.sizes[s.end-s.start] {
			sum := sumOf(model[s.start:s.end])
			if _, ok := keys[sum]; ok {
				if counts[s.name] == nil {
					counts[s.name] = map[xmlSum]int{}
				}
				counts[s.name][sum]++
			}
		}
		for _, c := range s.children {
			walk(c)
		}
	}
	walk(m)
	for name, keys := range p.byName {
		for sum, list := range keys {
			if counts[name][sum] != len(list) {
				delete(keys, sum)
			}
		}
		if len(keys) == 0 {
			delete(p.byName, name)
		}
	}
}

func (p *xmlPatches) add(name string, sum xmlSum, size int, patch *xmlPatch) {
	if p.byName == nil {
		p.byName = map[string]map[xmlSum][]*xmlPatch{}
		p.sizes = map[int]bool{}
	}
	if p.byName[name] == nil {
		p.byName[name] = map[xmlSum][]*xmlPatch{}
	}
	p.byName[name][sum] = append(p.byName[name][sum], patch)
	p.sizes[size] = true
}

// xmlComment is a comment, with the number of child elements before it
type xmlComment struct {
	slot int
	text string
}

type xmlSum [16]byte

func sumOf(b []byte) xmlSum {
	h := sha256.Sum256(b)
	return xmlSum(h[:16])
}

// utf8Document returns data as UTF-8 if it declares another encoding, with
// the encoding of the declaration changed to match
func utf8Document(data []byte) ([]byte, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	m := xmlDeclEncoding.FindSubmatchIndex(data)
	if m == nil {
		return data, nil
	}
	label := strings.TrimSpace(string(data[m[2]:m[3]]))
	if strings.EqualFold(label, "utf-8") || strings.EqualFold(label, "utf8") || strings.EqualFold(label, "us-ascii") {
		return data, nil
	}
	enc, _ := charset.Lookup(label)
	if enc == nil {
		return nil, fmt.Errorf("unknown encoding %q", label)
	}
	converted, err := enc.NewDecoder().Bytes(data)
	if err != nil {
		return nil, err
	}
	m = xmlDeclEncoding.FindSubmatchIndex(converted)
	if m == nil {
		return nil, errors.New("encoding declaration lost")
	}
	return append(converted[:m[2]:m[2]], append([]byte("UTF-8"), converted[m[3]:]...)...), nil
}

var xmlDeclEncoding = regexp.MustCompile(`^\s*<\?xml[^>]*?\sencoding\s*=\s*["']([^"']*)["']`)

// buildXMLOrder returns what is to be remembered of the document data,
// which is read as UTF-8, with the order of the child elements at every
// level. If model is given, which is the document that the model writes
// for data, then the elements that are not the same as in data are
// remembered. It returns nil for a root element with nothing to remember.
func buildXMLOrder(data, model []byte) (*xmlOrder, error) {
	root, _, err := parseXMLSpans(data, true)
	if err != nil {
		return nil, err
	}
	var modelRoot *xmlSpan
	if model != nil {
		if modelRoot, _, err = parseXMLSpans(model, true); err != nil {
			return nil, err
		}
	}
	patches := &xmlPatches{}
	order := root.order(data, modelRoot, model, true, patches)
	if modelRoot != nil {
		patches.dropAmbiguous(modelRoot, model)
	}
	if order == emptyOrder {
		order = nil
	}
	prolog := bytes.TrimSpace(data[:root.start])
	epilog := bytes.TrimSpace(data[root.end:])
	if len(patches.byName) > 0 || len(prolog) > 0 || len(epilog) > 0 {
		if order == nil {
			order = &xmlOrder{}
		}
		order.prolog, order.epilog = bytes.Clone(prolog), bytes.Clone(epilog)
		if len(patches.byName) > 0 {
			order.patches = patches
		}
	}
	return order, nil
}

// order returns the xmlOrder for s, an element of the document data, with
// m, the element the model writes for it, if there is one, and adds to
// patches any elements that are not written as they are. The root
// element is never patched, its attributes are those of the Gateway.
func (s *xmlSpan) order(data []byte, m *xmlSpan, model []byte, root bool, patches *xmlPatches) *xmlOrder {
	o := &xmlOrder{attrs: s.attrs}
	if m != nil && !root && !s.sameAs(m) {
		o.patch = &xmlPatch{raw: bytes.Clone(data[s.start:s.end]), rawElems: s.count()}
		o.sum = sumOf(model[m.start:m.end])
		patches.add(s.name, o.sum, m.end-m.start, o.patch)
	}
	if m != nil && !s.sameComments(data, m, model) {
		for _, c := range s.comments {
			o.comments = append(o.comments, xmlComment{slot: c.slot, text: string(data[c.start:c.end])})
		}
		o.commentsSum = m.commentsSum(model)
	}

	// the children are only paired if the model has the same number of
	// each, as otherwise it is not known which is which
	var pairs map[string][]*xmlSpan
	if m != nil && (root || s.sameChildren(m)) {
		pairs = map[string][]*xmlSpan{}
		for _, c := range m.children {
			pairs[c.name] = append(pairs[c.name], c)
		}
	}
	names := map[string]int{}
	for _, c := range s.children {
		o.names = append(o.names, c.name)
		var cm *xmlSpan
		if pairs != nil {
			// the counts are the same unless this is the root
			if list := pairs[c.name]; names[c.name] < len(list) {
				cm = list[names[c.name]]
			}
			names[c.name]++
		}
		kid := c.order(data, cm, model, false, patches)
		if root && pairs != nil && cm == nil {
			kid = &xmlOrder{ghost: bytes.Clone(data[c.start:c.end]), ghostElems: c.count()}
		}
		o.kids = append(o.kids, kid)
	}
	if !slices.ContainsFunc(o.kids, func(k *xmlOrder) bool { return k != nil }) {
		o.kids = nil
	}
	if len(o.names) == 0 && o.comments == nil && o.patch == nil {
		if o.attrs != "" {
			return o // so that the attributes are checked
		}
		if len(s.children) == 0 && s.attrs == "" && s.text == "" && s.space == "" {
			return emptyOrder
		}
		return nil
	}
	return o
}

// sameComments returns true if the model writes the comments of s as
// they are, including between which children they are
func (s *xmlSpan) sameComments(data []byte, m *xmlSpan, model []byte) bool {
	if len(s.comments) != len(m.comments) {
		return false
	}
	for i, c := range s.comments {
		if c.slot != m.comments[i].slot || !bytes.Equal(data[c.start:c.end], model[m.comments[i].start:m.comments[i].end]) {
			return false
		}
	}
	return true
}

// sameAs returns true if s and m have the same attributes and text, and
// children with the same names, which may be in a different order
// as that is restored separately, and the same as each other.
func (s *xmlSpan) sameAs(m *xmlSpan) bool {
	return s.name == m.name && s.attrs == m.attrs && s.text == m.text && s.sameChildren(m)
}

// sameChildren returns true if s and m have the same number of children
// with each name
func (s *xmlSpan) sameChildren(m *xmlSpan) bool {
	if len(s.children) != len(m.children) {
		return false
	}
	counts := map[string]int{}
	for _, c := range s.children {
		counts[c.name]++
	}
	for _, c := range m.children {
		counts[c.name]--
	}
	for _, n := range counts {
		if n != 0 {
			return false
		}
	}
	return true
}

func (s *xmlSpan) count() (n int) {
	n = 1
	for _, c := range s.children {
		n += c.count()
	}
	return
}

// GatewayOrderError reports that the order of the elements of a
// document that was read could not be restored when a Gateway was
// written. The document returned with it is valid and complete but has
// the elements in the order that encoding/xml wrote them.
type GatewayOrderError struct {
	Err error
}

func (e *GatewayOrderError) Error() string {
	return "restore gateway element order: " + e.Err.Error()
}

func (e *GatewayOrderError) Unwrap() error { return e.Err }

// PreserveOrderOf makes g keep the order of the sibling elements of the
// document that src was read from when it is written, for example for a
// Gateway built as a modified copy of src. It has no effect if src was
// not read from a document.
func (g *Gateway) PreserveOrderOf(src *Gateway) {
	if g == nil || src == nil {
		return
	}
	g.source = src.source
}

// MarshalOrdered returns the XML document that WriteGateway writes: the
// Gateway indented with two spaces and a final newline. If the Gateway
// was read from a document (or PreserveOrderOf was used) then the
// differently named sibling elements are in the order of that document,
// see ReadGateway.
//
// If the order cannot be restored the document is returned in the order
// that encoding/xml writes it, with an error that is a
// *GatewayOrderError. Other errors are returned without a document.
func (g *Gateway) MarshalOrdered() ([]byte, error) {
	if g == nil {
		return nil, errors.New("gateway is nil")
	}
	return marshalGatewayOrdered(g, g.source)
}

// gatewayReorder is a variable so that tests can make it fail
var gatewayReorder = reorderXML

func marshalGatewayOrdered(g *Gateway, ref *xmlOrder) ([]byte, error) {
	data, err := xml.MarshalIndent(g, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal gateway XML: %w", err)
	}
	if ref != nil {
		ordered, err := gatewayReorder(data, ref)
		if err != nil {
			return append(data, '\n'), &GatewayOrderError{Err: err}
		}
		data = ordered
	}
	return append(data, '\n'), nil
}

// xmlSpan is an element in a document: the bytes of its start tag are
// [start, openEnd), of its end tag [closeStart, end) and everything in
// between, the content, is the children and the gaps before, between
// and after them. A self-closing element has openEnd == closeStart == end.
// attrs and text, used to compare elements, are only set if wanted.
type xmlSpan struct {
	name                            string
	start, openEnd, closeStart, end int
	space                           string // namespace, if wanted
	attrs                           string // sorted, with namespaces
	text                            string // character data of the element with white space trimmed
	children                        []*xmlSpan
	comments                        []xmlSpanComment // directly in this element
}

// xmlSpanComment is a comment, [start, end) in the document, with the
// number of child elements before it
type xmlSpanComment struct {
	slot       int
	start, end int
}

func (s *xmlSpan) commentsSum(data []byte) xmlSum {
	var b []byte
	for _, c := range s.comments {
		b = append(b, data[c.start:c.end]...)
		b = append(b, 0)
	}
	return sumOf(b)
}

func parseXMLSpans(data []byte, detail bool) (root *xmlSpan, count int, err error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	d.CharsetReader = charset.NewReaderLabel
	var stack []*xmlSpan
	var texts []*strings.Builder
	for {
		before := int(d.InputOffset())
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			s := &xmlSpan{name: t.Name.Local, start: before, openEnd: int(d.InputOffset())}
			s.space = t.Name.Space
			if len(t.Attr) > 0 {
				attrs := make([]string, 0, len(t.Attr))
				for _, a := range t.Attr {
					attrs = append(attrs, a.Name.Space+"|"+a.Name.Local+"="+strconv.Quote(a.Value))
				}
				slices.Sort(attrs)
				s.attrs = strings.Join(attrs, ",")
			}
			if detail {
				texts = append(texts, &strings.Builder{})
			}
			count++
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, s)
			} else if root != nil {
				return nil, 0, errors.New("more than one root element")
			} else {
				root = s
			}
			stack = append(stack, s)
		case xml.EndElement:
			s := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			s.closeStart, s.end = before, int(d.InputOffset())
			if detail {
				s.text = strings.TrimSpace(texts[len(texts)-1].String())
				texts = texts[:len(texts)-1]
			}
		case xml.CharData:
			if detail && len(texts) > 0 {
				texts[len(texts)-1].Write(t)
			}
		case xml.Comment:
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.comments = append(parent.comments, xmlSpanComment{slot: len(parent.children), start: before, end: int(d.InputOffset())})
			}
		}
	}
	if root == nil || len(stack) != 0 {
		return nil, 0, errors.New("no complete root element")
	}
	return root, count, nil
}

// reorderXML returns data, a document written by encoding/xml, with the
// children of every element in the order of ref, the order of the
// elements of the document that the data was originally read from.
//
// The elements are matched with those of ref by path, using the local
// name of the element and the number of earlier siblings with the same
// name, so the n-th sampler is matched with the n-th sampler of the
// original. Siblings with the same name are never swapped, only the
// way that differently named siblings are interleaved is restored. For
// each parent the names of the children in ref are walked and each is
// replaced by the next child with that name, that has not yet been
// used. Children that are not in ref are put after the preceding
// sibling in data that was placed, or first if there is none, and
// those that are only in ref are ignored. The children with a name
// always stay in the order they have in data.
//
// The result is built from the bytes of data. The start and end tags
// and the content of each element, with attributes and namespace
// prefixes as they were marshalled, are copied unchanged and the
// children are moved as a whole. Everything between the children,
// which is white space and any comments, processing instructions and
// text, stays in the same slot between the same number of children and
// does not move with them. This keeps the indentation as written and
// does not need encoding/xml to re-encode names or namespaces.
//
// An element that ref has the original of, because the model did not
// write it as it was read, is replaced by the original if data still has
// what the model wrote when the document was read. Elements that have
// moved are found by what the model wrote. The original elements are
// copied unchanged, so have the white space and order of the original.
// Anything before and after the root element of the original, such as
// the XML declaration, is written before and after the root element.
func reorderXML(data []byte, ref *xmlOrder) ([]byte, error) {
	root, count, err := parseXMLSpans(data, false)
	if err != nil {
		return nil, fmt.Errorf("parse marshalled XML: %w", err)
	}
	r := &xmlRenderer{data: data, used: map[string]int{}, out: bytes.NewBuffer(make([]byte, 0, len(data)+len(data)/16)), wantCount: count}
	if ref != nil {
		r.patches = ref.patches
	}
	if ref != nil && len(ref.prolog) > 0 {
		r.out.Write(ref.prolog)
		r.out.WriteByte('\n')
	}
	r.out.Write(data[:root.start])
	if err := r.render(root, ref, 0, true); err != nil {
		return nil, err
	}
	r.out.Write(data[root.end:])
	if ref != nil && len(ref.epilog) > 0 {
		r.out.WriteByte('\n')
		r.out.Write(ref.epilog)
	}

	// the children have only moved, so check that nothing was lost or
	// repeated, and that the result is still the same document
	result := r.out.Bytes()
	_, again, err := parseXMLSpans(result, false)
	if err != nil {
		return nil, fmt.Errorf("parse reordered XML: %w", err)
	}
	if again != r.wantCount {
		return nil, fmt.Errorf("reordered document has %d elements, not %d", again, r.wantCount)
	}
	return result, nil
}

type xmlRenderer struct {
	data      []byte
	patches   *xmlPatches
	used      map[string]int
	out       *bytes.Buffer
	wantCount int
}

// render writes s, an element of data at depth, with the children in the
// order of ref. trusted is false if the children of s may not be those of
// ref, because the number of siblings with the name of s has changed.
func (r *xmlRenderer) render(s *xmlSpan, ref *xmlOrder, depth int, trusted bool) error {
	raw := r.patch(s)
	if raw == nil && ref != nil && ref.patch != nil && sumOf(r.data[s.start:s.end]) == ref.sum {
		raw = ref.patch
	}
	if raw != nil {
		r.out.Write(raw.raw)
		r.wantCount += raw.rawElems - s.count()
		return nil
	}
	// the comments that the model wrote are replaced by the original ones
	replace := ref != nil && ref.comments != nil && s.commentsSum(r.data) == ref.commentsSum
	if len(s.children) == 0 && !replace && (ref == nil || len(ref.names) == 0) {
		r.out.Write(r.data[s.start:s.end])
		return nil
	}
	names := make([]string, len(s.children))
	counts := map[string]int{}
	for i, c := range s.children {
		names[i] = c.name
		counts[c.name]++
	}
	perm, kids, synth := matchChildOrder(names, ref, trusted)
	if err := checkPermutation(perm, len(names)+len(synth)); err != nil {
		return fmt.Errorf("element %q: %w", s.name, err)
	}
	if len(perm) == 0 && !replace {
		r.out.Write(r.data[s.start:s.end])
		return nil
	}

	// The gaps between the children stay in their places and the same
	// ones are used if there are more children. Elements that are put back
	// are written with the indentation of encoding/xml.
	n := len(s.children)
	childIndent := "\n" + strings.Repeat("  ", depth+1)
	// gap returns the bytes between a and b, without the comments if
	// those are replaced
	gap := func(a, b int) []byte {
		if !replace || len(s.comments) == 0 {
			return r.data[a:b]
		}
		var out []byte
		for _, c := range s.comments {
			if c.start >= a && c.end <= b {
				out = append(out, bytes.TrimRight(r.data[a:c.start], " \t\r\n")...)
				a = c.end
			}
		}
		return append(out, r.data[a:b]...)
	}
	var pre, post []byte
	var between [][]byte
	if n > 0 {
		pre = gap(s.openEnd, s.children[0].start)
		post = gap(s.children[n-1].end, s.closeStart)
		for k := 1; k < n; k++ {
			between = append(between, gap(s.children[k-1].end, s.children[k].start))
		}
	} else if len(perm) > 0 {
		pre = []byte(childIndent)
		post = []byte("\n" + strings.Repeat("  ", depth))
	} else {
		post = gap(s.openEnd, s.closeStart)
	}
	before := func(k int) []byte {
		switch {
		case k == 0 || len(between) == 0:
			return pre
		default:
			return between[min(k, len(between))-1]
		}
	}
	comments := func(k int) {
		if !replace {
			return
		}
		for _, c := range ref.comments {
			if min(c.slot, len(perm)) == k {
				r.out.WriteString(childIndent + c.text)
			}
		}
	}

	r.out.Write(r.data[s.start:s.openEnd])
	for k, i := range perm {
		comments(k)
		r.out.Write(before(k))
		if i >= n {
			r.out.WriteString(synth[i-n].text)
			r.wantCount += synth[i-n].elems
			continue
		}
		child := s.children[i]
		// the child is the one that was read if there are the same
		// number of siblings with the name and the attributes are the same
		sure := trusted && counts[child.name] == ref.count(child.name) && (kids[i] == nil && child.attrs == "" || kids[i] != nil && kids[i].attrs == child.attrs)
		if err := r.render(child, kids[i], depth+1, sure); err != nil {
			return err
		}
	}
	comments(len(perm))
	r.out.Write(post)
	r.out.Write(r.data[s.closeStart:s.end])
	return nil
}

func (o *xmlOrder) count(name string) (n int) {
	if o == nil {
		return 0
	}
	for _, c := range o.names {
		if c == name {
			n++
		}
	}
	return
}

// patch returns the original of s if the model wrote it as it did
// for the document that was read
func (r *xmlRenderer) patch(s *xmlSpan) *xmlPatch {
	if r.patches == nil || len(r.patches.byName) == 0 || !r.patches.sizes[s.end-s.start] {
		return nil
	}
	list := r.patches.byName[s.name][sumOf(r.data[s.start:s.end])]
	if len(list) == 0 {
		return nil
	}
	// elements that were written the same way but were different in the
	// document are used in turn
	sum := sumOf(r.data[s.start:s.end])
	n := r.used[s.name+string(sum[:])]
	r.used[s.name+string(sum[:])]++
	return list[min(n, len(list)-1)]
}

// matchChildOrder returns the order, as indexes into names, in which the
// children are to be written, and for each child the part of ref that
// belongs to it, or nil. A nil ref keeps the order.
func matchChildOrder(names []string, ref *xmlOrder, trusted bool) (perm []int, kids []*xmlOrder, synth []*xmlGhost) {
	n := len(names)
	kids = make([]*xmlOrder, n)
	perm = make([]int, 0, n)
	if ref == nil {
		for i := range n {
			perm = append(perm, i)
		}
		return
	}
	if ref.names == nil {
		for i := range n {
			perm = append(perm, i)
		}
		return
	}

	byName := map[string][]int{}
	for i, name := range names {
		byName[name] = append(byName[name], i)
	}
	used := map[string]int{}
	placed := make([]bool, n)
	var sequence []int // the children that are in ref, in the order of ref
	for ri, name := range ref.names {
		candidates := byName[name]
		if used[name] >= len(candidates) {
			// no longer there, but an empty element is put back as it
			// cannot have been changed
			if trusted && ref.kids != nil {
				if k := ref.kids[ri]; k == emptyOrder {
					sequence = append(sequence, n+len(synth))
					synth = append(synth, &xmlGhost{text: "<" + name + "/>", elems: 1})
				} else if k != nil && k.ghost != nil {
					sequence = append(sequence, n+len(synth))
					synth = append(synth, &xmlGhost{text: string(k.ghost), elems: k.ghostElems})
				}
			}
			continue
		}
		i := candidates[used[name]]
		used[name]++
		placed[i] = true
		sequence = append(sequence, i)
		if ref.kids != nil {
			kids[i] = ref.kids[ri]
		}
	}

	// the children that are not in ref follow the child before them
	// that was placed, indexed by that child plus one so that zero
	// means at the start
	extra := make([][]int, n+1)
	anchor := 0
	for i := range n {
		if placed[i] {
			anchor = i + 1
		} else {
			extra[anchor] = append(extra[anchor], i)
		}
	}
	perm = append(perm, extra[0]...)
	for _, i := range sequence {
		perm = append(perm, i)
		if i < n {
			perm = append(perm, extra[i+1]...)
		}
	}

	// An extra child can end up before an earlier one with the same name,
	// if ref has fewer of that name than there are children, so keep
	// the sequence of names and give the children with each name the
	// places of that name in their own order
	nextByName := map[string]int{}
	for slot, i := range perm {
		if i < n {
			name := names[i]
			perm[slot] = byName[name][nextByName[name]]
			nextByName[name]++
		}
	}
	return
}

// xmlGhost is an element that is written although the model does not
// have it
type xmlGhost struct {
	text  string
	elems int
}

func checkPermutation(perm []int, n int) error {
	if len(perm) != n {
		return fmt.Errorf("%d children reordered into %d", n, len(perm))
	}
	seen := make([]bool, n)
	for _, i := range perm {
		if i < 0 || i >= n || seen[i] {
			return fmt.Errorf("child %d lost or repeated", i)
		}
		seen[i] = true
	}
	return nil
}
