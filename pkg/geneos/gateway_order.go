package geneos

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"golang.org/x/net/html/charset"
)

// The order of sibling elements is significant in a Gateway setup, but
// encoding/xml writes the modelled fields of a struct in field order and
// then the elements that were kept in its Other field. To write a
// Gateway read from a document with the sibling elements in the order
// of that document, ReadGateway records the order of the child elements
// of every element (xmlOrder) and writing restores it with a pass over
// the marshalled bytes (reorderXML). No schema is needed.

// xmlOrder is the order of the child elements of an element in a
// document that was read. Only the local names of the elements are
// used, so namespace prefixes, which encoding/xml may write
// differently, do not matter. kids is nil if no child has children of
// its own, and an element without children is a nil *xmlOrder. An
// xmlOrder is never changed after it is built so can be shared by
// copies of a Gateway and used from many goroutines.
type xmlOrder struct {
	names []string
	kids  []*xmlOrder
}

// buildXMLOrder returns the order of the child elements, at every
// level, of the root element of data. It returns nil for a root element
// without children.
func buildXMLOrder(data []byte) (*xmlOrder, error) {
	type frame struct {
		names []string
		kids  []*xmlOrder
	}
	d := xml.NewDecoder(bytes.NewReader(data))
	d.CharsetReader = charset.NewReaderLabel
	names := map[string]string{} // intern the element names, there are few
	var stack []*frame
	for {
		tok, err := d.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = io.ErrUnexpectedEOF
			}
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name, ok := names[t.Name.Local]
			if !ok {
				name = t.Name.Local
				names[name] = name
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.names = append(parent.names, name)
				parent.kids = append(parent.kids, nil)
			}
			stack = append(stack, &frame{})
		case xml.EndElement:
			f := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			var order *xmlOrder
			if len(f.names) > 0 {
				order = &xmlOrder{names: f.names}
				for _, k := range f.kids {
					if k != nil {
						order.kids = f.kids
						break
					}
				}
			}
			if len(stack) == 0 {
				return order, nil
			}
			parent := stack[len(stack)-1]
			parent.kids[len(parent.kids)-1] = order
		}
	}
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

// xmlSpan is an element in a marshalled document: the bytes of its
// start tag are [start, openEnd), of its end tag [closeStart, end) and
// everything in between, the content, is the children and the gaps
// before, between and after them. A self-closing element has
// openEnd == closeStart == end.
type xmlSpan struct {
	name                            string
	start, openEnd, closeStart, end int
	children                        []*xmlSpan
}

func parseXMLSpans(data []byte) (root *xmlSpan, count int, err error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	var stack []*xmlSpan
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
func reorderXML(data []byte, ref *xmlOrder) ([]byte, error) {
	root, count, err := parseXMLSpans(data)
	if err != nil {
		return nil, fmt.Errorf("parse marshalled XML: %w", err)
	}
	out := bytes.NewBuffer(make([]byte, 0, len(data)))
	out.Write(data[:root.start])
	if err := root.render(out, data, ref); err != nil {
		return nil, err
	}
	out.Write(data[root.end:])

	// the children have only moved, so check that nothing was lost or
	// repeated, and that the result is still the same document
	if out.Len() != len(data) {
		return nil, fmt.Errorf("reordered document is %d bytes, not %d", out.Len(), len(data))
	}
	result := out.Bytes()
	again, againCount, err := parseXMLSpans(result)
	if err != nil {
		return nil, fmt.Errorf("parse reordered XML: %w", err)
	}
	if againCount != count || !sameElementCounts(root, again) {
		return nil, errors.New("reordered document does not have the same elements")
	}
	return result, nil
}

// sameElementCounts returns true if for every element name a and b
// have the same number of elements
func sameElementCounts(a, b *xmlSpan) bool {
	counts := map[string]int{}
	var walk func(*xmlSpan, int)
	walk = func(s *xmlSpan, delta int) {
		counts[s.name] += delta
		for _, c := range s.children {
			walk(c, delta)
		}
	}
	walk(a, 1)
	walk(b, -1)
	for _, n := range counts {
		if n != 0 {
			return false
		}
	}
	return true
}

func (s *xmlSpan) render(out *bytes.Buffer, data []byte, ref *xmlOrder) error {
	if len(s.children) == 0 {
		out.Write(data[s.start:s.end])
		return nil
	}
	names := make([]string, len(s.children))
	for i, c := range s.children {
		names[i] = c.name
	}
	perm, kids := matchChildOrder(names, ref)
	if err := checkPermutation(perm, len(names)); err != nil {
		return fmt.Errorf("element %q: %w", s.name, err)
	}

	out.Write(data[s.start:s.children[0].start])
	for slot, i := range perm {
		if err := s.children[i].render(out, data, kids[i]); err != nil {
			return err
		}
		if next := slot + 1; next < len(s.children) {
			out.Write(data[s.children[slot].end:s.children[next].start])
		}
	}
	out.Write(data[s.children[len(s.children)-1].end:s.end])
	return nil
}

// matchChildOrder returns the order, as indexes into names, in which the
// children are to be written, and for each child the part of ref that
// belongs to it, or nil. A nil ref keeps the order.
func matchChildOrder(names []string, ref *xmlOrder) (perm []int, kids []*xmlOrder) {
	n := len(names)
	kids = make([]*xmlOrder, n)
	perm = make([]int, 0, n)
	if ref == nil {
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
			continue // no longer there
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
		perm = append(perm, extra[i+1]...)
	}

	// An extra child can end up before an earlier one with the same name,
	// if ref has fewer of that name than there are children, so keep
	// the sequence of names and give the children with each name the
	// places of that name in their own order
	nextByName := map[string]int{}
	for slot, i := range perm {
		name := names[i]
		perm[slot] = byName[name][nextByName[name]]
		nextByName[name]++
	}
	return
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
