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
