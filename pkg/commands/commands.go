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
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	"github.com/itrs-group/cordial/pkg/xpath"
)

// A Command is made up of a Name, a Target and optional Args
type Command struct {
	Name   string       `json:"command,omitempty"`
	Target *xpath.XPath `json:"target"`
	Args   *Args        `json:"args,omitempty"`
	Scope  Scope        `json:"scope,omitempty"`
	Limit  int          `json:"limit,omitempty"`
}

type Scope struct {
	Value          bool `json:"value,omitempty"`
	Severity       bool `json:"severity,omitempty"`
	Snooze         bool `json:"snooze,omitempty"`
	UserAssignment bool `json:"user-assignment,omitempty"`
}

type CommandsResponseRaw struct {
	Target     *xpath.XPath        `json:"target"`
	MimeType   []map[string]string `json:"mimetype"`
	Status     string              `json:"status"`
	StreamData []map[string]string `json:"streamdata"`
	Dataview   *Dataview           `json:"dataview"` // for snapshots only
	XPaths     []string            `json:"xpaths"`
}

type CommandsResponse struct {
	Target   *xpath.XPath      `json:"target"`
	MimeType map[string]string `json:"mimetype"`
	Status   string            `json:"status"`
	Stdout   string            `json:"stdout"`
	Stderr   string            `json:"stderr"`
	ExecLog  string            `json:"execLog"`
	Dataview *Dataview         `json:"dataview"` // for snapshots only
	XPaths   []string          `json:"xpaths"`
}

type Connection struct {
	BaseURL            *url.URL
	AuthType           int
	Username           string
	Password           string
	SSO                SSOAuth
	InsecureSkipVerify bool
	rrurls             []*url.URL
	ping               *func(*Connection) error
}

const endpoint = "/rest/runCommand"

// Set-up a Gateway REST command connection
func DialGateway(u *url.URL, options ...CommandOptions) (c *Connection, err error) {
	c = &Connection{
		rrurls: []*url.URL{u},
	}
	evalOptions(c, options...)
	err = c.Redial()
	return
}

// Round-Robin / random dialer. Given a slice of URLs, randomise and
// then try each one in turn until there is a response using the
// liveness ping function. If there is an existing connection configured
// then that is re-tested first before moving onto next URL
//
// all endpoints are given the same options
func DialGateways(urls []*url.URL, options ...CommandOptions) (c *Connection, err error) {
	c = &Connection{
		rrurls: urls,
	}
	evalOptions(c, options...)
	err = c.Redial()
	return
}

// Redial the connection, finding the next working endpooint using the liveness ping() function
// given
func (c *Connection) Redial() (err error) {
	// test existing connection, use default func if not overridden
	ping := func(*Connection) error {
		cr, err := c.Do("/rest/gatewayinfo/timezone", &Command{})
		if err != nil {
			return fmt.Errorf("%w - %s: %v", err, cr.Status, cr.Stderr)
		}
		if cr.Status == "error" {
			return fmt.Errorf("%s: %v", cr.Status, cr.Stderr)
		}
		return nil
	}

	if c.ping != nil {
		ping = *c.ping
	}
	if c.BaseURL != nil && ping(c) == nil {
		return nil
	}

	// shuffle list of connections
	rand.Shuffle(len(c.rrurls), func(i, j int) {
		c.rrurls[i], c.rrurls[j] = c.rrurls[j], c.rrurls[i]

	})

	// loop through, pick the first valid endpoint
	// save all the errors in case no gateway is valid
	errs := ""
	for _, u := range c.rrurls {
		c.BaseURL = u
		if err = ping(c); err == nil {
			return nil
		}
		errs += fmt.Sprintf("%s responded: %q\n", u, err)
	}
	return fmt.Errorf(errs)
}

type GeneosRESTError struct {
	Error string `json:"error"`
}

