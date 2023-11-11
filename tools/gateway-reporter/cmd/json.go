package cmd

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"

	"github.com/itrs-group/cordial/pkg/config"
)

// outputJSON writes the slice of Entity structs to w
func outputJSON(cf *config.Config, gateway string, Entities []Entity) (err error) {
	dir := cf.GetString("output.directory")
	_ = os.MkdirAll(dir, 0775)

	conftable := config.LookupTable(map[string]string{
		"gateway":  gateway,
		"datetime": startTimestamp,
	})

	filename := cf.GetString("output.formats.json", conftable)
	if !filepath.IsAbs(filename) {
		filename = path.Join(dir, filename)
	}

	w, err := os.Create(filename)
	if err != nil {
		return
	}
	defer w.Close()

	e := json.NewEncoder(w)
	e.SetEscapeHTML(false)
	e.SetIndent("", "    ")
	return e.Encode(Entities)
}
