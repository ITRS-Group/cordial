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
	"time"
)

// WriteMessage is the only function for a Stream that is data oriented.
// The others are administrative.
func (s Sampler) WriteMessage(streamname string, message string) (err error) {
	return s.addMessageStream(s.entityName, s.samplerName, streamname, message)
}

func (s Sampler) SignOnStream(streamname string, heartbeat time.Duration) error {
	return s.signOnStream(s.entityName, s.samplerName, streamname, int(heartbeat.Seconds()))
}

func (s Sampler) SignOffStream(streamname string) error {
	return s.signOffStream(s.entityName, s.samplerName, streamname)
}

func (s Sampler) HeartbeatStream(streamname string) error {
	return s.heartbeatStream(s.entityName, s.samplerName, streamname)
}
