/*
Copyright © 2026 ITRS Group

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

package ims

import (
	"runtime"
	"sync"
	"weak"

	"github.com/itrs-group/cordial/pkg/config"
)

type sectionKey struct {
	cf   weak.Pointer[config.Config]
	name string
}

var sections sync.Map // sectionKey -> *config.Config

// Section returns the configuration for the named section of cf, like
// [config.Config.Sub], but only creates it the first time it is asked
// for. The configuration of a server is not changed once it is in use,
// as a reload is a new configuration, so a section is the same for the
// life of its parent, which avoids copying it, and holding the
// configuration lock, for every request.
func Section(cf *config.Config, name string) *config.Config {
	key := sectionKey{weak.Make(cf), name}
	if sub, ok := sections.Load(key); ok {
		return sub.(*config.Config)
	}

	sub, loaded := sections.LoadOrStore(key, cf.Sub(name))
	if !loaded {
		// the section does not refer to cf, so cf can be collected
		runtime.AddCleanup(cf, func(k sectionKey) { sections.Delete(k) }, key)
	}
	return sub.(*config.Config)
}
