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

/*
Support for Geneos Gateway REST Commands

Based on https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_commands_tr.html#REST_Service

*/
package commands

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/itrs-group/cordial/pkg/xpath"
)

// A Command is made up of a Name, a Target and optional Args
type Command struct {
	Name   string            `json:"command"`
	Target string            `json:"target"`
	Args   map[string]string `json:"args,omitempty"`
}

type CommandsResponseRaw struct {
	MimeType   []map[string]string `json:"mimetype,omitempty"`
	Status     string              `json:"status,omitempty"`
	StreamData []map[string]string `json:"streamdata,omitempty"`
	Dataview   *Dataview           `json:"dataview,omitempty`
}

type CommandsResponse struct {
	MimeType   map[string]string   `json:"mimetype,omitempty"`
	Status     string              `json:"status,omitempty"`
	StreamData map[string][]string `json:"streamdata,omitempty"`
	Dataview   *Dataview           `json:"dataview"`
}

type Connection struct {
	BaseURL            *url.URL
	AuthType           int
	Username           string
	Password           string
	SSO                SSOAuth
	InsecureSkipVerify bool
}

// Set-up a Gateway REST command connection
func Dial(u *url.URL, options ...CommandOptions) (c *Connection, err error) {
	c = &Connection{
		BaseURL: u,
	}
	evalOptions(c, options...)
	return
}

// execute a command, return the http response
func (c *Connection) Do(endpoint string, command interface{}) (cr CommandsResponse, err error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify},
	}
	client := &http.Client{Transport: tr}

	r, err := url.Parse(endpoint)
	if err != nil {
		return
	}
	u := c.BaseURL.ResolveReference(r)

	q, err := json.Marshal(command)
	if err != nil {
		return
	}
	query := bytes.NewBuffer(q)
	req, err := http.NewRequest("POST", u.String(), query)
	if err != nil {
		return
	}
	switch c.AuthType {
	case Basic:
		req.SetBasicAuth(c.Username, c.Password)
	case SSO:
		//
	default:
		//
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode > 299 && resp.StatusCode != 400 {
		err = fmt.Errorf(resp.Status)
		return
	}
	b, _ := io.ReadAll(resp.Body)
	var raw CommandsResponseRaw
	if err = json.Unmarshal(b, &raw); err != nil {
		err = fmt.Errorf("%w", err)
		return
	}
	resp.Body.Close()
	cr = CookResponse(raw)
	if cr.Status == "error" {
		err = fmt.Errorf("%s: %v", cr.Status, cr.StreamData["stderr"])
		return
	}
	return
}

// Convert a raw response into a slightly more structured one where the interleaved
// stream messages are merged, in order, into slices for each stream
func CookResponse(raw CommandsResponseRaw) (cr CommandsResponse) {
	cr.Status = raw.Status
	cr.Dataview = raw.Dataview

	for _, m := range raw.MimeType {
		cr.MimeType = make(map[string]string)
		for k, v := range m {
			cr.MimeType[k] = v
		}
	}
	cr.StreamData = make(map[string][]string)
	for _, s := range raw.StreamData {
		for k, v := range s {
			if _, ok := cr.StreamData[k]; !ok {
				cr.StreamData[k] = []string{}
			}
			cr.StreamData[k] = append(cr.StreamData[k], strings.TrimSpace(v))
		}
	}
	return
}

func (c *Connection) SnoozeManual(target *xpath.XPath, info string) (err error) {
	const endpoint = "/rest/runCommand"
	command := &Command{
		Target: target.String(),
		Args:   map[string]string{"1": info},
	}
	if target.IsGateway() || target.IsProbe() || target.IsEntity() {
		command.Name = "/SNOOZE:manual"
	} else if target.IsSampler() || target.IsHeadline() || target.IsTableCell() || target.IsDataview() {
		command.Name = "/SNOOZE:manualAllMe"
		command.Args["5"] = "this"
	}
	if target.IsDataview() {
		fmt.Println("target is dataview", target.String())
	}
	_, err = c.Do(endpoint, command)

	return
}

func (c *Connection) Unsnooze(target *xpath.XPath, info string) (err error) {
	const endpoint = "/rest/runCommand"
	command := &Command{
		Target: target.String(),
	}
	if target.IsGateway() || target.IsProbe() || target.IsEntity() {
		command.Name = "/SNOOZE:unsnooze"
		command.Args = map[string]string{"1": info}
	}
	if target.Rows || target.Headline != nil || target.Sampler != nil {
		command.Name = "/SNOOZE:unsnoozeAllMe"
		command.Args = map[string]string{"1": "this", "2": info}
	}
	_, err = c.Do(endpoint, command)
	return
}

func (c *Connection) SnoozeInfo(target *xpath.XPath) (info string, err error) {
	const endpoint = "/rest/runCommand"
	command := &Command{
		Name:   "/SNOOZE:info",
		Target: target.String(),
	}
	cr, err := c.Do(endpoint, command)
	info = strings.Join(cr.StreamData["stdout"], "\n")
	return
}
