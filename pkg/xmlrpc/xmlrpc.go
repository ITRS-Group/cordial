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
	"github.com/itrs-group/cordial/pkg/logger"
)

func init() {
	// logger.EnableDebugLog()
}

var (
	log      = logger.Log
	logDebug = logger.Debug
	logError = logger.Error
)

// The XMLRPC type is a simple one
type XMLRPC interface {
	String() string
	IsValid() bool
}

func NewClient(url string, entityName string, samplerName string) (sampler Sampler, err error) {
	logDebug.Printf("%q, %q, %q", url, entityName, samplerName)
	c := Client{}
	c.SetURL(url)
	return c.NewSampler(entityName, samplerName)
}
