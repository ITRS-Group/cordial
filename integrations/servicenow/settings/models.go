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

type Settings struct {
	API struct {
		Host   string `yaml:"Host"`
		Port   int    `yaml:"Port"`
		APIKey string `yaml:"APIKey"`
		TLS    struct {
			Enabled     bool   `yaml:"Enabled"`
			Certificate string `yaml:"Certificate"`
			Key         string `yaml:"Key"`
		} `yaml:"TLS"`
	} `yaml:"API"`
	ServiceNow struct {
		Instance              string
		Username              string
		Password              string
		PasswordFile          string
		ClientID              string
		ClientSecret          string
		GeneosSeverityMap     map[string]string
		SearchType            string
		QueryResponseFields   string
		IncidentTable         string
		IncidentDefaults      map[string]string
		IncidentStates        map[int][]string
		IncidentStateDefaults map[string]map[string]string
	} `yaml:"ServiceNow"`
}
