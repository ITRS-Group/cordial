package includes

import (
	"encoding/xml"
	"io"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html/charset"
)

// lintFile walks the tokens of one include file and reports problems
// that the Gateway Setup Editor or the Gateway would reject or
// silently mishandle. It checks structure only, not rule behaviour.
//
//   - key severities are "fail" or "warning" and rule modes are known
//   - regular expression keys compile
//   - a block nested in a block or an if never has only one child,
//     which the GSE text view cannot represent
//   - there are no comments inside rule "if" statements, which the
//     GSE text view cannot represent
//   - rule names are unique within their rule group
//
// It returns the names of the FKM tables defined and referenced.
func lintFile(t *testing.T, path string) (defined, referenced []string) {
	t.Helper()

	f, err := Files.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := xml.NewDecoder(f)
	d.CharsetReader = charset.NewReaderLabel

	type frame struct {
		name     string
		children []string // names of child elements, in order
		attrs    map[string]string
		text     strings.Builder
	}
	var stack []*frame
	inside := func(name string) bool {
		for _, fr := range stack {
			if fr.name == name {
				return true
			}
		}
		return false
	}
	parent := func(n int) string { // n levels above the current element
		if len(stack) <= n {
			return ""
		}
		return stack[len(stack)-1-n].name
	}
	ruleNames := map[string]bool{}
	var groups []string
	// the regular expression and its mode are siblings inside a match
	var matchData, matchRules string

	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				p.children = append(p.children, tok.Name.Local)
			}
			fr := &frame{name: tok.Name.Local, attrs: map[string]string{}}
			for _, a := range tok.Attr {
				fr.attrs[a.Name.Local] = a.Value
			}
			stack = append(stack, fr)
			switch tok.Name.Local {
			case "ruleGroup":
				groups = append(groups, fr.attrs["name"])
			case "rule":
				if n := strings.Join(groups, "/") + "/" + fr.attrs["name"]; ruleNames[n] {
					t.Errorf("%s: duplicate rule name %q in its rule group", path, n)
				} else {
					ruleNames[n] = true
				}
			case "fkmTable":
				if parent(1) == "keyTable" {
					referenced = append(referenced, fr.attrs["ref"])
				} else if n := fr.attrs["name"]; n != "" {
					defined = append(defined, n)
				}
			case "match":
				matchData, matchRules = "", ""
			}

		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].text.Write(tok)
			}

		case xml.Comment:
			if inside("if") {
				t.Errorf("%s: comment inside an if statement: %q", path, strings.TrimSpace(string(tok)))
			}

		case xml.EndElement:
			fr := stack[len(stack)-1]
			text := strings.TrimSpace(fr.text.String())
			switch fr.name {
			case "block":
				// the top-level block of a rule, whose parent is the
				// rule, is fine
				if p := parent(1); len(fr.children) == 1 && (p == "block" || p == "if") {
					t.Errorf("%s: byte %d: nested block with a single child, which the GSE cannot show", path, d.InputOffset())
				}
			case "severity":
				// key severities are limited by the schema, other
				// severities are Geneos severities
				if p := parent(1); p == "key" && text != "fail" && text != "warning" {
					t.Errorf("%s: key severity %q, want fail or warning", path, text)
				}
			case "data":
				if parent(1) == "searchString" {
					matchData = text
				}
			case "rules":
				if parent(1) == "match" {
					matchRules = text
					switch text {
					case "BASIC", "REGEXP", "REGEXP_IGNORE_CASE":
					default:
						t.Errorf("%s: unknown key rules %q", path, text)
					}
				}
			case "match":
				if strings.HasPrefix(matchRules, "REGEXP") && matchData != "" {
					expr := matchData
					if matchRules == "REGEXP_IGNORE_CASE" {
						expr = "(?i)" + expr
					}
					if _, err := regexp.Compile(expr); err != nil {
						t.Errorf("%s: key expression %q: %v", path, matchData, err)
					}
				}
			}
			if fr.name == "ruleGroup" {
				groups = groups[:len(groups)-1]
			}
			stack = stack[:len(stack)-1]
		}
	}
	return
}

func TestIncludeFilesLint(t *testing.T) {
	paths, err := fs.Glob(Files, "files/*.xml")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no include files: %v", err)
	}

	defined := map[string]bool{}
	type ref struct{ path, name string }
	var refs []ref
	for _, path := range paths {
		t.Run(strings.TrimPrefix(path, "files/"), func(t *testing.T) {
			d, r := lintFile(t, path)
			for _, n := range d {
				if defined[n] {
					t.Errorf("FKM table %q is defined more than once", n)
				}
				defined[n] = true
			}
			for _, n := range r {
				refs = append(refs, ref{path, n})
			}
		})
	}

	// a table may be referenced from a different file to the one that
	// defines it, so resolve against all of the files
	for _, r := range refs {
		if !defined[r.name] {
			t.Errorf("%s: FKM table reference %q is not defined in any include", r.path, r.name)
		}
	}
}
