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

package xmlrpc

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"strconv"
)

type valueArray struct {
	String string
	Int    int
	Array  interface{}
}

type methodCall struct {
	Name   string
	Values []valueArray
}

type members struct {
	Name   string `xml:"name"`
	Int    int    `xml:"value>int"`
	String string `xml:"value>string"`
}

type methodResponse struct {
	Boolean      bool      `xml:"params>param>value>boolean"`
	String       string    `xml:"params>param>value>string"`
	Int          int       `xml:"params>param>value>int"`
	SliceStrings []string  `xml:"params>param>value>array>data>value>string"`
	Fault        []members `xml:"fault>value>struct>member"`
}

func (c Client) post(data methodCall) (result methodResponse, err error) {
	// use a custom marshal function as the standard XML ones, even
	// with customer marshallers are almost impossible to control in
	// a way that works consistently. xml.Unmarshal() still works fine.
	output, err := marshal(data)
	if err != nil {
		return
	}

	body := bytes.NewReader(output)
	resp, err := c.Post(c.URL(), "text/xml", body)
	if err != nil {
		logError.Print(err)
		return
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = xml.Unmarshal(b, &result)
	if err != nil {
		return
	}

	if result.Fault != nil {
		err = fmt.Errorf("%d %s", result.Fault[0].Int, result.Fault[1].String)
	}
	return
}

func (c Client) methodBoolWithArgs(method string, args []valueArray) (result bool, err error) {
	req := methodCall{Name: method, Values: args}

	res, err := c.post(req)
	if err != nil {
		return
	}
	result = res.Boolean
	return
}

func (c Client) methodBoolNoArgs(method string) (result bool, err error) {
	req := methodCall{Name: method}

	res, err := c.post(req)
	if err != nil {
		return
	}
	result = res.Boolean
	return
}

func (c Client) methodIntNoArgs(method string) (result int, err error) {
	req := methodCall{Name: method}

	res, err := c.post(req)
	if err != nil {
		return
	}
	result = res.Int
	return
}

func (c Client) methodIntWithArgs(method string, args []valueArray) (result int, err error) {
	req := methodCall{Name: method, Values: args}

	res, err := c.post(req)
	if err != nil {
		return
	}
	result = res.Int
	return
}

func (c Client) methodStringWithArgs(method string, args []valueArray) (result string, err error) {
	req := methodCall{Name: method, Values: args}

	res, err := c.post(req)
	if err != nil {
		return
	}
	result = res.String
	return result, err
}

func (c Client) methodStringsNoArgs(method string) (strings []string, err error) {
	req := methodCall{Name: method}

	result, err := c.post(req)
	if err != nil {
		return
	}
	strings = result.SliceStrings
	return
}

func (c Client) methodStringsWithArgs(method string, args []valueArray) (strings []string, err error) {
	req := methodCall{Name: method, Values: args}

	result, err := c.post(req)
	if err != nil {
		return
	}
	strings = result.SliceStrings
	return
}

func (c Client) methodWithArgs(method string, args []valueArray) (err error) {
	req := methodCall{Name: method, Values: args}

	_, err = c.post(req)
	return
}

func (c Client) methodNoArgs(method string) (err error) {
	req := methodCall{Name: method}

	_, err = c.post(req)
	return
}

func marshal(c methodCall) ([]byte, error) {
	var err error
	var data = "<methodCall><methodName>" + c.Name + "</methodName>"
	if len(c.Values) > 0 {
		data += "<params>"
		for _, a := range c.Values {
			data += "<param><value>"
			if a.String != "" {
				data += "<string>" + a.String + "</string>"
			} else if a.Int != 0 {
				// the only call that passes an Int is a duration in seconds,
				// which must be greater than zero, hence this is valid
				data += "<int>" + strconv.Itoa(a.Int) + "</int>"
			} else if a.Array != nil {
				data += "<array><data>"
				switch a.Array.(type) {
				case []string:
					as := a.Array.([]string)
					for _, s := range as {
						data += "<value><string>" + s + "</string></value>"
					}
				case [][]string:
					ass := a.Array.([][]string)
					for _, s1 := range ass {
						data += "<value><array><data>"
						for _, s2 := range s1 {
							data += "<value><string>" + s2 + "</string></value>"

						}
						data += "</data></array></value>"

					}
				default:
					err = fmt.Errorf("unknown type in args")
				}

				data += "</data></array>"
			}
			data += "</value></param>"
		}
		data += "</params>"
	}
	data += "</methodCall>"
	return []byte(data), err
}
