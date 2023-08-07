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
	"errors"
	"os"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Stop an instance
func Stop(c geneos.Instance, force, kill bool) (err error) {
	if !force && IsProtected(c) {
		return geneos.ErrProtected
	}

	if !IsRunning(c) {
		log.Debug().Msgf("%s not running", c)
		return os.ErrProcessDone
	}

	// start := time.Now()

	if !kill {
		if err = Signal(c, syscall.SIGTERM); err == os.ErrProcessDone {
			return nil
		}

		if errors.Is(err, syscall.EPERM) {
			return os.ErrPermission
		}

		for i := 0; i < 10; i++ {
			time.Sleep(250 * time.Millisecond)
			if err = Signal(c, syscall.SIGTERM); err == os.ErrProcessDone {
				return nil
			}
		}

		if !IsRunning(c) {
			return nil
		}
	}

	if err = Signal(c, syscall.SIGKILL); err == os.ErrProcessDone {
		return nil
	}

	time.Sleep(250 * time.Millisecond)
	if !IsRunning(c) {
		return nil
	}
	return
}
