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

package plugins

import (
	"net/url"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/xmlrpc"
)

// all Plugins must implement these methods
type Plugins interface {
	SetInterval(time.Duration)
	Interval() time.Duration
	Start(*sync.WaitGroup) error
	Close() error
}

type Connection struct {
	xmlrpc.Sampler
}

func Open(url *url.URL, entityName string, samplerName string) (s *Connection, err error) {
	sampler, err := xmlrpc.NewClient(url).Sampler(entityName, samplerName)
	s = &Connection{sampler}
	return
}