// execute a command, return the http response
func (c *Connection) Do(endpoint string, command *Command) (cr CommandsResponse, err error) {
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
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode > 299 && resp.StatusCode != 400 {
		var geneosError GeneosRESTError
		if err = json.Unmarshal(b, &geneosError); err != nil {
			geneosError.Error = fmt.Sprintf("unknown error (%s)", string(b))
		}
		cr.Status = "error"
		cr.Stderr = geneosError.Error
		err = fmt.Errorf(resp.Status)
		return
	}

	var raw CommandsResponseRaw
	if err = json.Unmarshal(b, &raw); err != nil {
		err = fmt.Errorf("cannot unmarshal response: %w", err)
		return
	}
	raw.Target = command.Target

	cr = CookResponse(raw)
	if cr.Status == "error" {
		err = fmt.Errorf("%s: %v", cr.Status, cr.Stderr)
		return
	}
	return
}

// Convert a raw response into a slightly more structured one where the interleaved
// stream messages are merged, in order, into slices for each stream
func CookResponse(raw CommandsResponseRaw) (cr CommandsResponse) {
	cr = CommandsResponse{
		Status:   raw.Status,
		Dataview: raw.Dataview,
		XPaths:   raw.XPaths,
		Target:   raw.Target,
	}

	for _, m := range raw.MimeType {
		cr.MimeType = make(map[string]string)
		for k, v := range m {
			cr.MimeType[k] = v
		}
	}

	for _, s := range raw.StreamData {
		for k, v := range s {
			switch k {
			case "stdout":
				cr.Stdout += strings.TrimSpace(v) + "\n"
			case "stderr":
				cr.Stderr += strings.TrimSpace(v) + "\n"
			case "execLog":
				cr.ExecLog += strings.TrimSpace(v) + "\n"
			default:
				panic(fmt.Sprintf("unknown response stream %q", k))
			}
		}
	}
	return
}

// run a command against exactly one valid target, returning the response
func (c *Connection) RunCommand(name string, target *xpath.XPath, options ...ArgOptions) (cr CommandsResponse, err error) {
	args := &Args{}
	evalArgOptions(args, options...)
	targets, err := c.CommandTargets(name, target)
	if err != nil {
		return
	}
	if len(targets) != 1 {
		err = fmt.Errorf("target does not match exactly one data item")
		return
	}
	command := &Command{
		Name:   name,
		Target: target,
		Args:   args,
	}
	return c.Do(endpoint, command)
}

// run command against all matching data items, returning stdout,
// stderr and execlog (concatenated) where applicable
func (c *Connection) RunCommandAll(name string, target *xpath.XPath, options ...ArgOptions) (crs []CommandsResponse, err error) {
	args := &Args{}
	evalArgOptions(args, options...)
	targets, err := c.CommandTargets(name, target)
	if err != nil {
		return
	}
	crs = []CommandsResponse{}

	for _, t := range targets {
		command := &Command{
			Name:   name,
			Target: t,
			Args:   args,
		}
		cr, err := c.Do(endpoint, command)
		crs = append(crs, cr)
		if err != nil {
			continue
		}
	}
	return
}

func (c *Connection) Match(target *xpath.XPath, limit int) (matches []*xpath.XPath, err error) {
	const endpoint = "/rest/xpaths/match"
	if limit < 1 {
		limit = 100
	}
	command := &Command{
		Target: target,
		Limit:  limit,
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

func (c *Connection) CommandTargets(name string, target *xpath.XPath) (matches []*xpath.XPath, err error) {
	const endpoint = "/rest/xpaths/commandTargets"
	command := &Command{
		Target: target,
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
		_, err = c.RunCommandAll("/SNOOZE:manual", target, Arg(1, info))
		return
	}
	if target.IsSampler() || target.IsHeadline() || target.IsTableCell() || target.IsDataview() {
		_, err = c.RunCommandAll("/SNOOZE:manualAllMe", target, Arg(1, info), Arg(5, "this"))
	}
	return
}

func (c *Connection) Unsnooze(target *xpath.XPath, info string) (err error) {
	if target.IsGateway() || target.IsProbe() || target.IsEntity() {
		_, err = c.RunCommandAll("/SNOOZE:unsnooze", target, Arg(1, info))
		return
	}
	if target.Rows || target.Headline != nil || target.Sampler != nil {
		_, err = c.RunCommandAll("/SNOOZE:unsnoozeAllMe", target, Arg(1, "this"), Arg(2, info))
	}
	return
}

func (c *Connection) SnoozeInfo(target *xpath.XPath) (crs []CommandsResponse, err error) {
	crs, err = c.RunCommandAll("/SNOOZE:info", target)
	return
}
