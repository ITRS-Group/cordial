/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package settings

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

// This function and module is designed to provide a single setting repo to enable all ITRS API replated applications to share a standard base.
func GetConfig(conffile string, prefix string) Settings {
	// Initialize Viper interface
	v := viper.New()
	v.SetConfigType("yaml")

	if conffile != "" {
		v.SetConfigFile(conffile)
	} else {
		// Specify Path and Name of the config file
		v.AddConfigPath(".")
		if confdir, err := os.UserConfigDir(); err == nil {
			v.AddConfigPath(confdir)
		}
		v.AddConfigPath("/etc/itrs/")
		v.SetConfigName(prefix + ".yaml")
	}

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	for _, key := range v.AllKeys() {
		val := v.GetString(key)
		v.Set(key, os.ExpandEnv(val))
	}

	// Initialize the "Example" struct
	var c Settings

	// Unmarshal the viper interface into the example struct
	err := v.Unmarshal(&c)
	if err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}

	// Return the settings struct to the application
	return c
}
