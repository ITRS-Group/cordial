/*
Copyright © 2023 ITRS Group

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

package ca3

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

func getPID(i interface{}) (pid int, err error) {
	switch c := i.(type) {
	case *CA3s:
		var pids []int

		// safe to ignore error as it can only be bad pattern,
		// which means no matches to range over
		dirs, _ := c.Host().Glob("/proc/[0-9]*")

		for _, dir := range dirs {
			p, _ := strconv.Atoi(path.Base(dir))
			pids = append(pids, p)
		}

		sort.Ints(pids)

		var data []byte
		for _, pid = range pids {
			var jarOK, configOK bool
			if data, err = c.Host().ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err != nil {
				// process may disappear by this point, ignore error
				continue
			}
			args := bytes.Split(data, []byte("\000"))
			execfile := path.Base(string(args[0]))
			if execfile != "java" {
				continue
			}
			for _, arg := range args[1:] {
				if strings.Contains(string(arg), "collection-agent") {
					jarOK = true
				}
				if strings.Contains(string(arg), c.Config().GetString("config")) {
					configOK = true
				}
				if jarOK && configOK {
					return
				}
			}
		}
	default:
		return 0, os.ErrProcessDone
	}
	return 0, os.ErrProcessDone
}
