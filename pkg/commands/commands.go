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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/xpath"
)

// Command is the wrapper for a Geneos REST Command
type Command struct {
	Name   string       `json:"command,omitempty"`
	Target *xpath.XPath `json:"target"`
	Args   *CommandArgs `json:"args,omitempty"`
	Scope  Scope        `json:"scope,omitempty"`
	Limit  int          `json:"limit,omitempty"`
}

// CommandArgs is a map of argument indices to values
type CommandArgs map[string]string

// A Scope selects which properties the REST dataview snaptshot command
// returns.
type Scope struct {
	Value          bool `json:"value,omitempty"`
	Severity       bool `json:"severity,omitempty"`
	Snooze         bool `json:"snooze,omitempty"`
	UserAssignment bool `json:"user-assignment,omitempty"`
}

// A CommandResponseRaw holds the raw response to a REST command. In
// general callers will only receive and use [CommandResponse]
type CommandResponseRaw struct {
	Target     *xpath.XPath        `json:"target"`
	MimeType   []map[string]string `json:"mimetype"`
	Status     string              `json:"status"`
	StreamData []map[string]string `json:"streamdata"`
	Dataview   *Dataview           `json:"dataview"` // for snapshots only
	XPaths     []string            `json:"xpaths"`
}

// A CommandResponse holds the response from a REST command. The fields
// with values will depends on the command called.
type CommandResponse struct {
	Target         *xpath.XPath      `json:"target"`
	MimeType       map[string]string `json:"mimetype"`
	Status         string            `json:"status"`
	Stdout         string            `json:"stdout"`
	StdoutMimeType string            `json:"stdout_mimetype"`
	Stderr         string            `json:"stderr"`
	ExecLog        string            `json:"execLog"`
	Dataview       *Dataview         `json:"dataview"` // for snapshots only
	XPaths         []string          `json:"xpaths"`
}

// A Connection defines the REST command connection details to a Geneos
// Gateway.
type Connection struct {
	BaseURL            *url.URL
	AuthType           int
	Username           string
	Password           string
	SSO                SSOAuth
	InsecureSkipVerify bool
	Timeout            time.Duration
	rrurls             []*url.URL
	ping               *func(*Connection) error
}

type GeneosRESTError struct {
	Error string `json:"error"`
}

// the default endpoint for normal commands
const endpoint = "/rest/runCommand"

// Connect to a Geneos gateway on the given URL and check the connection.
// The connection is checked by trying a lightweight REST command (fetch
// gateway timezone and time) and if an error is returned then the connection
// should not be reused.
//
// Options can be given to set authentication, ignore unverifiable certificates
// and to override the default "ping" to check the gateway connection
func DialGateway(u *url.URL, options ...Options) (c *Connection, err error) {
	c = &Connection{
		rrurls: []*url.URL{u},
	}
	evalOptions(c, options...)
	err = c.Redial()
	return
}

// Connect to a Geneos gateway given a slice of URLs. This is to support
// standby pairs. Each URL is checked in a random order and the first working
// one is returned. If all URLs fail the check then an error is returned.
//
// Options are the same as for DialGateway()
func DialGateways(urls []*url.URL, options ...Options) (c *Connection, err error) {
	c = &Connection{
		rrurls: urls,
	}
	evalOptions(c, options...)
	err = c.Redial()
	return
}

// Redial the connection, finding the next working endpoint using either the default
// ping function or the one provided when the connection was originally dialled.
//
// An aggregated error is returned if all endpoints fail the connection test.
func (c *Connection) Redial() (err error) {
	// test existing connection, use default func if not overridden
	ping := func(*Connection) error {
		cr, err := c.Do("/rest/gatewayinfo/timezone", &Command{})
		if err != nil {
			return err
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
	errs := []string{}
	for _, u := range c.rrurls {
		c.BaseURL = u
		if err = ping(c); err == nil {
			return nil
		}
		errs = append(errs, fmt.Sprintf("gateway %q: %s", u, err))
	}
	return fmt.Errorf(strings.Join(errs, "\n"))
}

// Do executes command on the REST endpoint, return the http response
func (c *Connection) Do(endpoint string, command *Command) (response CommandResponse, err error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify},
	}
	client := &http.Client{Transport: tr, Timeout: c.Timeout}

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
		// XXX
	default:
		// No auth
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode > 299 {
		var geneosError GeneosRESTError
		if err = json.Unmarshal(b, &geneosError); err != nil {
			geneosError.Error = fmt.Sprintf("unknown error (%s)", string(b))
		}
		response.Status = "error"
		response.Stderr = geneosError.Error
		err = fmt.Errorf(resp.Status)
		return
	}

	var raw CommandResponseRaw
	if err = json.Unmarshal(b, &raw); err != nil {
		err = fmt.Errorf("cannot unmarshal response: %w", err)
		return
	}
	raw.Target = command.Target

	response = cookResponse(raw)
	if response.Status == "error" {
		err = fmt.Errorf("%s: %v", response.Status, response.Stderr)
		return
	}
	return
}

