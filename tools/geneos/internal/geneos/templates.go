package geneos

import (
	"fmt"
	"path"
)

// Initialise templates for the given component on the specified host.
// If ct is nil then all component templates will be initialised.
func WriteTemplates(h *Host, ct *Component) (err error) {
	for ct := range ct.OrList() {
		if len(ct.Templates) == 0 {
			continue
		}
		templateDir := h.PathTo(ct, "templates")
		h.MkdirAll(templateDir, 0775)

		for _, t := range ct.Templates {
			tmpl := t.Content

			if err = h.WriteFile(path.Join(templateDir, t.Filename), tmpl, 0664); err != nil {
				return
			}
			fmt.Printf("%s template %q written to %s\n", ct, t.Filename, templateDir)
		}
	}

	return
}
