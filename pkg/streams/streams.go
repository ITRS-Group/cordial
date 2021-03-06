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
package streams

import (
	"fmt"
	"io"

	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/pkg/xmlrpc"
)

func init() {
	// logger.EnableDebugLog()
}

var (
	log      = logger.Log
	logDebug = logger.Debug
	logError = logger.Error
)

type Stream struct {
	io.Writer
	io.StringWriter
	xmlrpc.Sampler
	name string
}

// Sampler - wrap calls to xmlrpc
func Sampler(url string, entityName string, samplerName string) (s Stream, err error) {
	logDebug.Printf("called")
	sampler, err := xmlrpc.NewClient(url, entityName, samplerName)
	s = Stream{}
	s.Sampler = sampler
	return
}

func (s *Stream) SetStreamName(name string) {
	s.name = name
}

func (s Stream) Write(data []byte) (n int, err error) {
	if s.name == "" {
		return 0, fmt.Errorf("streamname not set")
	}
	err = s.WriteMessage(s.name, string(data))
	if err != nil {
		return 0, err
	}
	n = len(data)
	return
}

func (s Stream) WriteString(data string) (n int, err error) {
	if s.name == "" {
		return 0, fmt.Errorf("streamname not set")
	}
	err = s.WriteMessage(s.name, data)
	if err != nil {
		return 0, err
	}
	n = len(data)
	return
}
