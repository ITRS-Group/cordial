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

package instance

import (
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Clean removes all the files and directories listed in the component
// type of the instance c for removal. First all entries in the
// CleanList are removed and then, if full is true, the instance is
// stopped and the entries in FullClean are removed. Any instances
// stopped are started up, but any that were already stopped will be
// left stopped.
func Clean(c geneos.Instance, full bool) (err error) {
	var stopped bool

	cleanlist := config.GetString(c.Type().CleanList)
	purgelist := config.GetString(c.Type().PurgeList)

	if !full {
		if cleanlist != "" {
			if err = RemovePaths(c, cleanlist); err == nil {
				log.Debug().Msgf("%s cleaned", c)
			}
		}
		return
	}

	if !IsRunning(c) {
		stopped = false
	} else if err = Stop(c, true, false); err != nil {
		return
	} else {
		stopped = true
	}

	if cleanlist != "" {
		if err = RemovePaths(c, cleanlist); err != nil {
			return
		}
	}
	if purgelist != "" {
		if err = RemovePaths(c, purgelist); err != nil {
			return
		}
	}
	log.Debug().Msgf("%s fully cleaned", c)
	if stopped {
		err = Start(c)
	}
	return
}
