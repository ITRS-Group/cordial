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
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func GetPID(c geneos.Instance) (pid int, err error) {
	return
}

func GetPIDInfo(c geneos.Instance) (pid int, uid uint32, gid uint32, mtime time.Time, err error) {
	return
}

func ListeningPorts(c geneos.Instance) (ports []int) {
	return
}

func AllListeningPorts(h *geneos.Host, min, max int) (ports []int) {
	return
}

func Files(c geneos.Instance) (links map[int]string) {
	return
}

func Sockets(c geneos.Instance) (links map[int]string) {
	return
}
