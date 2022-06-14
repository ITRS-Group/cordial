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
	"time"
)

// WriteMessage is the only function for a Stream that is data oriented.
// The others are administrative.
func (s Sampler) WriteMessage(streamname string, message string) (err error) {
	return s.addMessageStream(s.EntityName(), s.SamplerName(), streamname, message)
}

func (s Sampler) SignOnStream(streamname string, heartbeat time.Duration) error {
	return s.signOnStream(s.EntityName(), s.SamplerName(), streamname, int(heartbeat.Seconds()))
}

func (s Sampler) SignOffStream(streamname string) error {
	return s.signOffStream(s.EntityName(), s.SamplerName(), streamname)
}

func (s Sampler) HeartbeatStream(streamname string) error {
	return s.heartbeatStream(s.EntityName(), s.SamplerName(), streamname)
}
