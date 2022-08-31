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
	Name   string `json:"command"`
	Target string `json:"target"`
	Args   *Args  `json:"args,omitempty"`
}

type CommandsResponseRaw struct {
	MimeType   []map[string]string `json:"mimetype"`
	Status     string              `json:"status"`
	StreamData []map[string]string `json:"streamdata"`
	Dataview   *Dataview           `json:"dataview`
	XPaths     []string            `json:"xpaths`
}

type CommandsResponse struct {
	MimeType   map[string]string   `json:"mimetype"`
	Status     string              `json:"status"`
	StreamData map[string][]string `json:"streamdata"`
	Dataview   *Dataview           `json:"dataview"`
	XPaths     []string            `json:"xpaths"`
}

type Connection struct {
	BaseURL            *url.URL
	AuthType           int
	Username           string
	Password           string
	SSO                SSOAuth
	InsecureSkipVerify bool
}

const endpoint = "/rest/runCommand"

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
	cr.XPaths = raw.XPaths

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

// run a command against exactly one valid target, returning stdout,
// stderr and execlog where applicable
func (c *Connection) RunCommand(name string, target *xpath.XPath, options ...ArgOptions) (stdout, stderr, execlog string, err error) {
	args := &Args{}
	evalArgOptions(args, options...)
	targets := c.CommandTargets(name, target)
	if len(targets) != 1 {
		err = fmt.Errorf("target does not match exactly one data item")
		return
	}
	command := &Command{
		Name:   name,
		Target: target.String(),
		Args:   args,
	}
	cr, err := c.Do(endpoint, command)
	if err != nil {
		return
	}
	stdout = strings.Join(cr.StreamData["stdout"], "\n")
	stderr = strings.Join(cr.StreamData["stderr"], "\n")
	execlog = strings.Join(cr.StreamData["execlog"], "\n")
	return
}

// run command against all matching data items, returning stdout,
// stderr and execlog (concatenated) where applicable
func (c *Connection) RunCommandAll(name string, target *xpath.XPath, options ...ArgOptions) (stdout, stderr, execlog map[string]string, err error) {
	args := &Args{}
	evalArgOptions(args, options...)
	targets := c.CommandTargets(name, target)
	stdout = map[string]string{}
	stderr = map[string]string{}
	execlog = map[string]string{}

	for _, t := range targets {
		command := &Command{
			Name:   name,
			Target: t.String(),
			Args:   args,
		}
		cr, err := c.Do(endpoint, command)
		if err != nil {
			continue
		}
		stdout[t.String()] = strings.Join(cr.StreamData["stdout"], "\n")
		stderr[t.String()] = strings.Join(cr.StreamData["stderr"], "\n")
		execlog[t.String()] = strings.Join(cr.StreamData["execlog"], "\n")
	}
	return
}

func (c *Connection) CommandTargets(name string, target *xpath.XPath) (matches []*xpath.XPath) {
	const endpoint = "/rest/xpaths/commandTargets"
	command := &Command{
		Target: target.String(),
		Name:   name,
	}
	cr, err := c.Do(endpoint, command)
	if err != nil {
		panic(err)
		// return
	}
	for _, p := range cr.XPaths {
		x, err := xpath.Parse(p)
		if err != nil {
			panic(err)
			// continue
		}
		matches = append(matches, x)
	}
	return
}

// test commands to work out kinks in args and returns

func (c *Connection) SnoozeManual(target *xpath.XPath, info string) (err error) {
	if target.IsGateway() || target.IsProbe() || target.IsEntity() {
		_, _, _, err = c.RunCommandAll("/SNOOZE:manual", target, Arg(1, info))
		return
	}
	if target.IsSampler() || target.IsHeadline() || target.IsTableCell() || target.IsDataview() {
		_, _, _, err = c.RunCommandAll("/SNOOZE:manualAllMe", target, Arg(1, info), Arg(5, "this"))
	}
	return
}

func (c *Connection) Unsnooze(target *xpath.XPath, info string) (err error) {
	if target.IsGateway() || target.IsProbe() || target.IsEntity() {
		_, _, _, err = c.RunCommandAll("/SNOOZE:unsnooze", target, Arg(1, info))
		return
	}
	if target.Rows || target.Headline != nil || target.Sampler != nil {
		_, _, _, err = c.RunCommandAll("/SNOOZE:unsnoozeAllMe", target, Arg(1, "this"), Arg(2, info))
	}
	return
}

func (c *Connection) SnoozeInfo(target *xpath.XPath) (info map[string]string, err error) {
	info, _, _, err = c.RunCommandAll("/SNOOZE:info", target)
	return
}
