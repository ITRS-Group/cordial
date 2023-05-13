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
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/awnumar/memguard"
	"golang.org/x/term"
)

// ReadUserInput reads input from Stdin and returns the input unless
// there is an error. The prompt is shown to the user as-is.
func ReadUserInput(prompt string) (input string, err error) {
	var oldState *term.State
	if oldState, err = term.MakeRaw(int(os.Stdin.Fd())); err != nil {
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	t := term.NewTerminal(os.Stdin, prompt)
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
func ReadPasswordInput(match bool, maxtries int, prompt ...string) (pw *memguard.Enclave, err error) {
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
			pwt, err = term.ReadPassword(syscall.Stdin)
			pw1 := memguard.NewEnclave(pwt)
			fmt.Println() // always move to new line even on error
			if err != nil {
				return
			}
			if len(prompt) < 2 {
				fmt.Printf("Re-enter Password: ")
			} else {
				fmt.Printf("%s: ", prompt[1])
			}
			pwt, err = term.ReadPassword(syscall.Stdin)
			pw2 := memguard.NewEnclave(pwt)
			fmt.Println() // always move to new line even on error
			if err != nil {
				return
			}
			fmt.Println() // always move to new line even on error

			pw1b, _ := pw1.Open()
			pw2b, _ := pw2.Open()
			if pw1b.EqualTo(pw2b.Bytes()) {
				pw2b.Destroy()
				pw = pw1
				matched = true
				break
			}
			pw1b.Destroy()
			pw2b.Destroy()
			fmt.Println("Passwords do not match. Please try again.")
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
		pwt, err = term.ReadPassword(syscall.Stdin)
		pw = memguard.NewEnclave(pwt)
		fmt.Println() // always move to new line even on error
		if err != nil {
			return
		}
		fmt.Println() // always move to new line even on error
		if err != nil {
			return
		}
	}

	return
}
