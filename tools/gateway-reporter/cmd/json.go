package cmd

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos"
)

type reportBy struct {
	CreatedBy string    `json:"createdBy,omitempty"`
	Version   string    `json:"version,omitempty"`
	Site      string    `json:"site,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Hostname  string    `json:"hostname,omitempty"`
	Gateway   string    `json:"gateway,omitempty"`
	Probes    int       `json:"probes,omitempty"`
	Entities  int       `json:"entities,omitempty"`
}

type reportJSON struct {
	Report   reportBy `json:"report"`
	Entities []Entity `json:"entities"`
}

// outputJSON writes the slice of Entity structs to w
func outputJSON(cf *config.Config, gateway string, entities []Entity, probes map[string]geneos.Probe) (err error) {
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
	hostname, _ := os.Hostname()
	report := reportJSON{
		Report: reportBy{
			CreatedBy: "ITRS Gateway Reporter",
			Version:   cordial.VERSION,
			Site:      cf.GetString("site", config.Default("ITRS")),
			Timestamp: startTime,
			Hostname:  hostname,
			Gateway:   gateway,
			Probes:    len(probes),
			Entities:  len(entities),
		},
		Entities: entities,
	}
	return e.Encode(report)
}
