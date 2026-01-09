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

package config

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/term"
)

// ErrNotInteractive is returned when input is required and STDIN is a
// named pipe
var ErrNotInteractive = errors.New("not an interactive session")

// ReadUserInputLine reads input from Stdin and returns the line unless
// there is an error. The prompt is made up from format and args (passed
// to fmt.Sprintf) and then shown to the user as-is. If STDIN is a named
// pipe (and not interactive) then a syscall.ENOTTY is returned.
func ReadUserInputLine(format string, args ...any) (input string, err error) {
	var oldState *term.State
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		err = ErrNotInteractive
		return
	}
	if oldState, err = term.MakeRaw(int(os.Stdin.Fd())); err != nil {
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	t := term.NewTerminal(os.Stdin, fmt.Sprintf(format, args...))
	return t.ReadLine()
}

// ReadPEMString reads and returns a PEM formatted input (without
// validation) from one of these sources:
//
//   - If `from` is empty then an empty string is returned
//   - If `from` is a dash (`-`) then data is read from STDIN the after
//     the user is prompted with `Paste PEM formatted [PROMPT], end
//     with newline + CTRL-D:` where `[PROMPT]` is taken from
//     the prompt argument.
//   - If `from` has the prefix `pem:` then the data is taken from the
//     remainder of the argument.
//   - If `from` has the prefix `http://` or `https://` then the data
//     is fetched from the URL pointed to by `from` using an HTTP GET
//     request.
//   - Otherwise the file at the path pointed to by `from` is read and
//     returned
//
// Any error when reading the input is returned.
func ReadPEMString(from, prompt string) (data string, err error) {
	b, err := ReadPEMBytes(from, prompt)
	if err != nil {
		return data, err
	}
	data = string(b)
	return
}

// ReadPEMBytes reads and returns a PEM formatted input (without
// validation) from one of these sources:
//
//   - If `from` is empty then an empty string is returned
//   - If `from` is a dash (`-`) then data is read from STDIN the after
//     the user is prompted with `Paste PEM formatted [PROMPT], end
//     with newline + CTRL-D:` where `[PROMPT]` is taken from
//     the prompt argument.
//   - If `from` has the prefix `pem:` then the data is taken from the
//     remainder of the argument.
//   - If `from` has the prefix `http://` or `https://` then the data
//     is fetched from the URL pointed to by `from` using an HTTP GET
//     request.
//   - Otherwise the file at the path pointed to by `from` is read and
//     returned
//
// Any error when reading the input is returned.
func ReadPEMBytes(from, prompt string) (data []byte, err error) {
	if from == "" {
		return
	}
	switch {
	case from == "":
		break
	case strings.HasPrefix(from, "pem:"):
		data = []byte(strings.TrimPrefix(from, "pem:"))
		return
	case strings.HasPrefix(from, "http://"), strings.HasPrefix(from, "https://"):
		resp, err2 := http.Get(from)
		if err != nil {
			err = err2
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode > 299 {
			err = fmt.Errorf("error fetching %s: %s", from, resp.Status)
			return
		}
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return
		}
	case from == "-":
		fmt.Printf("Paste PEM formatted %s, end with newline + CTRL-D:\n", prompt)
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return data, err
		}
		fmt.Println()
	default:
		data, err = os.ReadFile(from)
		if err != nil {
			return data, err
		}
	}

	return
}

// ReadPasswordInput prompts the user for a password without echoing the input.
// This is returned as a memguard LockBuffer. If match is true then the user is
// prompted twice and the two instances checked for a match. Up to maxtries
// attempts are allowed after which an error is returned. If maxtries is 0 then
// a default of 3 attempts is set.
//
// If prompt is given then it must either be one or two strings, depending on
// match set. The prompt(s) are suffixed with ": " in both cases. The defaults
// are "Password" and "Re-enter Password".
//
// On error the pw is empty and does not need to be Destroy()ed.
//
// If STDIN is not a terminal then config.ErrNotInteractive is returned.
func ReadPasswordInput(match bool, maxtries int, prompt ...string) (plaintext *Plaintext, err error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		err = ErrNotInteractive
		return
	}
	if match {
		var matched bool
		if len(prompt) != 2 {
			prompt = []string{}
		}

		if maxtries == 0 {
			maxtries = 3
		}

		for i := 0; i < maxtries; i++ {
			var pwt []byte
			if len(prompt) == 0 {
				fmt.Printf("Password: ")
			} else {
				fmt.Printf("%s: ", prompt[0])
			}
			pwt, err = term.ReadPassword(int(os.Stdin.Fd()))
			pw1 := NewPlaintext(pwt)
			fmt.Println() // always move to new line even on error
			if err != nil {
				return
			}
			if len(prompt) < 2 {
				fmt.Printf("Re-enter Password: ")
			} else {
				fmt.Printf("%s: ", prompt[1])
			}
			pwt, err = term.ReadPassword(int(os.Stdin.Fd()))
			pw2 := NewPlaintext(pwt)
			fmt.Println() // always move to new line even on error
			if err != nil {
				return
			}

			if pw1.IsNil() || pw2.IsNil() {
				fmt.Println("Invalid password(s)")
				continue
			}
			pw1b, _ := pw1.Open()
			pw2b, _ := pw2.Open()
			if pw1b.EqualTo(pw2b.Bytes()) {
				matched = true
			}
			pw1b.Destroy()
			pw2b.Destroy()

			if matched {
				plaintext = pw1
				break
			}
			fmt.Println("Entries do not match")
		}
		if !matched {
			err = fmt.Errorf("too many attempts, giving up")
			return
		}
	} else {
		var pwt []byte
		if len(prompt) == 0 {
			fmt.Printf("Password: ")
		} else {
			fmt.Printf("%s: ", strings.Join(prompt, " "))
		}
		pwt, err = term.ReadPassword(int(os.Stdin.Fd()))
		plaintext = NewPlaintext(pwt)
		fmt.Println() // always move to new line even on error
		if err != nil {
			return
		}
	}

	return
}
