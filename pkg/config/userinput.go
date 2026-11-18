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
	"bytes"
	"errors"
	"fmt"
	"io"
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

// ReadPEM reads and returns a PEM formatted input (without
// validation) from one of these sources:
//
//   - If `from` is empty then an empty string is returned
//   - If `from` is a dash (`-`) then data is read from STDIN the after
//     the user is prompted, on STDERR, with `Paste PEM formatted
//     [PROMPT], end with newline + CTRL-D:` where `[PROMPT]` is taken
//     from the prompt argument.
//   - If `from` has the prefix `pem:` then the data is taken from the
//     remainder of the argument.
//   - If `from` has the prefix `http://` or `https://` then the data
//     is fetched from the URL pointed to by `from` using an HTTP GET
//     request, which times out after 30 seconds.
//   - Otherwise the file at the path pointed to by `from` is read and
//     returned
//
// Any error when reading the input is returned.
func ReadPEM(from, prompt string) (data []byte, err error) {
	switch {
	case from == "":
		return
	case strings.HasPrefix(from, "pem:"):
		data = []byte(strings.TrimPrefix(from, "pem:"))
		return
	case strings.HasPrefix(from, "http://"), strings.HasPrefix(from, "https://"):
		resp, err := fetchClient.Get(from)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode > 299 {
			return nil, fmt.Errorf("error fetching %s: %s", from, resp.Status)
		}
		return io.ReadAll(resp.Body)
	case from == "-":
		isterm := term.IsTerminal(int(os.Stdin.Fd()))
		if isterm {
			// only prompt if STDIN is a terminal session
			fmt.Fprintf(os.Stderr, "Paste PEM formatted %s, end with newline + CTRL-D:\n", prompt)
		}
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return data, err
		}
		if isterm {
			fmt.Fprintln(os.Stderr)
		}
	default:
		return os.ReadFile(ResolveHome(from))
	}

	return
}

// ReadPasswordInput prompts the user for a password without echoing the
// input. This is returned as a Secret. If match is true then the user
// is prompted twice and the two instances checked for a match. Up to
// maxtries attempts are allowed after which an error is returned. If
// maxtries is 0 then a default of 3 attempts is set.
//
// If prompt is given then it must either be one or two strings,
// depending on match set. The prompt(s) are suffixed with ": " in both
// cases. The defaults are "Password" and "Re-enter Password". The
// prompts and any messages are written to STDERR, not STDOUT.
//
// On error the pw is empty and does not need to be Destroy()ed. Any
// other passwords read, such as ones that did not match, are cleared.
//
// If STDIN is not a terminal then config.ErrNotInteractive is returned.
//
// The returned Secret should be cleared by the caller when it is no
// longer needed, typically by calling clear() on the Secret which will
// zero the memory used by the Secret.
func ReadPasswordInput(match bool, maxtries int, prompt ...string) (secret Secret, err error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		err = ErrNotInteractive
		return
	}

	// prompts and messages go to STDERR so that STDOUT can be used for
	// the results
	readOne := func(prompt string) (Secret, error) {
		fmt.Fprintf(os.Stderr, "%s: ", prompt)
		pw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // always move to new line even on error
		return Secret(pw), err
	}

	if !match {
		p := "Password"
		if len(prompt) > 0 {
			p = strings.Join(prompt, " ")
		}
		secret, err = readOne(p)
		if err != nil {
			clear(secret)
			return nil, err
		}
		return
	}

	if len(prompt) != 2 {
		prompt = []string{"Password", "Re-enter Password"}
	}

	if maxtries == 0 {
		maxtries = 3
	}

	for range maxtries {
		pw1, err := readOne(prompt[0])
		if err != nil {
			clear(pw1)
			return nil, err
		}
		pw2, err := readOne(prompt[1])
		if err != nil {
			clear(pw1)
			clear(pw2)
			return nil, err
		}

		switch {
		case len(pw1) == 0 || len(pw2) == 0:
			fmt.Fprintln(os.Stderr, "Invalid password(s)")
		case bytes.Equal(pw1, pw2):
			clear(pw2)
			return pw1, nil
		default:
			fmt.Fprintln(os.Stderr, "Entries do not match")
		}
		clear(pw1)
		clear(pw2)
	}

	return nil, fmt.Errorf("too many attempts, giving up")
}
