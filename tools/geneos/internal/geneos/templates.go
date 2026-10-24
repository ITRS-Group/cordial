package geneos

import (
	"fmt"
	"path"
)

// Initialise templates for the given component on the specified host.
// If ct is nil then all component templates will be initialised.
func WriteTemplates(h *Host, ct *Component, override bool) (err error) {
	for ct := range ct.OrList() {
		if len(ct.Templates) == 0 {
			continue
		}
		templateDir := h.PathTo(ct, "templates")
		h.MkdirAll(templateDir, 0775)

		for _, t := range ct.Templates {
			tmpl := t.Content

			output := path.Join(templateDir, t.Filename)
			if _, err := h.Stat(output); err == nil && !override {
				continue
			}
			if err = h.WriteFile(output, tmpl, 0664); err != nil {
				return
			}
			fmt.Printf("%s template %q written to %s\n", ct, t.Filename, templateDir)
		}
	}

	return
}
