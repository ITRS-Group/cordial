/*
Copyright © 2023 ITRS Group

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