// Convert a raw response into a structured one where the interleaved
// stream messages are concatenated into strings for each stream
func cookResponse(raw CommandResponseRaw) (cr CommandResponse) {
	cr = CommandResponse{
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

// RunCommand runs command against exactly one target, returning the response
func (c *Connection) RunCommand(command string, target *xpath.XPath, args ...Args) (response CommandResponse, err error) {
	arguments := &CommandArgs{}
	evalArgOptions(arguments, args...)
	targets, err := c.CommandTargets(command, target)
	if err != nil {
		return
	}
	if len(targets) != 1 {
		err = fmt.Errorf("target does not match exactly one data item")
		return
	}
	return c.Do(endpoint, &Command{
		Name:   command,
		Target: target,
		Args:   arguments,
	})
}

// RunCommands runs command against all matching data items, returning
// separately concatenated stdout, stderr and execlog when returned by
// the underlying command
func (c *Connection) RunCommandAll(command string, target *xpath.XPath, args ...Args) (responses []CommandResponse, err error) {
	arguments := &CommandArgs{}
	evalArgOptions(arguments, args...)
	targets, err := c.CommandTargets(command, target)
	if err != nil {
		return
	}
	if len(targets) == 0 {
		err = fmt.Errorf("no matches")
	}
	responses = []CommandResponse{}

	for _, t := range targets {
		cr, err := c.Do(endpoint, &Command{
			Name:   command,
			Target: t,
			Args:   arguments,
		})
		responses = append(responses, cr)
		if err != nil {
			continue
		}
	}
	return
}

var limitRE = regexp.MustCompile(`matches at least (\d+) items`)

// Match returns a slice of all matching XPaths for the target up to
// limit items. If limit is 0 then first a match is tried with the
// default limit of 100 and if that fails with an error hinting at the
// approximate number of matches, then retry with twice this vaule. If
// limit is less than 0 then the default limit of 100 is used.
func (c *Connection) Match(target *xpath.XPath, limit int) (matches []*xpath.XPath, err error) {
	const endpoint = "/rest/xpaths/match"
	if limit < 0 {
		limit = 100
	}
	command := &Command{
		Target: target,
		Limit:  limit,
	}
	cr, err := c.Do(endpoint, command)
	if err != nil {
		if limit == 0 {
			// try again with twice the limit in the error returned
			lims := limitRE.FindStringSubmatch(cr.Stderr)
			if len(lims) > 0 {
				newlimit, _ := strconv.Atoi(lims[1])
				command.Limit = newlimit * 2
				cr, err = c.Do(endpoint, command)
				if err != nil {
					err = fmt.Errorf("%w: %s", err, cr.Stderr)
					return
				}
			}
		} else {
			err = fmt.Errorf("%w: %s", err, cr.Stderr)
			return
		}
	}
	for _, p := range cr.XPaths {
		x, err := xpath.Parse(p)
		if err != nil {
			continue
		}
		matches = append(matches, x)
	}
	return
}

// CommandTargets returns a slice of all XPaths that support the command
// for the given target.
func (c *Connection) CommandTargets(command string, target *xpath.XPath) (matches []*xpath.XPath, err error) {
	const endpoint = "/rest/xpaths/commandTargets"
	cr, err := c.Do(endpoint, &Command{
		Target: target,
		Name:   command,
	})
	if err != nil {
		err = fmt.Errorf("%w: %s", err, cr.Stderr)
		return
	}
	for _, p := range cr.XPaths {
		x, err := xpath.Parse(p)
		if err != nil {
			continue
		}
		matches = append(matches, x)
	}
	return
}
