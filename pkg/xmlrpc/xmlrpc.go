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

package xmlrpc

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

type methodCall struct {
	XMLName xml.Name     `xml:"methodCall"`
	Name    string       `xml:"methodName"`
	Params  methodParams `xml:"params,omitempty"`
}

type methodParams struct {
	Params []interface{}
}

type methodScalar struct {
	XMLName xml.Name `xml:"param"`
	Scalar  interface{}
}

type methodArray struct {
	XMLName xml.Name `xml:"param"`
	Array   interface{}
}

type methodArrayData struct {
	XMLName xml.Name     `xml:"value"`
	Data    []methodData `xml:"array>data"`
}

type methodData struct {
	Value interface{}
}

type methodString struct {
	XMLName xml.Name `xml:"value"`
	Value   string   `xml:"string"`
}

type methodInt struct {
	XMLName xml.Name `xml:"value"`
	Value   int32    `xml:"int"`
}

type methodBool struct {
	XMLName xml.Name `xml:"value"`
	Value   int      `xml:"boolean"`
}

type methodDouble struct {
	XMLName xml.Name `xml:"value"`
	Value   float64  `xml:"double"`
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

func (c Client) post(method string, args ...interface{}) (result methodResponse, err error) {
	data := &methodCall{Name: method}

	params := []any{}

	for _, arg := range args {
		switch a := arg.(type) {
		case string:
			params = append(params, methodScalar{Scalar: methodString{Value: a}})
		case int:
			params = append(params, methodScalar{Scalar: methodInt{Value: int32(a)}})
		case int32:
			params = append(params, methodScalar{Scalar: methodInt{Value: a}})
		case float64:
			params = append(params, methodScalar{Scalar: methodDouble{Value: a}})
		case bool:
			v := 0
			if a {
				v = 1
			}
			params = append(params, methodScalar{Scalar: methodBool{Value: v}})
		case []string:
			strings := []interface{}{}
			for _, s := range a {
				strings = append(strings, methodString{Value: s})
			}
			params = append(params, methodArray{Array: methodArrayData{Data: []methodData{{Value: strings}}}})
		case [][]string:
			stringstrings := []interface{}{}
			for _, ss := range a {
				strings := []interface{}{}
				for _, s := range ss {
					strings = append(strings, methodString{Value: s})
				}
				stringstrings = append(stringstrings, methodArrayData{Data: []methodData{{Value: strings}}})
			}
			params = append(params, methodArray{Array: methodArrayData{Data: []methodData{{Value: stringstrings}}}})

		default:
			err = fmt.Errorf("unsupported argument \"%v\" type %T", a, a)
			return
		}
	}

	data.Params = methodParams{Params: params}

	output, err := xml.MarshalIndent(data, "", "    ")
	if err != nil {
		return
	}

	body := bytes.NewReader(output)
	resp, err := c.Post(c.String(), "text/xml", body)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	defer c.CloseIdleConnections()

	b, err := io.ReadAll(resp.Body)
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

func (c Client) callMethodBool(method string, args ...interface{}) (result bool, err error) {
	res, err := c.post(method, args...)
	if err != nil {
		return
	}
	result = res.Boolean
	return
}

func (c Client) callMethodInt(method string, args ...interface{}) (result int, err error) {
	res, err := c.post(method, args...)
	if err != nil {
		return
	}
	result = res.Int
	return
}

func (c Client) callMethodString(method string, args ...interface{}) (result string, err error) {
	res, err := c.post(method, args...)
	if err != nil {
		return
	}
	result = res.String
	return result, err
}

func (c Client) callMethodStringSlice(method string, args ...interface{}) (strings []string, err error) {
	result, err := c.post(method, args...)
	if err != nil {
		return
	}
	strings = result.SliceStrings
	return
}

func (c Client) callMethod(method string, args ...interface{}) (err error) {
	_, err = c.post(method, args...)
	return
}
