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

package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// ErrNotInteractive is returned when input is required and STDIN is a
// named pipe
var ErrNotInteractive = errors.New("not an interactive session")

// ReadUserInput reads input from Stdin and returns the input unless
// there is an error. The prompt is made up from format and args (passed
// to fmt.Sprintf) and then shown to the user as-is. If STDIN is a named
// pipe (and not interactive) then a syscall.ENOTTY is returned.
func ReadUserInput(format string, args ...any) (input string, err error) {
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
// On error the pw is empty and does not need to be Destory()ed.
//
// If STDIN is a named pipe (not interactive) a syscall.ENOTTY is
// returned as an error
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
			fmt.Println("Passwords do not match")
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
