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

package host

import (
	"bufio"
	"bytes"
	"fmt"
	"runtime"
	"strings"
)

func (h *Host) SetOSReleaseEnv() (err error) {
	osinfo := make(map[string]string)
	switch runtime.GOOS {
	case "windows":
		// XXX simulate values?
	default:
		f, err := h.ReadFile("/etc/os-release")
		if err != nil {
			if f, err = h.ReadFile("/usr/lib/os-release"); err != nil {
				return fmt.Errorf("cannot open /etc/os-release or /usr/lib/os-release")
			}
		}

		releaseFile := bytes.NewBuffer(f)
		scanner := bufio.NewScanner(releaseFile)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) == 0 || strings.HasPrefix(line, "#") {
				continue
			}
			s := strings.SplitN(line, "=", 2)
			if len(s) != 2 {
				return ErrInvalidArgs
			}
			key, value := s[0], s[1]
			value = strings.Trim(value, "\"")
			osinfo[key] = value
		}
	}
	h.Set("osinfo", osinfo)
	return
}
