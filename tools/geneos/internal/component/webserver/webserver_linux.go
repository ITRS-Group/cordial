/*
Copyright © 2023 ITRS Group

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

package webserver

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

func webserverGetPID(i interface{}) (pid int, err error) {
	switch w := i.(type) {
	case *Webservers:
		var pids []int

		// safe to ignore error as it can only be bad pattern,
		// which means no matches to range over
		dirs, _ := w.Host().Glob("/proc/[0-9]*")

		for _, dir := range dirs {
			p, _ := strconv.Atoi(path.Base(dir))
			pids = append(pids, p)
		}

		sort.Ints(pids)

		var data []byte
		for _, pid = range pids {
			var wdOK, jarOK bool
			if data, err = w.Host().ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err != nil {
				// process may disappear by this point, ignore error
				continue
			}
			args := bytes.Split(data, []byte("\000"))
			execfile := path.Base(string(args[0]))
			if execfile != "java" {
				continue
			}
			for _, arg := range args[1:] {
				if string(arg) == "-Dworking.directory="+w.Home() {
					wdOK = true
				}
				if strings.HasSuffix(string(arg), "geneos-web-server.jar") {
					jarOK = true
				}
				if wdOK && jarOK {
					return
				}
			}
		}
	default:
		return 0, os.ErrProcessDone
	}
	return 0, os.ErrProcessDone
}
