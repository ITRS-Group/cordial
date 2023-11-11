package cmd

import (
	"bytes"
	"os"
	"path"
	"path/filepath"

	"github.com/itrs-group/cordial/pkg/config"
)

func outputXML(cf *config.Config, gateway string, savedXML *bytes.Buffer) (err error) {
	dir := cf.GetString("output.directory")
	_ = os.MkdirAll(dir, 0775)

	conftable := config.LookupTable(map[string]string{
		"gateway":  gateway,
		"datetime": startTimestamp,
	})

	filename := cf.GetString("output.formats.xml", conftable)
	if !filepath.IsAbs(filename) {
		filename = path.Join(dir, filename)
	}

	f, err := os.Create(filename)
	if err != nil {
		return
	}
	defer f.Close()

	_, err = savedXML.WriteTo(f)
	return
}
