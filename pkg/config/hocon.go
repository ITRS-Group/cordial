/*
Copyright © 2022 ITRS Group

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

package config

import (
	"bytes"
	"encoding/json"
	"os"
	"regexp"

	"github.com/gurkankaymak/hocon"
	"github.com/spf13/viper"
)

var discardRE = regexp.MustCompile(`(?m)^\s*#.*$`)
var shrinkBackSlashRE = regexp.MustCompile(`(?m)\\\\`)

// MergeHOCONConfig parses the HOCON configuration in conf and merges the
// results into the cf *config.Config object
func (c *Config) MergeHOCONConfig(conf string) (err error) {
	conf = discardRE.ReplaceAllString(conf, "")
	hc, err := hocon.ParseString(conf)
	if err != nil {
		return
	}

	vc := viper.New()
	vc.SetConfigType("json")

	j, err := json.Marshal(hc.GetRoot())
	j = shrinkBackSlashRE.ReplaceAll(j, []byte{'\\'})
	if err != nil {
		return
	}
	cs := bytes.NewReader(j)
	if err := vc.ReadConfig(cs); err != nil {
		return err
	}

	c.MergeConfigMap(vc.AllSettings())
	return
}

// MergeHOCONFile reads a HOCON configuration file in path and
// merges the settings into the cf *config.Config object
func (c *Config) MergeHOCONFile(p string) (err error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}
	return c.MergeHOCONConfig(string(b))
}

// ReadHOCONFile loads a HOCON format configuration from file. To
// control behaviour and options like config.Load() use
// MergeHOCONConfig() or MergeHOCONFile() with an existing config.Config
// structure.
func ReadHOCONFile(file string) (c *Config, err error) {
	c = New()
	err = c.MergeHOCONFile(file)
	return
}
